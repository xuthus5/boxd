//go:build linux

package core

import (
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func platformNetworkCapabilities() NetworkCapabilities {
	tun, tunReason := linuxTUNCapability(linuxNetworkAdmin, openLinuxTUN)
	ipv6, ipv6Reason := linuxIPv6Capability(os.ReadFile, probeIPv6Socket)
	return NetworkCapabilities{TUNAvailable: tun, TUNReason: tunReason, IPv6Available: ipv6, IPv6Reason: ipv6Reason}
}

func linuxNetworkAdmin() (bool, error) {
	header := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := [2]unix.CapUserData{}
	if err := unix.Capget(&header, &data[0]); err != nil {
		return false, err
	}
	return data[0].Effective&(1<<unix.CAP_NET_ADMIN) != 0, nil
}

func openLinuxTUN() (io.Closer, error) {
	return os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
}

func linuxTUNCapability(admin func() (bool, error), open func() (io.Closer, error)) (bool, string) {
	allowed, err := admin()
	if err != nil {
		return false, NetworkTUNUnknown
	}
	if !allowed {
		return false, NetworkTUNPermission
	}
	device, err := open()
	if err != nil {
		return false, NetworkTUNDevice
	}
	if err := device.Close(); err != nil {
		return false, NetworkTUNUnknown
	}
	return true, ""
}

func linuxIPv6Capability(read func(string) ([]byte, error), socket func() error) (bool, string) {
	socketErr := socket()
	if errors.Is(socketErr, unix.EAFNOSUPPORT) || errors.Is(socketErr, unix.EPROTONOSUPPORT) {
		return false, NetworkIPv6Unsupported
	}
	for _, scope := range []string{"all", "default"} {
		value, err := read("/proc/sys/net/ipv6/conf/" + scope + "/disable_ipv6")
		if err != nil {
			return false, NetworkIPv6Unknown
		}
		switch strings.TrimSpace(string(value)) {
		case "1":
			return false, NetworkIPv6Disabled
		case "0":
		default:
			return false, NetworkIPv6Unknown
		}
	}
	if socketErr != nil {
		return false, NetworkIPv6Unknown
	}
	return true, ""
}
