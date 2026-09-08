//go:build !linux && !windows

package core

import (
	"os"
	"runtime"
)

func platformNetworkCapabilities() NetworkCapabilities {
	tun, reason := privilegedTUNCapability(os.Geteuid() == 0, nil)
	if runtime.GOOS != "darwin" {
		tun, reason = false, NetworkTUNUnsupported
	}
	caps := NetworkCapabilities{TUNAvailable: tun, TUNReason: reason}
	if err := probeIPv6Socket(); err != nil {
		caps.IPv6Reason = NetworkIPv6Unknown
		return caps
	}
	caps.IPv6Available = true
	return caps
}
