package core

import "fmt"

// 入站配置错误码，HTTP 与桌面入口共用。
const (
	InboundProfileInvalid     = "invalid_inbound_profile"
	InboundProfileTUN         = "tun_unavailable"
	InboundProfileIPv6        = "ipv6_unavailable"
	InboundProfileIPv6Unknown = "ipv6_unknown"
	InboundProfileConflict    = "inbound_profile_conflict"
)

// InboundProfileError 提供稳定错误码和可操作说明。
type InboundProfileError struct {
	Code    string
	Message string
}

func (e *InboundProfileError) Error() string { return e.Message }

// InboundProfileOptions 控制显式代理模式及 TUN 地址族。
type InboundProfileOptions struct {
	Mode string `json:"mode"`
	IPv6 string `json:"ipv6"`
}

// BuildInboundProfile 生成入站配置；不修改输入或创建系统网络接口。
func BuildInboundProfile(cfg map[string]any, options InboundProfileOptions, caps NetworkCapabilities) (*InboundDefaultsResult, error) {
	options, err := normalizeInboundProfile(options)
	if err != nil {
		return nil, err
	}
	ipv6, err := profileIPv6(options, caps)
	if err != nil {
		return nil, err
	}
	existing, err := profileExistingInbounds(cfg, options.Mode)
	if err != nil {
		return nil, err
	}
	byTag, order, passthrough := indexInbounds(existing)
	profile := inboundProfile{options: options, caps: caps, ipv6: ipv6}
	if err := configureProfileMixed(byTag, &order, profile); err != nil {
		return nil, err
	}
	if err := configureProfileTUN(byTag, &order, profile); err != nil {
		return nil, err
	}
	return buildInboundDefaultsResult(byTag, order, passthrough), nil
}

type inboundProfile struct {
	options InboundProfileOptions
	caps    NetworkCapabilities
	ipv6    bool
}

func normalizeInboundProfile(options InboundProfileOptions) (InboundProfileOptions, error) {
	if options.Mode == "" {
		options.Mode = "preserve"
	}
	if options.IPv6 == "" {
		options.IPv6 = "auto"
	}
	if options.Mode != "proxy" && options.Mode != "tun" && options.Mode != "preserve" {
		return options, &InboundProfileError{Code: InboundProfileInvalid, Message: "mode must be proxy, tun, or preserve"}
	}
	if options.IPv6 != "auto" && options.IPv6 != "on" && options.IPv6 != "off" {
		return options, &InboundProfileError{Code: InboundProfileInvalid, Message: "ipv6 must be auto, on, or off"}
	}
	return options, nil
}

func profileIPv6(options InboundProfileOptions, caps NetworkCapabilities) (bool, error) {
	if options.Mode != "tun" {
		return false, nil
	}
	if !caps.TUNAvailable {
		return false, &InboundProfileError{Code: InboundProfileTUN,
			Message: "tun is unavailable (" + caps.TUNReason + "); grant network administration and TUN device access, or use proxy mode"}
	}
	if options.IPv6 == "off" {
		return false, nil
	}
	if caps.IPv6Available {
		return true, nil
	}
	if caps.IPv6Reason == NetworkIPv6Unknown || caps.IPv6Reason == "" {
		if options.IPv6 == "on" {
			return true, nil
		}
		return false, &InboundProfileError{Code: InboundProfileIPv6Unknown,
			Message: "ipv6 capability is unknown; choose on to attempt IPv6, or off to explicitly limit TUN to IPv4"}
	}
	if options.IPv6 == "on" {
		return false, &InboundProfileError{Code: InboundProfileIPv6,
			Message: "ipv6 is unavailable (" + caps.IPv6Reason + "); enable system IPv6 or choose off"}
	}
	return false, nil
}

func profileExistingInbounds(cfg map[string]any, mode string) ([]any, error) {
	value := cfg["inbounds"]
	if value == nil {
		return nil, nil
	}
	inbounds, ok := value.([]any)
	if !ok {
		return nil, &InboundProfileError{Code: InboundProfileConflict, Message: "inbounds must be an array"}
	}
	seen := make(map[string]bool)
	for _, value := range inbounds {
		inbound, _ := value.(map[string]any)
		tag, _ := inbound["tag"].(string)
		if mode != "preserve" && tag == "" && inbound["type"] == "tun" {
			return nil, &InboundProfileError{Code: InboundProfileConflict, Message: "untagged custom TUN exists; edit it explicitly before selecting a profile"}
		}
		if tag == "" {
			continue
		}
		if seen[tag] {
			return nil, &InboundProfileError{Code: InboundProfileConflict, Message: fmt.Sprintf("duplicate inbound tag %q; resolve the conflict first", tag)}
		}
		seen[tag] = true
	}
	return inbounds, nil
}
