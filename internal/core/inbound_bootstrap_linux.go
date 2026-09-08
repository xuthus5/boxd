//go:build linux

package core

func bootstrapTUNAvailable() bool {
	available, _ := linuxTUNCapability(linuxNetworkAdmin, openLinuxTUN)
	return available
}
