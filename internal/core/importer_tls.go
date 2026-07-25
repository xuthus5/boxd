package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

func normalizeCertificatePublicKeySHA256(value string) ([]string, error) {
	items := splitLinkList(value)
	hashes := make([]string, 0, len(items))
	for _, item := range items {
		hash, err := normalizeCertificateHash(item)
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}
	return hashes, nil
}

func normalizeCertificateHash(value string) (string, error) {
	hexValue := strings.NewReplacer(":", "", "-", "").Replace(strings.TrimSpace(value))
	if len(hexValue) == sha256.Size*2 {
		decoded, err := hex.DecodeString(hexValue)
		if err == nil {
			return base64.StdEncoding.EncodeToString(decoded), nil
		}
	}
	decoded, err := decodeLinkBase64(value)
	if err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("invalid SHA-256 certificate pin %q", value)
	}
	return base64.StdEncoding.EncodeToString(decoded), nil
}
