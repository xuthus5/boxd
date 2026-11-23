// Package nodelink converts common share links into engine-neutral nodes.
package nodelink

import (
	"context"
	"strings"

	"github.com/xuthus5/boxd/control"
	"github.com/xuthus5/boxd/kernel"
)

// Codec implements control.NodeCodec for subscription and manual links.
type Codec struct{}

var _ control.NodeCodec = (*Codec)(nil)

// New constructs a stateless link codec.
func New() *Codec { return &Codec{} }

func (c *Codec) Decode(ctx context.Context, link string) (control.Node, error) {
	if err := ctx.Err(); err != nil {
		return control.Node{}, err
	}
	link = strings.TrimSpace(link)
	if link == "" {
		return control.Node{}, linkError("nodes.decode_link", "node link is empty")
	}
	separator := strings.Index(link, "://")
	if separator <= 0 {
		return control.Node{}, linkError("nodes.decode_link", "node link has no scheme")
	}
	scheme := strings.ToLower(link[:separator])
	body := link[separator+3:]
	switch scheme {
	case "ss":
		return decodeShadowsocks(body)
	case "vmess":
		return decodeVMess(body)
	case "vless":
		return decodeURLNode(scheme, link)
	case "trojan":
		return decodeURLNode(scheme, link)
	case "hysteria2", "hy2":
		return decodeURLNode(scheme, link)
	case "socks", "socks5":
		return decodeURLNode(scheme, link)
	case "http", "https":
		return decodeURLNode(scheme, link)
	default:
		return control.Node{}, linkError("nodes.decode_link", "unsupported node link scheme "+scheme)
	}
}

func (c *Codec) Encode(ctx context.Context, node control.Node) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(node.Protocol) == "" || strings.TrimSpace(node.Server) == "" || node.Port == 0 {
		return "", linkError("nodes.encode_link", "node protocol, server, and port are required")
	}
	switch strings.ToLower(node.Protocol) {
	case "shadowsocks", "ss":
		return encodeShadowsocks(node)
	case "vmess":
		return encodeVMess(node)
	case "vless":
		return encodeURLNode(node, "vless")
	case "trojan":
		return encodeURLNode(node, "trojan")
	case "hysteria2", "hy2":
		return encodeURLNode(node, "hysteria2")
	case "socks", "socks5":
		return encodeURLNode(node, "socks5")
	case "http":
		return encodeURLNode(node, "http")
	default:
		return "", linkError("nodes.encode_link", "unsupported node protocol "+node.Protocol)
	}
}

func linkError(op, message string) error {
	return kernel.NewError(kernel.ErrCodeInvalidConfig, op, message).
		WithHint("check the share link scheme and required fields")
}
