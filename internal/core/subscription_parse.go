package core

import (
	"encoding/json"
	"strings"

	"github.com/xuthus5/boxui/internal/model"
)

func parseSubscriptionContent(body []byte) []model.Outbound {
	// Try JSON format (sing-box outbounds)
	var singBox struct {
		Outbounds []model.Outbound `json:"outbounds"`
	}
	if err := json.Unmarshal(body, &singBox); err == nil && len(singBox.Outbounds) > 0 {
		return singBox.Outbounds
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
