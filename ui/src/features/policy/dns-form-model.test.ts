import { describe, expect, it } from "vitest"

import {
  applyDNSFakeIPFieldChange,
  applyDNSGlobalFieldChange,
  applyDNSRuleFieldChange,
  applyDNSServerFieldChange,
  changeDNSAction,
  changeDNSRuleType,
  changeDNSServerType,
  dnsActionFields,
  dnsActions,
  dnsGlobalFields,
  dnsRuleMatchFields,
  dnsRules,
  dnsServerFields,
  dnsServers,
  dnsServerTypes,
  inferDNSServerType,
  isDNSRuleComplete,
  legacyFakeIPFields,
  setDNSRules,
  setDNSServers,
  summarizeDNSRule,
  summarizeDNSServer,
} from "@/features/policy/dns-form-model"

const paths = (fields: readonly { path: string }[]) => fields.map((field) => field.path)

describe("DNS form metadata", () => {
  it("models DNS globals and legacy FakeIP fields", () => {
    expect(paths(dnsGlobalFields)).toEqual([
      "final", "strategy", "client_subnet", "disable_cache", "disable_expire", "independent_cache",
      "cache_capacity", "reverse_mapping",
    ])
    expect(dnsGlobalFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "final", kind: "ref", ref: "dns-server" }),
      expect.objectContaining({ path: "strategy", kind: "select" }),
      expect.objectContaining({ path: "disable_cache", kind: "boolean" }),
      expect.objectContaining({ path: "cache_capacity", kind: "number" }),
      expect.objectContaining({ path: "reverse_mapping", kind: "boolean" }),
    ]))
    expect(legacyFakeIPFields).toEqual([
      expect.objectContaining({ path: "fakeip.enabled", kind: "boolean" }),
      expect.objectContaining({ path: "fakeip.inet4_range", when: { path: "fakeip.enabled", is: true } }),
      expect.objectContaining({ path: "fakeip.inet6_range", when: { path: "fakeip.enabled", is: true } }),
    ])
  })

  it("models all approved server types", () => {
    expect(dnsServerTypes).toEqual([
      "legacy", "local", "hosts", "udp", "tcp", "tls", "quic", "https", "h3", "dhcp", "fakeip",
    ])
  })

  it("models default DNS rule matches with their JSON kinds", () => {
    expect(paths(dnsRuleMatchFields)).toEqual([
      "type", "inbound", "ip_version", "query_type", "network", "auth_user", "protocol",
      "domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr",
      "source_ip_is_private", "ip_cidr", "ip_is_private", "ip_accept_any", "source_port", "source_port_range",
      "port", "port_range", "process_name", "process_path", "process_path_regex", "package_name",
      "user", "user_id", "outbound", "clash_mode", "rule_set", "rule_set_ip_cidr_match_source",
      "network_type", "network_is_expensive", "network_is_constrained", "interface_address",
      "network_interface_address", "default_interface_address", "wifi_ssid", "wifi_bssid",
      "rule_set_ip_cidr_accept_empty", "invert",
    ])
    expect(dnsRuleMatchFields).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: "query_type", kind: "list" }),
      expect.objectContaining({ path: "source_port", kind: "number-list" }),
      expect.objectContaining({ path: "user_id", kind: "number-list" }),
      expect.objectContaining({ path: "network_is_expensive", kind: "boolean" }),
      expect.objectContaining({ path: "interface_address", kind: "json-object" }),
      expect.objectContaining({ path: "network_interface_address", kind: "json-object" }),
      expect.objectContaining({ path: "default_interface_address", kind: "list" }),
      expect.objectContaining({ path: "ip_accept_any", kind: "boolean" }),
      expect.objectContaining({ path: "rule_set_ip_cidr_accept_empty", kind: "boolean" }),
    ]))
  })

  it("models all DNS actions and action-specific JSON kinds", () => {
    expect(dnsActions).toEqual(["route", "route-options", "reject", "predefined"])
    expect(paths(dnsActionFields.route)).toEqual([
      "server", "strategy", "disable_cache", "rewrite_ttl", "client_subnet",
    ])
    expect(paths(dnsActionFields["route-options"])).not.toContain("server")
    expect(dnsActionFields.reject).toEqual([
      expect.objectContaining({ path: "method", kind: "select" }),
      expect.objectContaining({ path: "no_drop", kind: "boolean" }),
    ])
    expect(dnsActionFields.predefined).toEqual([
      expect.objectContaining({ path: "rcode" }),
      expect.objectContaining({ path: "answer", kind: "list" }),
      expect.objectContaining({ path: "ns", kind: "list" }),
      expect.objectContaining({ path: "extra", kind: "list" }),
    ])
  })
})

describe("DNS server transitions", () => {
  it("infers legacy, modern, and unknown server types without migration", () => {
    expect(inferDNSServerType({ address: "https://dns.google/dns-query" })).toBe("legacy")
    expect(inferDNSServerType({ type: "https", server: "dns.google" })).toBe("https")
    expect(inferDNSServerType({ type: "custom", payload: true })).toBe("custom")
    expect(inferDNSServerType({})).toBe("legacy")
  })

  it("keeps same-type legacy and modern objects by identity", () => {
    const legacy = { address: "local", detour: "direct", custom: "keep" }
    const https = { type: "https", server: "dns.google", custom: "keep" }
    expect(changeDNSServerType(legacy, "legacy")).toBe(legacy)
    expect(changeDNSServerType(https, "https")).toBe(https)
  })

  it("converts legacy only after an explicit Select change", () => {
    expect(changeDNSServerType({ address: "local", detour: "direct", custom: "keep" }, "udp"))
      .toEqual({ type: "udp", custom: "keep" })
  })

  it("preserves compatible modern and nested TLS fields", () => {
    expect(changeDNSServerType({
      type: "https", server: "dns.google", path: "/dns-query", headers: { X: "1" },
      tls: { enabled: true, custom: "keep" }, custom: "keep",
    }, "tls")).toEqual({
      type: "tls", server: "dns.google", tls: { enabled: true, custom: "keep" }, custom: "keep",
    })
  })

  it("retains target-compatible fields from unknown sources and validates JSON kinds", () => {
    expect(changeDNSServerType({
      type: "custom", server: "dns.google", server_port: 443, path: "/dns-query",
      headers: { X: "1" }, network_type: ["wifi"], fallback_network_type: [1],
      tls: { enabled: true, alpn: ["h2", 3], custom: "keep" }, address: "old",
      inet4_range: "198.18.0.0/15", routing_mark: 123, payload: { keep: true },
    }, "https")).toEqual({
      type: "https", server: "dns.google", server_port: 443, path: "/dns-query",
      headers: { X: "1" }, network_type: ["wifi"], routing_mark: 123,
      tls: { enabled: true, custom: "keep" },
      payload: { keep: true },
    })
  })

  it("retains compatible legacy fields from an unknown source", () => {
    expect(changeDNSServerType({
      type: "custom", address: "local", detour: "direct", client_subnet: "192.0.2.0/24",
      server: "old", payload: "keep",
    }, "legacy")).toEqual({
      address: "local", detour: "direct", client_subnet: "192.0.2.0/24", payload: "keep",
    })
  })

  it.each([
    ["legacy", { address: "local" }], ["local", { prefer_go: true }],
    ["hosts", { path: "/etc/hosts" }], ["udp", { server: "1.1.1.1" }],
    ["tcp", { server_port: 53 }], ["tls", { tls: { enabled: true } }],
    ["quic", { tls: { server_name: "dns.example" } }], ["https", { path: "/dns-query" }],
    ["h3", { headers: { X: "1" } }], ["dhcp", { interface: "eth0" }],
    ["fakeip", { inet4_range: "198.18.0.0/15" }],
  ] as const)("cleans known %s fields when entering an unknown type", (type, fields) => {
    expect(changeDNSServerType({ type, tag: "dns", ...fields, payload: "keep" }, "custom"))
      .toEqual({ type: "custom", tag: "dns", payload: "keep" })
  })
})

describe("DNS rule transitions", () => {
  it("moves between default and logical rules with target-driven cleanup", () => {
    expect(changeDNSRuleType({ domain: ["example.com"], invert: true, custom: "keep" }, "logical"))
      .toEqual({ type: "logical", invert: true, custom: "keep" })
    expect(changeDNSRuleType({ type: "logical", mode: "and", rules: [], invert: true, custom: "keep" }, "default"))
      .toEqual({ invert: true, custom: "keep" })
  })

  it("preserves same and unknown rule types by identity", () => {
    const logical = { type: "logical", mode: "or", rules: [] }
    const custom = { type: "custom", payload: { enabled: true } }
    expect(changeDNSRuleType(logical, "logical")).toBe(logical)
    expect(changeDNSRuleType(custom, "custom")).toBe(custom)
  })

  it("retains valid target fields from unknown sources and removes stale or invalid known fields", () => {
    expect(changeDNSRuleType({
      type: "custom", domain: ["example.com"], network: "tcp", source_port: [53],
      query_type: [1, "A"], wifi_ssid: [3], mode: "and", rules: [], payload: "keep",
    }, "default")).toEqual({
      domain: ["example.com"], network: "tcp", source_port: [53], query_type: [1, "A"], payload: "keep",
    })
  })

  it("removes default fields when entering an unknown type without touching action payload", () => {
    expect(changeDNSRuleType({ domain: ["example.com"], server: "dns", custom: "keep" }, "custom"))
      .toEqual({ type: "custom", server: "dns", custom: "keep" })
  })
})

describe("DNS action transitions", () => {
  it("preserves shared route options and removes the route server", () => {
    expect(changeDNSAction({
      server: "dns", strategy: "prefer_ipv4", disable_cache: true, rewrite_ttl: 60,
      client_subnet: "192.0.2.0/24", custom: "keep",
    }, "route-options")).toEqual({
      action: "route-options", strategy: "prefer_ipv4", disable_cache: true,
      rewrite_ttl: 60, client_subnet: "192.0.2.0/24", custom: "keep",
    })
  })

  it("preserves same and unknown actions by identity", () => {
    const reject = { action: "reject", method: "drop" }
    const custom = { action: "custom", payload: { enabled: true } }
    expect(changeDNSAction(reject, "reject")).toBe(reject)
    expect(changeDNSAction(custom, "custom")).toBe(custom)
  })

  it("retains target-compatible unknown-source values and rejects list kind mismatches", () => {
    expect(changeDNSAction({
      action: "custom", rcode: 5, answer: ["example. 60 IN A 192.0.2.1"],
      ns: "example. 60 IN NS ns.example.", extra: [3], method: "drop", server: "old", payload: "keep",
    }, "predefined")).toEqual({
      action: "predefined", rcode: 5, answer: ["example. 60 IN A 192.0.2.1"],
      ns: "example. 60 IN NS ns.example.", payload: "keep",
    })
  })

  it("removes stale known fields for known and unknown target actions", () => {
    expect(changeDNSAction({ action: "predefined", rcode: "REFUSED", answer: ["record"], custom: "keep" }, "reject"))
      .toEqual({ action: "reject", custom: "keep" })
    expect(changeDNSAction({ action: "reject", method: "drop", domain: ["example.com"], custom: "keep" }, "custom"))
      .toEqual({ action: "custom", domain: ["example.com"], custom: "keep" })
  })
})

describe("DNS arrays and summaries", () => {
  it("reads object entries and sets copied arrays immutably", () => {
    expect(dnsServers({ servers: [{ address: "local" }, null, "bad"] })).toEqual([{ address: "local" }])
    expect(dnsRules({ rules: [{ server: "dns" }, 1] })).toEqual([{ server: "dns" }])
    expect(dnsServers({ servers: {} })).toEqual([])
    expect(dnsRules({})).toEqual([])
    const object = { final: "dns", custom: "keep" }
    const servers = [{ type: "local" }] as const
    const rules = [{ action: "reject" }] as const
    const withServers = setDNSServers(object, servers)
    const withRules = setDNSRules(object, rules)
    expect(withServers).toEqual({ ...object, servers })
    expect(withRules).toEqual({ ...object, rules })
    expect(withServers).not.toBe(object)
    expect(withServers.servers).not.toBe(servers)
    expect(withRules.rules).not.toBe(rules)
  })

  it("summarizes legacy and modern servers", () => {
    expect(summarizeDNSServer({ tag: "google", address: "https://dns.google/dns-query" }))
      .toEqual({ type: "legacy", detail: "https://dns.google/dns-query · tag google" })
    expect(summarizeDNSServer({ type: "https", tag: "google", server: "dns.google", server_port: 443 }))
      .toEqual({ type: "https", detail: "dns.google:443 · tag google" })
    expect(summarizeDNSServer({ type: "dhcp", tag: "lan", interface: "eth0" }))
      .toEqual({ type: "dhcp", detail: "eth0 · tag lan" })
    expect(summarizeDNSServer({ type: "local", tag: "local" })).toEqual({ type: "local", detail: "tag local" })
    expect(summarizeDNSServer({ type: "hosts", tag: "hosts", path: ["/etc/hosts"], predefined: { router: "192.0.2.1" } }))
      .toEqual({ type: "hosts", detail: "path /etc/hosts · predefined 1 · tag hosts" })
    expect(summarizeDNSServer({ type: "fakeip", tag: "fake", inet4_range: "198.18.0.0/15", inet6_range: "fc00::/18" }))
      .toEqual({ type: "fakeip", detail: "inet4 198.18.0.0/15 · inet6 fc00::/18 · tag fake" })
  })

  it("summarizes match values and DNS actions", () => {
    expect(summarizeDNSRule({ domain_suffix: ["example.com"], network: "udp", server: "dns" }))
      .toEqual({ matches: ["example.com", "udp"], action: "route · dns" })
    expect(summarizeDNSRule({ source_ip_is_private: true, query_type: ["A", 28], action: "reject" }))
      .toEqual({ matches: ["source_ip_is_private", "A", "28"], action: "reject" })
    expect(summarizeDNSRule({ port: [53, null, false], action: "route-options" }).matches).toEqual(["53"])
    expect(summarizeDNSRule({ rule_set_ip_cidr_match_source: true, action: "predefined" }).matches)
      .toEqual(["rule_set_ip_cidr_match_source"])
    expect(summarizeDNSRule({ action: "custom" }).action).toBe("custom")
  })

  it("summarizes DNS address matching flags", () => {
    expect(summarizeDNSRule({ ip_accept_any: true, rule_set_ip_cidr_accept_empty: true, action: "reject" }).matches)
      .toEqual(["ip_accept_any", "rule_set_ip_cidr_accept_empty"])
  })

  it("summarizes non-empty interface address maps", () => {
    expect(summarizeDNSRule({ network_interface_address: { wifi: ["192.0.2.0/24"] }, action: "reject" }).matches)
      .toEqual(["network_interface_address"])
  })
})

describe("dns match field kinds", () => {
  it("dns match fields use ref-multi and ip select", () => {
    expect(dnsRuleMatchFields.find((field) => field.path === "inbound")).toMatchObject({ kind: "ref-multi", ref: "inbound" })
    expect(dnsRuleMatchFields.find((field) => field.path === "rule_set")).toMatchObject({ kind: "ref-multi", ref: "rule-set" })
    expect(dnsRuleMatchFields.find((field) => field.path === "ip_version")).toMatchObject({ kind: "select", options: ["4", "6"] })
    expect(dnsRuleMatchFields.find((field) => field.path === "network")).toMatchObject({ kind: "network-multi" })
  })
})

describe("DNS hierarchical field pruning", () => {
  it("prunes FakeIP ranges when disabled and drops false enabled flag", () => {
    const next = applyDNSFakeIPFieldChange({}, {
      fakeip: { enabled: false, inet4_range: "198.18.0.0/15", inet6_range: "fc00::/18", future: 1 },
    })
    expect(next).toEqual({ fakeip: { future: 1 } })
  })

  it("keeps FakeIP ranges while enabled", () => {
    expect(applyDNSFakeIPFieldChange({}, {
      fakeip: { enabled: true, inet4_range: "198.18.0.0/15" },
    })).toEqual({ fakeip: { enabled: true, inet4_range: "198.18.0.0/15" } })
  })

  it("prunes TLS and domain-resolver children when parents are off", () => {
    const next = applyDNSServerFieldChange("https", {
      type: "https",
      tag: "dns",
      server: "dns.example",
      tls: { enabled: false, server_name: "dns.example", insecure: true },
      domain_resolver: { strategy: "prefer_ipv4", rewrite_ttl: 60 },
      disable_tcp_keep_alive: true,
      tcp_keep_alive: "5m",
    })
    expect(next.tls).toEqual({ enabled: false })
    expect(next.domain_resolver).toBeUndefined()
    expect(next.tcp_keep_alive).toBeUndefined()
  })

  it("keeps TLS children and resolver children when parents are on", () => {
    const next = applyDNSServerFieldChange("tls", {
      type: "tls",
      tag: "dns",
      server: "dns.example",
      tls: { enabled: true, server_name: "dns.example" },
      domain_resolver: { server: "local", strategy: "prefer_ipv4" },
    })
    expect(next.tls).toEqual({ enabled: true, server_name: "dns.example" })
    expect(next.domain_resolver).toEqual({ server: "local", strategy: "prefer_ipv4" })
  })

  it("models server field hierarchy metadata for dialer and TLS", () => {
    const https = dnsServerFields.https
    expect(https.find((field) => field.path === "tls.server_name")).toMatchObject({
      when: { path: "tls.enabled", is: true },
    })
    expect(https.find((field) => field.path === "domain_resolver.strategy")).toMatchObject({
      when: { path: "domain_resolver.server" },
    })
    expect(https.find((field) => field.path === "detour")).toMatchObject({ kind: "ref", ref: "outbound" })
    expect(https.find((field) => field.path === "bind_interface")).toMatchObject({ kind: "network-interface" })
  })

  it("retains unknown rule keys while applying rule field prune", () => {
    const next = applyDNSRuleFieldChange(
      { action: "reject" },
      { action: "reject", method: "drop", custom: true },
    )
    expect(next).toEqual({ action: "reject", method: "drop", custom: true })
  })

  it("leaves global DNS fields untouched by prune helpers", () => {
    expect(applyDNSGlobalFieldChange({}, { final: "dns", strategy: "prefer_ipv4", custom: true }))
      .toEqual({ final: "dns", strategy: "prefer_ipv4", custom: true })
  })
})

describe("DNS summary edge branches", () => {
  it("covers no-op action change and string list helpers", () => {
    const rule = { action: "reject", method: "drop" }
    expect(changeDNSAction(rule, "reject")).toBe(rule)
    expect(summarizeDNSServer({ type: "hosts", tag: "h", path: "/etc/hosts" }).detail).toContain("path /etc/hosts")
    expect(summarizeDNSServer({ type: "udp", tag: "x", server: "1.1.1.1", server_port: Number.NaN }).detail)
      .toContain("1.1.1.1")
    expect(summarizeDNSServer({ type: "fakeip", tag: "f" }).detail).toContain("tag f")
  })

  it("covers empty action fields and unknown server type prune", () => {
    expect(applyDNSRuleFieldChange({}, { action: "unknown-action", custom: 1 }))
      .toEqual({ action: "unknown-action", custom: 1 })
    expect(applyDNSServerFieldChange("future", { type: "future", payload: true }))
      .toEqual({ type: "future", payload: true })
  })
})

describe("DNS rule completeness", () => {
  it("checks route targets and recursive logical children", () => {
    expect(isDNSRuleComplete({ server: "dns" })).toBe(true)
    expect(isDNSRuleComplete({ action: "route" })).toBe(false)
    expect(isDNSRuleComplete({ action: "reject" })).toBe(true)
    expect(isDNSRuleComplete({ type: "logical", mode: "or", rules: [{ action: "reject" }], action: "reject" })).toBe(true)
    expect(isDNSRuleComplete({ type: "logical", mode: "or", rules: [{ action: "route" }], action: "reject" })).toBe(false)
    expect(isDNSRuleComplete({ type: "logical", mode: "or", rules: [], action: "reject" })).toBe(false)
    expect(isDNSRuleComplete({ type: "logical", rules: [{ action: "reject" }], action: "reject" })).toBe(false)
    expect(isDNSRuleComplete({ type: "logical", mode: "or", action: "reject" })).toBe(false)
    expect(isDNSRuleComplete({ type: "logical", mode: "or", rules: [{ action: "reject" }], action: "reject" }, 64)).toBe(false)
    expect(isDNSRuleComplete({ type: "future", action: "future" })).toBe(true)
  })
})
