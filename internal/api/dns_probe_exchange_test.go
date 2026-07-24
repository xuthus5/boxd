package api

import (
	"errors"
	"testing"
)

type dnsReadError struct{}

func (dnsReadError) Read([]byte) (int, error) {
	return 0, errors.New("dns read failed")
}

func TestReadDNSMessageBodyReadError(t *testing.T) {
	if _, err := readDNSMessageBody(dnsReadError{}); err == nil {
		t.Fatal("expected reader error")
	}
}
