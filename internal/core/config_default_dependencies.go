package core

import "errors"

// PrepareDNSDefaults 在同一次配置应用内补齐 DNS 必需的安全代理出站。
// 已有但不安全的代理策略应由用户修正，不能静默切换至直连。
func PrepareDNSDefaults(
	cfg map[string]any,
	dnsInstaller DNSDefaultsInstaller,
	outboundInstaller OutboundDefaultsInstaller,
) (*DNSDefaultsResult, error) {
	result, err := dnsInstaller.Install(cfg)
	if !errors.Is(err, ErrDNSProxyRequired) {
		if err == nil {
			ensureDefaultProxyFinal(cfg)
		}
		return result, err
	}
	if outboundInstaller == nil {
		outboundInstaller = NewDefaultOutboundsInstaller()
	}
	outbounds, err := outboundInstaller.Install(cfg)
	if err != nil {
		return nil, err
	}
	cfg["outbounds"] = outbounds.Outbounds
	ensureDefaultProxyFinal(cfg)
	return dnsInstaller.Install(cfg)
}

// EnsureDefaultDNSForRouting 避免新路由规则在没有 DNS 配置时隐式使用系统解析器。
func EnsureDefaultDNSForRouting(cfg map[string]any) error {
	dns, _ := cfg["dns"].(map[string]any)
	servers, _ := dns["servers"].([]any)
	if len(servers) > 0 {
		ensureDefaultProxyFinal(cfg)
		return nil
	}
	result, err := PrepareDNSDefaults(cfg, NewDefaultDNSInstaller(), NewDefaultOutboundsInstaller())
	if err != nil {
		return err
	}
	cfg["dns"] = result.DNS
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	route["default_domain_resolver"] = result.DefaultDomainResolver
	cfg["route"] = route
	return nil
}

func ensureDefaultProxyFinal(cfg map[string]any) {
	if existingOutbounds(cfg)["proxy"] == nil {
		return
	}
	route, _ := cfg["route"].(map[string]any)
	if route == nil {
		route = map[string]any{}
	}
	if final, _ := route["final"].(string); final == "" {
		route["final"] = "proxy"
	}
	cfg["route"] = route
}
