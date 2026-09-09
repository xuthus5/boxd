import type { PolicyFieldSpec } from "@/features/policy/policy-form-model"

export const dns114GlobalFields: readonly PolicyFieldSpec[] = [
  { path: "timeout", label: "dnsTimeout", section: "basic" },
  { path: "optimistic", label: "optimisticCache", kind: "json-value", section: "cache" },
]

export const dns114MatchFields: readonly PolicyFieldSpec[] = [
  { path: "query_client_subnet", label: "queryClientSubnet", kind: "list", section: "basic" },
  { path: "query_dnssec", label: "queryDNSSEC", kind: "boolean", section: "basic" },
  { path: "match_response", label: "matchResponse", kind: "json-value", section: "domain" },
  { path: "response_rcode", label: "responseRCode", section: "domain" },
  { path: "response_answer", label: "responseAnswer", kind: "list", section: "domain" },
  { path: "response_ns", label: "responseNS", kind: "list", section: "domain" },
  { path: "response_extra", label: "responseExtra", kind: "list", section: "domain" },
  { path: "package_name_regex", label: "packageNameRegex", kind: "list", section: "process" },
  { path: "source_mac_address", label: "sourceMACAddress", kind: "list", section: "process" },
  { path: "source_hostname", label: "sourceHostname", kind: "list", section: "process" },
  { path: "preferred_by", label: "preferredBy", kind: "list", section: "process" },
]

export const dns114RouteOptions: readonly PolicyFieldSpec[] = [
  { path: "timeout", label: "dnsTimeout", section: "action" },
  { path: "disable_optimistic_cache", label: "disableOptimisticCache", kind: "boolean", section: "action" },
  { path: "remove_client_subnet", label: "removeClientSubnet", kind: "boolean", section: "action" },
]

export const dnsRaceField = { path: "race", label: "dnsRace", kind: "boolean", section: "action" } as const
export const dnsSpeculativeField = { path: "speculative", label: "dnsSpeculative", kind: "boolean", section: "action" } as const

export const endpointDNSFields: readonly PolicyFieldSpec[] = [
  { path: "endpoint", label: "endpoint", section: "special" },
  { path: "accept_default_resolvers", label: "acceptDefaultResolvers", kind: "boolean", section: "special" },
  { path: "accept_search_domain", label: "acceptSearchDomain", kind: "boolean", section: "special" },
]
