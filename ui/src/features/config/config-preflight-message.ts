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
  }
}
