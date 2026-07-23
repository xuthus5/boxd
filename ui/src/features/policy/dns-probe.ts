import type { JsonObject } from "@/features/policy/policy-form-model"
import type { DNSProbeInput } from "@/lib/api/types"

const nonProbeable = new Set(["local", "hosts", "dhcp", "fakeip", "tailscale"])

export function isDNSServerProbeable(item: JsonObject): boolean {
  const type = typeof item.type === "string" ? item.type.toLowerCase() : ""
  if (type && nonProbeable.has(type)) return false
  if (typeof item.server === "string" && item.server.trim()) return true
  if (typeof item.address === "string" && item.address.trim()) return true
  return false
}

export function dnsProbeInput(item: JsonObject, domain?: string): DNSProbeInput | null {
  if (!isDNSServerProbeable(item)) return null
  const input: DNSProbeInput = {}
  if (typeof item.tag === "string" && item.tag) input.tag = item.tag
  if (typeof item.type === "string" && item.type) input.type = item.type
  if (typeof item.server === "string" && item.server) input.server = item.server
  if (typeof item.server_port === "number" && Number.isFinite(item.server_port)) {
    input.server_port = item.server_port
  }
  if (typeof item.address === "string" && item.address) input.address = item.address
  if (typeof item.path === "string" && item.path) input.path = item.path
  if (domain) input.domain = domain
  return input
}
