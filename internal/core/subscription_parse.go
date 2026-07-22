package core

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/xuthus5/boxd/internal/model"
)

func parseSubscriptionContent(body []byte) []model.Outbound {
	if outbounds := parseSubscriptionPayload(body); len(outbounds) > 0 {
		return outbounds
	}
	// 常见机场返回整包 base64 的代理链接列表。
	if decoded, ok := decodeSubscriptionBody(body); ok {
		if outbounds := parseSubscriptionPayload(decoded); len(outbounds) > 0 {
			return outbounds
		}
	}
	return nil
}

func parseSubscriptionPayload(body []byte) []model.Outbound {
	// Try JSON format (sing-box outbounds)
	var singBox struct {
		Outbounds []model.Outbound `json:"outbounds"`
	}
	if err := json.Unmarshal(body, &singBox); err == nil && len(singBox.Outbounds) > 0 {
		return singBox.Outbounds
	}

	// Try Clash / Meta YAML proxies list.
	if outbounds := parseClashSubscription(body); len(outbounds) > 0 {
		return outbounds
	}

	// Try proxy links format (one URL per line)
	lines := strings.Split(string(body), "\n")
	var outbounds []model.Outbound
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		result, err := ParseProxyLink(line)
		if err != nil {
			continue
		}
		outbounds = append(outbounds, model.Outbound{
			Tag:    result.Tag,
			Type:   result.Type,
			Server: result.Server,
			Port:   result.Port,
			Raw:    result.Config,
		})
	}
	if len(outbounds) > 0 {
		return outbounds
	}
	return nil
}

func decodeSubscriptionBody(body []byte) ([]byte, bool) {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return nil, false
	}
	// 去掉空白与换行，兼容 padded / raw base64。
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, raw)
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, encoding := range decoders {
		decoded, err := encoding.DecodeString(compact)
		if err != nil || len(decoded) == 0 {
			continue
		}
		// 粗略校验：解码结果应是可打印文本（JSON 或代理链接）。
		sample := string(decoded)
		if strings.Contains(sample, "://") || strings.Contains(sample, "outbounds") || strings.Contains(sample, "proxies") || strings.Contains(sample, "{") {
			return decoded, true
		}
	}
	return nil, false
}
