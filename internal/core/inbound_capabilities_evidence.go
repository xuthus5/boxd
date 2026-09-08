package core

const (
	windowsIPv6DisableNative  = 0x10
	windowsIPv6DisableTunnels = 0x01
)

// Windows 的 PreferIPv4(0x20) 不等于禁用 IPv6；隧道策略不明确时要求显式选择。
func windowsIPv6Policy(disabled uint64, err error) (bool, string) {
	if err != nil {
		return false, NetworkIPv6Unknown
	}
	if disabled&windowsIPv6DisableNative != 0 {
		return false, NetworkIPv6Disabled
	}
	if disabled&windowsIPv6DisableTunnels != 0 {
		return false, NetworkIPv6Unknown
	}
	return true, ""
}

func privilegedTUNCapability(elevated bool, err error) (bool, string) {
	if err != nil {
		return false, NetworkTUNUnknown
	}
	if !elevated {
		return false, NetworkTUNPermission
	}
	return true, ""
}
