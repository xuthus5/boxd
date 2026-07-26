package api

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/miekg/dns"
	quic "github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/http3"
)

func serveDNSQUICOnce(ctx context.Context, listener *quic.Listener) error {
	connection, err := listener.Accept(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = connection.CloseWithError(0, "test complete") }()
	stream, err := connection.AcceptStream(ctx)
	if err != nil {
		return err
	}
	request, err := readDNSQUICMessage(stream)
	if err != nil {
		return err
	}
	if request.Id != 0 {
		return fmt.Errorf("doq request id = %d", request.Id)
	}
	response := new(dns.Msg)
	response.SetReply(request)
	response.Answer = []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Name: request.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}, A: net.IPv4(1, 2, 3, 4)},
	}
	if err := writeDNSQUICMessage(stream, response); err != nil {
		return err
	}
	select {
	case <-connection.Context().Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func readDNSQUICMessage(reader io.Reader) (*dns.Msg, error) {
	var size uint16
	if err := binary.Read(reader, binary.BigEndian, &size); err != nil {
		return nil, err
	}
	wire := make([]byte, int(size))
	if _, err := io.ReadFull(reader, wire); err != nil {
		return nil, err
	}
	message := new(dns.Msg)
	if err := message.Unpack(wire); err != nil {
		return nil, err
	}
	return message, nil
}

func writeDNSQUICMessage(writer io.WriteCloser, message *dns.Msg) error {
	wire, err := message.Pack()
	if err != nil {
		return err
	}
	if err := binary.Write(writer, binary.BigEndian, uint16(len(wire))); err != nil {
		return err
	}
	if _, err := writer.Write(wire); err != nil {
		return err
	}
	return writer.Close()
}

func writeDNSHTTP3TestResponse(writer http.ResponseWriter, request *http.Request) error {
	if request.Method != http.MethodPost || request.URL.Path != "/custom-dns" || request.ProtoMajor != 3 {
		http.Error(writer, "unexpected request", http.StatusBadRequest)
		return fmt.Errorf("request = %s %s %s", request.Method, request.URL.Path, request.Proto)
	}
	wire, err := readDNSMessageBody(request.Body)
	if err != nil {
		return err
	}
	message := new(dns.Msg)
	if err := message.Unpack(wire); err != nil {
		return err
	}
	response := new(dns.Msg)
	response.SetReply(message)
	response.Answer = []dns.RR{
		&dns.A{Hdr: dns.RR_Header{Name: message.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 30}, A: net.IPv4(4, 3, 2, 1)},
	}
	responseWire, err := response.Pack()
	if err != nil {
		return err
	}
	writer.Header().Set("Content-Type", "application/dns-message")
	_, err = writer.Write(responseWire)
	return err
}

func startDNSHTTP3TestServer(t *testing.T, handler http.Handler) (int, *x509.CertPool) {
	t.Helper()
	certificate, roots := newDNSProbeTestCertificate(t)
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http3.Server{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{certificate},
			MinVersion:   tls.VersionTLS13,
			NextProtos:   []string{http3.NextProtoH3},
		},
		Handler: handler,
	}
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Serve(packetConn) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = packetConn.Close()
		if serveErr := <-serverResult; serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			t.Errorf("close HTTP/3 server: %v", serveErr)
		}
	})
	_, portText, err := net.SplitHostPort(packetConn.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	return port, roots
}

func newDNSProbeTestCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, publicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{certificateDER}, PrivateKey: privateKey, Leaf: leaf}, roots
}

func installDNSProbeTestRoots(t *testing.T, roots *x509.CertPool) {
	t.Helper()
	original := newDNSProbeTLSConfig
	t.Cleanup(func() { newDNSProbeTLSConfig = original })
	newDNSProbeTLSConfig = func(serverName string) *tls.Config {
		return &tls.Config{MinVersion: tls.VersionTLS13, ServerName: serverName, RootCAs: roots}
	}
}
