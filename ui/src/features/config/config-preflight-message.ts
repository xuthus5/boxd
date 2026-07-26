import type { ConfigPreflightCode } from "@/features/config/config-preflight"

export function configPreflightMessageKey(code: ConfigPreflightCode): string {
  switch (code) {
    case "duplicate_tag":
      return "advanced.preflightDuplicateTag"
    case "missing_tag":
      return "advanced.preflightMissingTag"
    case "missing_outbound":
      return "advanced.preflightMissingOutbound"
    case "missing_dns_server":
      return "advanced.preflightMissingDNS"
    case "missing_rule_set":
      return "advanced.preflightMissingRuleSet"
    case "empty_group":
      return "advanced.preflightEmptyGroup"
    case "invalid_group_default":
      return "advanced.preflightInvalidGroupDefault"
    case "outbound_dependency_cycle":
      return "advanced.preflightOutboundDependencyCycle"
    case "dns_dependency_cycle":
      return "advanced.preflightDNSDependencyCycle"
    case "invalid_dns_default":
      return "advanced.preflightInvalidDNSDefault"
    case "multiple_fakeip_dns_servers":
      return "advanced.preflightMultipleFakeIPDNSServers"
  }
}
