package service

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"unicode"
)

const (
	maxDNSProbeDomainLength = 253
	maxDNSProbePathLength   = 2048
	maxDNSProbeItems        = 128
	maxDNSProbeConcurrency  = 32
)

func normalizeDNSProbeServer(raw string) (string, error) {
	server := strings.TrimSpace(raw)
	if server == "" {
		return "", fmt.Errorf("server is required")
	}
	if server != raw || strings.IndexFunc(server, unicode.IsSpace) >= 0 {
		return "", fmt.Errorf("invalid server address")
	}
	if strings.ContainsAny(server, "/?#@") {
		return "", fmt.Errorf("invalid server address")
	}
	if strings.HasPrefix(server, "[") || strings.HasSuffix(server, "]") {
		if len(server) < 2 || !strings.HasPrefix(server, "[") || !strings.HasSuffix(server, "]") {
			return "", fmt.Errorf("invalid server address")
		}
		server = server[1 : len(server)-1]
	}
	if net.ParseIP(server) != nil {
		return server, nil
	}
	if strings.Contains(server, ":") || len(server) > maxDNSProbeDomainLength {
		return "", fmt.Errorf("invalid server address")
	}
	labels := strings.Split(strings.TrimSuffix(server, "."), ".")
	if len(labels) == 0 {
		return "", fmt.Errorf("invalid server address")
	}
	for _, label := range labels {
		if !validDNSProbeLabel(label) {
			return "", fmt.Errorf("invalid server address")
		}
	}
	return server, nil
}

func validDNSProbeLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for _, character := range label {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' {
			continue
		}
		return false
	}
	return true
}

func normalizeDNSProbePort(proto string, port int) (int, error) {
	if port == 0 {
		return defaultDNSPort(proto), nil
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid server port")
	}
	return port, nil
}

func parseDNSProbePort(raw string) (int, error) {
	port, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid port: %w", err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid server port")
	}
	return port, nil
}

func validateDNSProbePath(path string) error {
	if path == "" {
		return nil
	}
	if len(path) > maxDNSProbePathLength || strings.ContainsAny(path, "?#") {
		return fmt.Errorf("invalid dns probe path")
	}
	if strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return fmt.Errorf("invalid dns probe path")
	}
	return nil
}

func validateDNSProbeDomain(domain string) error {
	if domain == "" {
		return nil
	}
	if len(domain) > maxDNSProbeDomainLength || strings.ContainsAny(domain, "/?#") {
		return fmt.Errorf("invalid dns probe domain")
	}
	if strings.IndexFunc(domain, unicode.IsSpace) >= 0 || strings.IndexFunc(domain, unicode.IsControl) >= 0 {
		return fmt.Errorf("invalid dns probe domain")
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
