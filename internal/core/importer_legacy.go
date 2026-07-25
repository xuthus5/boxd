package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

type ssrLinkData struct {
	server        string
	port          int
	method        string
	password      string
	protocol      string
	protocolParam string
	obfs          string
	obfsParam     string
	remarks       string
}

func parseSSRLink(raw string) (*model.ImportResult, error) {
	payload, tag, err := extractSSRPayload(raw)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeLinkBase64(payload)
	if err != nil {
		return parseSSRAuthority(payload, tag)
	}
	data, err := decodeSSRData(string(decoded))
	if err != nil {
		return nil, fmt.Errorf("invalid ssr link: %w", err)
	}
	if tag == "" {
		tag = data.remarks
	}
	if tag == "" {
		tag = fmt.Sprintf("ssr-%s-%d", data.server, data.port)
	}
	config := map[string]any{
		"type":        "shadowsocksr",
		"server":      data.server,
		"server_port": data.port,
		"method":      data.method,
		"password":    data.password,
	}
	if data.protocol != "" {
		config["protocol"] = data.protocol
	}
	if data.protocolParam != "" {
		config["protocol_param"] = data.protocolParam
	}
	if data.obfs != "" {
		config["obfs"] = data.obfs
	}
	if data.obfsParam != "" {
		config["obfs_param"] = data.obfsParam
	}
	return &model.ImportResult{Tag: tag, Type: "shadowsocksr", Server: data.server, Port: data.port, Config: config}, nil
}

func extractSSRPayload(raw string) (string, string, error) {
	separator := strings.Index(raw, "://")
	if separator < 0 {
		return "", "", fmt.Errorf("invalid ssr link")
	}
	payload, tag, _ := strings.Cut(raw[separator+3:], "#")
	payload, err := url.PathUnescape(payload)
	if err != nil {
		return "", "", fmt.Errorf("invalid ssr payload: %w", err)
	}
	tag, err = url.QueryUnescape(tag)
	if err != nil {
		return "", "", fmt.Errorf("invalid ssr tag: %w", err)
	}
	return payload, tag, nil
}

func decodeSSRData(raw string) (ssrLinkData, error) {
	base, queryText, _ := strings.Cut(raw, "/?")
	parts := strings.SplitN(base, ":", 6)
	if len(parts) != 6 {
		return ssrLinkData{}, fmt.Errorf("invalid ssr payload fields")
	}
	server := parts[0]
	if server == "" {
		return ssrLinkData{}, fmt.Errorf("missing ssr server")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil || port < 1 || port > 65535 {
		return ssrLinkData{}, fmt.Errorf("invalid ssr server port %q", parts[1])
	}
	password := decodeSSRValue(parts[5])
	if parts[3] == "" || password == "" {
		return ssrLinkData{}, fmt.Errorf("missing ssr method or password")
	}
	query, err := url.ParseQuery(queryText)
	if err != nil {
		return ssrLinkData{}, fmt.Errorf("invalid ssr parameters: %w", err)
	}
	return ssrLinkData{
		server:        server,
		port:          port,
		method:        parts[3],
		password:      password,
		protocol:      parts[2],
		protocolParam: decodeSSRValue(query.Get("protoparam")),
		obfs:          parts[4],
		obfsParam:     decodeSSRValue(query.Get("obfsparam")),
		remarks:       decodeSSRValue(query.Get("remarks")),
	}, nil
}

func decodeSSRValue(raw string) string {
	if raw == "" {
		return ""
	}
	decoded, err := decodeLinkBase64(raw)
	if err == nil && len(decoded) > 0 {
		return string(decoded)
	}
	unescaped, err := url.QueryUnescape(raw)
	if err != nil {
		return raw
	}
	return unescaped
}

func parseSSRAuthority(raw, tag string) (*model.ImportResult, error) {
	parsed, err := url.Parse("//" + raw)
	if err != nil {
		return nil, fmt.Errorf("invalid ssr authority: %w", err)
	}
	server, port, err := parseModernLinkServer(parsed, "ssr", 0)
	if err != nil {
		return nil, err
	}
	if tag == "" {
		tag = fmt.Sprintf("ssr-%s-%d", server, port)
	}
	return &model.ImportResult{
		Tag: tag, Type: "shadowsocksr", Server: server, Port: port,
		Config: map[string]any{"type": "shadowsocksr", "server": server, "server_port": port},
	}, nil
}
