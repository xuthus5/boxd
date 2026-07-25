package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func isHysteria2Link(raw string) bool {
	scheme, _, ok := strings.Cut(raw, "://")
	return ok && (strings.EqualFold(scheme, "hysteria2") || strings.EqualFold(scheme, "hy2"))
}

func parseHysteria2RawLink(raw string) (*model.ImportResult, error) {
	parsed, parseErr := url.Parse(raw)
	if parseErr == nil {
		return parseHysteria2(parsed)
	}
	rewritten, serverPorts, err := rewriteHysteria2PortHopping(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid link: %w", err)
	}
	if len(serverPorts) == 0 {
		return nil, fmt.Errorf("invalid link: %w", parseErr)
	}
	parsed, err = url.Parse(rewritten)
	if err != nil {
		return nil, fmt.Errorf("invalid link: %w", err)
	}
	query := parsed.Query()
	if firstHysteria2Value(query, "mport", "server_ports", "server-ports") != "" {
		return nil, fmt.Errorf("invalid hysteria2 link: conflicting port hopping options")
	}
	query.Set("server_ports", strings.Join(serverPorts, ","))
	parsed.RawQuery = query.Encode()
	return parseHysteria2(parsed)
}

func rewriteHysteria2PortHopping(raw string) (string, []string, error) {
	authorityStart := strings.Index(raw, "://") + 3
	authorityEnd := len(raw)
	if offset := strings.IndexAny(raw[authorityStart:], "/?#"); offset >= 0 {
		authorityEnd = authorityStart + offset
	}
	authority := raw[authorityStart:authorityEnd]
	userInfo, hostPort := "", authority
	if separator := strings.LastIndexByte(authority, '@'); separator >= 0 {
		userInfo, hostPort = authority[:separator+1], authority[separator+1:]
	}
	host, portSpec, hasPort, err := splitHysteria2Authority(hostPort)
	if err != nil || !hasPort || !strings.ContainsAny(portSpec, "-,") {
		return "", nil, err
	}
	serverPorts, firstPort, err := normalizeHysteria2PortUnion(portSpec)
	if err != nil {
		return "", nil, err
	}
	rewrittenAuthority := userInfo + host + ":" + strconv.Itoa(firstPort)
	return raw[:authorityStart] + rewrittenAuthority + raw[authorityEnd:], serverPorts, nil
}

func splitHysteria2Authority(hostPort string) (string, string, bool, error) {
	if strings.HasPrefix(hostPort, "[") {
		closing := strings.IndexByte(hostPort, ']')
		if closing < 0 {
			return "", "", false, fmt.Errorf("invalid hysteria2 IPv6 server")
		}
		if closing+1 == len(hostPort) {
			return hostPort, "", false, nil
		}
		if hostPort[closing+1] != ':' {
			return "", "", false, fmt.Errorf("invalid hysteria2 server authority")
		}
		return hostPort[:closing+1], hostPort[closing+2:], true, nil
	}
	separator := strings.LastIndexByte(hostPort, ':')
	if separator < 0 {
		return hostPort, "", false, nil
	}
	if strings.ContainsRune(hostPort[:separator], ':') {
		return "", "", false, fmt.Errorf("IPv6 server must use brackets")
	}
	return hostPort[:separator], hostPort[separator+1:], true, nil
}

func normalizeHysteria2PortUnion(raw string) ([]string, int, error) {
	parts := strings.Split(raw, ",")
	serverPorts := make([]string, 0, len(parts))
	firstPort := 0
	for _, part := range parts {
		normalized, err := normalizeHysteria2PortRange(part)
		if err != nil {
			return nil, 0, err
		}
		serverPorts = append(serverPorts, normalized)
		if firstPort == 0 {
			firstPort, err = firstHysteria2Port(normalized)
			if err != nil {
				return nil, 0, err
			}
		}
	}
	return serverPorts, firstPort, nil
}

func firstHysteria2Port(portRange string) (int, error) {
	start, end, _ := strings.Cut(portRange, ":")
	value := firstNonEmpty(start, end)
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid server port range %q", portRange)
	}
	return port, nil
}
