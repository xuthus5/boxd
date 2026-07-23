package core

func clashShadowsocks(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "shadowsocks", "server": server, "server_port": port,
		"method": firstString(proxy, "cipher", "method"), "password": asString(proxy["password"]),
	}
	if plugin := firstString(proxy, "plugin"); plugin != "" {
		config["plugin"] = plugin
		if opts := asString(proxy["plugin-opts"]); opts != "" {
			config["plugin_opts"] = opts
		} else if optsMap, ok := proxy["plugin-opts"].(map[string]any); ok {
			config["plugin_opts"] = optsMap
		}
	}
	return config
}

func clashTrojan(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "trojan", "server": server, "server_port": port, "password": asString(proxy["password"]),
	}
	if tls := clashTLS(proxy); tls != nil {
		tls["enabled"] = true
		config["tls"] = tls
	} else {
		config["tls"] = map[string]any{"enabled": true}
	}
	if transport := clashTransport(proxy); transport != nil {
		config["transport"] = transport
	}
	return config
}

func clashVMess(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "vmess", "server": server, "server_port": port,
		"uuid": firstString(proxy, "uuid", "id"), "alter_id": asInt(proxy["alterId"]),
		"security": firstString(proxy, "cipher", "security"),
	}
	if transport := clashTransport(proxy); transport != nil {
		config["transport"] = transport
	}
	if tls := clashTLS(proxy); tls != nil {
		config["tls"] = tls
	}
	return config
}

func clashVLESS(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "vless", "server": server, "server_port": port, "uuid": firstString(proxy, "uuid", "id"),
	}
	if flow := asString(proxy["flow"]); flow != "" {
		config["flow"] = flow
	}
	if packetEncoding := firstString(proxy, "packet-encoding", "packet_encoding"); packetEncoding != "" {
		config["packet_encoding"] = packetEncoding
	}
	if transport := clashTransport(proxy); transport != nil {
		config["transport"] = transport
	}
	if tls := clashTLS(proxy); tls != nil {
		config["tls"] = tls
	}
	return config
}

func clashHysteria2(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "hysteria2", "server": server, "server_port": port,
		"password": firstString(proxy, "password", "auth"),
	}
	if up := asInt(proxy["up"]); up > 0 {
		config["up_mbps"] = up
	} else if up := firstString(proxy, "up", "up-mbps"); up != "" {
		config["up"] = up
	}
	if down := asInt(proxy["down"]); down > 0 {
		config["down_mbps"] = down
	} else if down := firstString(proxy, "down", "down-mbps"); down != "" {
		config["down"] = down
	}
	if obfs := firstString(proxy, "obfs"); obfs != "" {
		config["obfs"] = map[string]any{
			"type": obfs, "password": firstString(proxy, "obfs-password", "obfs_password"),
		}
	}
	if tls := clashTLS(proxy); tls != nil {
		config["tls"] = tls
	} else if sni := firstString(proxy, "sni", "servername"); sni != "" {
		config["tls"] = map[string]any{"enabled": true, "server_name": sni}
	}
	return config
}

func clashTUIC(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{
		"type": "tuic", "server": server, "server_port": port,
		"uuid": asString(proxy["uuid"]), "password": asString(proxy["password"]),
	}
	if cc := firstString(proxy, "congestion-controller", "congestion_control"); cc != "" {
		config["congestion_control"] = cc
	}
	if tls := clashTLS(proxy); tls != nil {
		config["tls"] = tls
	} else if sni := firstString(proxy, "sni", "servername"); sni != "" {
		config["tls"] = map[string]any{"enabled": true, "server_name": sni}
	}
	return config
}

func clashHTTP(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{"type": "http", "server": server, "server_port": port}
	if user := firstString(proxy, "username", "user"); user != "" {
		config["username"] = user
	}
	if pass := asString(proxy["password"]); pass != "" {
		config["password"] = pass
	}
	if tls := clashTLS(proxy); tls != nil {
		config["tls"] = tls
	}
	return config
}

func clashSocks(proxy map[string]any, server string, port int) map[string]any {
	config := map[string]any{"type": "socks", "server": server, "server_port": port, "version": "5"}
	if user := firstString(proxy, "username", "user"); user != "" {
		config["username"] = user
	}
	if pass := asString(proxy["password"]); pass != "" {
		config["password"] = pass
	}
	return config
}
