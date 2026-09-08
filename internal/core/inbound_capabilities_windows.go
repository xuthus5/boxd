//go:build windows

package core

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	windowsAdapterBufferSize = 15 * 1024
	windowsAdapterBufferMax  = 1024 * 1024
	windowsAdapterIPv6Flag   = 0x100
)

func platformNetworkCapabilities() NetworkCapabilities {
	elevated, err := windowsNetworkAdmin()
	tun, tunReason := privilegedTUNCapability(elevated, err)
	ipv6, ipv6Reason := windowsIPv6Capability()
	return NetworkCapabilities{TUNAvailable: tun, TUNReason: tunReason, IPv6Available: ipv6, IPv6Reason: ipv6Reason}
}

func windowsNetworkAdmin() (bool, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false, err
	}
	var elevated, length uint32
	err = windows.GetTokenInformation(token, windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &length)
	return elevated != 0, errors.Join(err, token.Close())
}

func windowsIPv6DisabledComponents() (uint64, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters`, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	value, _, readErr := key.GetIntegerValue("DisabledComponents")
	if errors.Is(readErr, registry.ErrNotExist) {
		readErr = nil
	}
	return value, errors.Join(readErr, key.Close())
}

func windowsIPv6Capability() (bool, string) {
	value, err := windowsIPv6DisabledComponents()
	if allowed, reason := windowsIPv6Policy(value, err); !allowed {
		return false, reason
	}
	if err := probeIPv6Socket(); err != nil {
		return false, NetworkIPv6Unknown
	}
	// 全局栈可用不证明 Wintun 已绑定 IPv6；首次创建设备前明确返回未知。
	return windowsTUNIPv6Binding()
}

func windowsTUNIPv6Binding() (bool, string) {
	size := uint32(windowsAdapterBufferSize)
	for size > 0 && size <= windowsAdapterBufferMax {
		buffer := make([]byte, size)
		adapters := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_INCLUDE_ALL_INTERFACES,
			0, adapters, &size)
		if errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			continue
		}
		if err != nil {
			return false, NetworkIPv6Unknown
		}
		for adapter := adapters; adapter != nil; adapter = adapter.Next {
			if windows.UTF16PtrToString(adapter.FriendlyName) != "boxd0" {
				continue
			}
			if adapter.Flags&windowsAdapterIPv6Flag == 0 {
				return false, NetworkIPv6Disabled
			}
			return true, ""
		}
		return false, NetworkIPv6Unknown
	}
	return false, NetworkIPv6Unknown
}
