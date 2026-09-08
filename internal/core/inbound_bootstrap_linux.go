//go:build linux

package core

import (
	"os"

	"golang.org/x/sys/unix"
)

func bootstrapTUNAvailable() bool {
	header := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := [2]unix.CapUserData{}
	if err := unix.Capget(&header, &data[0]); err != nil {
		return false
	}
	if data[0].Effective&(1<<unix.CAP_NET_ADMIN) == 0 {
		return false
	}
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	closeErr := file.Close()
	return closeErr == nil
}
