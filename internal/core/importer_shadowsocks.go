package core

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type shadowsocksLinkData struct {
	server   string
	port     int
	method   string
	password string
}

func parseShadowsocksLink(u *url.URL) (shadowsocksLinkData, error) {
	if u == nil {
		return shadowsocksLinkData{}, fmt.Errorf("invalid ss link")
	}
	if u.User != nil {
		return parseShadowsocksUserInfo(u)
	}
	return parseShadowsocksLegacy(u)
}

func parseShadowsocksUserInfo(u *url.URL) (shadowsocksLinkData, error) {
	method := u.User.Username()
	password, hasPassword := u.User.Password()
	if hasPassword {
		if method == "" || password == "" {
			return shadowsocksLinkData{}, fmt.Errorf("missing credentials in ss link")
		}
		return newShadowsocksLinkData(u.Hostname(), u.Port(), method, password)
	}

	decoded, err := decodeLinkBase64(method)
	if err != nil {
		return shadowsocksLinkData{}, fmt.Errorf("invalid credentials in ss link")
	}
	parsedMethod, parsedPassword, ok := splitShadowsocksCredentials(string(decoded))
	if !ok {
		return shadowsocksLinkData{}, fmt.Errorf("invalid credentials in ss link")
	}
	return newShadowsocksLinkData(u.Hostname(), u.Port(), parsedMethod, parsedPassword)
}

func parseShadowsocksLegacy(u *url.URL) (shadowsocksLinkData, error) {
	decoded, err := decodeLinkBase64(u.Hostname())
	if err != nil {
		return shadowsocksLinkData{}, fmt.Errorf("invalid credentials in ss link")
	}
	separator := strings.LastIndexByte(string(decoded), '@')
	if separator <= 0 || separator == len(decoded)-1 {
		return shadowsocksLinkData{}, fmt.Errorf("invalid credentials in ss link")
	}
	credentials := string(decoded[:separator])
	method, password, ok := splitShadowsocksCredentials(credentials)
	if !ok {
		return shadowsocksLinkData{}, fmt.Errorf("invalid credentials in ss link")
	}
	authority := string(decoded[separator+1:])
	parsed, err := url.Parse("//" + authority)
	if err != nil {
		return shadowsocksLinkData{}, fmt.Errorf("invalid server in ss link: %w", err)
	}
	return newShadowsocksLinkData(parsed.Hostname(), parsed.Port(), method, password)
}

func splitShadowsocksCredentials(value string) (string, string, bool) {
	separator := strings.IndexByte(value, ':')
	if separator <= 0 || separator == len(value)-1 {
		return "", "", false
	}
	return value[:separator], value[separator+1:], true
}

func newShadowsocksLinkData(server, portValue, method, password string) (shadowsocksLinkData, error) {
	if server == "" || method == "" || password == "" {
		return shadowsocksLinkData{}, fmt.Errorf("missing credentials or server in ss link")
	}
	port := 0
	if portValue != "" {
		parsedPort, err := strconv.Atoi(portValue)
		if err != nil || parsedPort < 1 || parsedPort > 65535 {
			return shadowsocksLinkData{}, fmt.Errorf("invalid port in ss link")
		}
		port = parsedPort
	}
	return shadowsocksLinkData{server: server, port: port, method: method, password: password}, nil
}

func shadowsocksPluginConfig(query url.Values) (string, string) {
	plugin := query.Get("plugin")
	pluginName, pluginOptions, _ := strings.Cut(plugin, ";")
	if pluginName == "" {
		return "", ""
	}
	if explicitOptions := firstNonEmpty(query.Get("plugin_opts"), query.Get("plugin-opts")); explicitOptions != "" {
		pluginOptions = explicitOptions
	}
	return pluginName, pluginOptions
}
