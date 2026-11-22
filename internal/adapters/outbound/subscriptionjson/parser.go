// Package subscriptionjson parses sing-box JSON subscriptions into the
// engine-neutral node model. It never imports sing-box itself.
package subscriptionjson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

var nonProxyOutbounds = map[string]struct{}{
	"direct":   {},
	"block":    {},
	"dns":      {},
	"selector": {},
	"urltest":  {},
}

// Parser implements control.SubscriptionParser for sing-box JSON documents and
// base64-encoded JSON documents.
type Parser struct{}

var _ control.SubscriptionParser = (*Parser)(nil)

// New constructs a stateless parser.
func New() *Parser { return &Parser{} }

// Parse implements control.SubscriptionParser.
func (p *Parser) Parse(ctx context.Context, result control.DownloadResult) ([]control.Node, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body := bytes.TrimSpace(result.Body)
	body = bytes.TrimPrefix(body, []byte{0xef, 0xbb, 0xbf})
	if len(body) == 0 {
		return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "subscription.parse", "subscription body is empty")
	}
	if body[0] != '{' && body[0] != '[' {
		decoded, err := decodeBase64(body)
		if err != nil {
			return nil, kernel.NewError(kernel.ErrCodeInvalidConfig, "subscription.parse", "subscription is neither JSON nor base64-encoded JSON").
				WithHint("first version supports sing-box JSON subscriptions")
		}
		body = bytes.TrimSpace(decoded)
	}
	outbounds, err := decodeOutbounds(body)
	if err != nil {
		return nil, err
	}
	nodes := make([]control.Node, 0, len(outbounds))
	for index, outbound := range outbounds {
		node, ok, err := nodeFromOutbound(outbound)
		if err != nil {
			return nil, kernel.WrapError(kernel.ErrCodeInvalidConfig, "subscription.parse", fmt.Sprintf("decode outbound %d", index), err)
		}
		if ok {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func decodeBase64(body []byte) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, string(body))
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, encoding := range encodings {
		decoded, err := encoding.DecodeString(compact)
		if err == nil && len(decoded) > 0 {
			return decoded, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func decodeOutbounds(body []byte) ([]map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if body[0] == '[' {
		var outbounds []map[string]any
		if err := decoder.Decode(&outbounds); err != nil {
			return nil, kernel.WrapError(kernel.ErrCodeInvalidConfig, "subscription.parse", "decode outbound array", err)
		}
		return outbounds, nil
	}
	var document struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := decoder.Decode(&document); err != nil {
		return nil, kernel.WrapError(kernel.ErrCodeInvalidConfig, "subscription.parse", "decode subscription JSON", err)
	}
	return document.Outbounds, nil
}

func nodeFromOutbound(outbound map[string]any) (control.Node, bool, error) {
	protocol, _ := outbound["type"].(string)
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol == "" {
		return control.Node{}, false, nil
	}
	if _, skip := nonProxyOutbounds[protocol]; skip {
		return control.Node{}, false, nil
	}
	server, _ := outbound["server"].(string)
	server = strings.TrimSpace(server)
	if server == "" {
		return control.Node{}, false, nil
	}
	port, err := portNumber(outbound["server_port"])
	if err != nil {
		return control.Node{}, false, err
	}
	if port == 0 {
		return control.Node{}, false, nil
	}
	name, _ := outbound["tag"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("%s %s:%d", protocol, server, port)
	}
	options := make(map[string]any, len(outbound))
	for key, value := range outbound {
		switch key {
		case "type", "tag", "server", "server_port", "detour":
			continue
		default:
			options[key] = value
		}
	}
	node := control.Node{
		Name:     name,
		Protocol: protocol,
		Server:   server,
		Port:     port,
		Options:  options,
	}
	if network, ok := outbound["network"].(string); ok && strings.EqualFold(strings.TrimSpace(network), "tcp") {
		disabled := false
		node.SupportsUDP = &disabled
	}
	return node, true, nil
}

func portNumber(value any) (uint16, error) {
	var parsed int64
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Int64()
		if err != nil {
			return 0, err
		}
		parsed = number
	case float64:
		parsed = int64(typed)
	case string:
		number, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, err
		}
		parsed = number
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("server_port has unsupported type %T", value)
	}
	if parsed < 0 || parsed > 65535 {
		return 0, fmt.Errorf("server_port %d is outside 0..65535", parsed)
	}
	return uint16(parsed), nil
}
