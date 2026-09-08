import { setupModuleIDs, type SetupStatus } from "@/lib/api/setup"

export function setupStatus(overrides: Partial<SetupStatus> = {}): SetupStatus {
  return {
    source_hash: "original-hash",
    capabilities: { platform: "linux", container: true, tun_available: false,
      tun_reason: "/dev/net/tun unavailable", ipv6_available: false, ipv6_reason: "IPv6 disabled", default_listen: "0.0.0.0" },
    modules: setupModuleIDs.map((id) => ({ id, state: "missing", count: 0, dependencies: [] })),
    listeners: [], kernel_running: false, proxy_ready: false, ...overrides,
  }
}
