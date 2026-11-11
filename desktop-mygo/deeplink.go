package main

import (
	"errors"
	"net/url"
	"strings"
)

func parseDeepLink(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.New("invalid deep link")
	}
	if !strings.EqualFold(parsed.Scheme, "boxd") {
		return "", errors.New("unsupported deep link scheme")
	}
	action := strings.Trim(parsed.Host, "/")
	if action == "" {
		action = strings.Trim(parsed.Path, "/")
	}
	if action != "import" {
		return "", nil
	}
	link := strings.TrimSpace(parsed.Query().Get("link"))
	if link == "" {
		return "", errors.New("deep link is missing the link parameter")
	}
	return link, nil
}
