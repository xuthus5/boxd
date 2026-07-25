package core

import (
	"strings"
	"testing"
)

func TestNormalizeCertificatePublicKeySHA256(t *testing.T) {
	hexPin := strings.Repeat("00:", 31) + "00"
	hashes, err := normalizeCertificatePublicKeySHA256(hexPin + "," + testSHA256PinBase64Alt)
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 || hashes[0] != testSHA256PinBase64 || hashes[1] != testSHA256PinBase64Alt {
		t.Fatalf("hashes = %#v", hashes)
	}
	if _, err := normalizeCertificatePublicKeySHA256("deadbeef"); err == nil {
		t.Fatal("short certificate pin unexpectedly succeeded")
	}
}
