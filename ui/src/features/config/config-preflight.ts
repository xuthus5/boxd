import type { SingBoxConfig } from "@/lib/api/types"

export type ConfigPreflightSeverity = "error" | "warning"

export type ConfigPreflightCode =
  | "duplicate_tag"
  | "missing_tag"
  | "missing_outbound"
  | "missing_dns_server"
  | "missing_rule_set"
  | "empty_group"

export interface ConfigPreflightIssue {
  severity: ConfigPreflightSeverity
  code: ConfigPreflightCode
  path: string
  reference?: string
  relatedPath?: string
}

type JsonObject = Record<string, unknown>

interface NamedEntry {
  tag: string
  tagPath: string
  value: JsonObject
}

interface NamedNamespace {
  tags: Map<string, NamedEntry>
  issues: ConfigPreflightIssue[]
}

function isObject(value: unknown): value is JsonObject {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function objectValue(value: unknown, key: string): unknown {
  return isObject(value) ? value[key] : undefined
}

function stringValue(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined
  const result = value.trim()
  return result || undefined
}

function arrayValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function issue(
  code: ConfigPreflightCode,
  path: string,
  reference?: string,
  relatedPath?: string,
  severity: ConfigPreflightSeverity = "error",
): ConfigPreflightIssue {
  return { code, path, reference, relatedPath, severity }
}

function namedEntries(value: unknown, section: string): NamedEntry[] {
  const entries: NamedEntry[] = []
  for (const [index, item] of arrayValue(value).entries()) {
    if (!isObject(item)) continue
    const path = `${section}[${index}]`
    const explicitTag = stringValue(item.tag)
    entries.push({
      tag: explicitTag ?? String(index),
      tagPath: `${path}.tag`,
      value: item,
    })
  }
  return entries
}

function collectEntries(entries: NamedEntry[]): NamedNamespace {
  const tags = new Map<string, NamedEntry>()
  const issues: ConfigPreflightIssue[] = []
  for (const entry of entries) {
    const previous = tags.get(entry.tag)
    if (previous) {
      issues.push(issue("duplicate_tag", entry.tagPath, entry.tag, previous.tagPath))
      continue
    }
    tags.set(entry.tag, entry)
  }
  return { tags, issues }
}

function collectNamespace(config: SingBoxConfig, sections: string[]): NamedNamespace {
  return collectEntries(sections.flatMap((section) => namedEntries(config[section], section)))
}

function collectRuleSets(route: JsonObject): NamedNamespace {
  const tags = new Map<string, NamedEntry>()
  const issues: ConfigPreflightIssue[] = []
  for (const [index, value] of arrayValue(route.rule_set).entries()) {
    if (!isObject(value)) continue
    const path = `route.rule_set[${index}]`
    const tag = stringValue(value.tag)
    if (!tag) {
      issues.push(issue("missing_tag", `${path}.tag`))
      continue
    }
    const previous = tags.get(tag)
    if (previous) {
      issues.push(issue("duplicate_tag", `${path}.tag`, tag, previous.tagPath))
      continue
    }
    tags.set(tag, { tag, tagPath: `${path}.tag`, value })
  }
  return { tags, issues }
}

function checkReference(
  value: unknown,
  path: string,
  tags: Map<string, NamedEntry>,
  code: Extract<ConfigPreflightCode, "missing_outbound" | "missing_dns_server" | "missing_rule_set">,
  issues: ConfigPreflightIssue[],
) {
  const reference = stringValue(value)
  if (reference && !tags.has(reference)) issues.push(issue(code, path, reference))
}

function checkReferenceList(
  value: unknown,
  path: string,
  tags: Map<string, NamedEntry>,
  code: Extract<ConfigPreflightCode, "missing_outbound" | "missing_rule_set">,
  issues: ConfigPreflightIssue[],
) {
  if (typeof value === "string") {
    checkReference(value, path, tags, code, issues)
    return
  }
  for (const [index, item] of arrayValue(value).entries()) {
    const reference = stringValue(item)
    if (reference && !tags.has(reference)) issues.push(issue(code, `${path}[${index}]`, reference))
  }
}

function checkDomainResolver(
  value: unknown,
  path: string,
  dnsTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  if (typeof value === "string") {
    checkReference(value, path, dnsTags, "missing_dns_server", issues)
    return
  }
  if (isObject(value)) checkReference(value.server, `${path}.server`, dnsTags, "missing_dns_server", issues)
}

function checkNestedRouteRules(
  rules: unknown,
  basePath: string,
  outboundTags: Map<string, NamedEntry>,
  dnsTags: Map<string, NamedEntry>,
  ruleSetTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  for (const [index, value] of arrayValue(rules).entries()) {
    if (!isObject(value)) continue
    const path = `${basePath}[${index}]`
    checkReference(value.outbound, `${path}.outbound`, outboundTags, "missing_outbound", issues)
    checkReference(value.server, `${path}.server`, dnsTags, "missing_dns_server", issues)
    checkReferenceList(value.rule_set, `${path}.rule_set`, ruleSetTags, "missing_rule_set", issues)
    checkNestedRouteRules(value.rules, `${path}.rules`, outboundTags, dnsTags, ruleSetTags, issues)
  }
}

function checkNestedDNSRules(
  rules: unknown,
  basePath: string,
  dnsTags: Map<string, NamedEntry>,
  ruleSetTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  for (const [index, value] of arrayValue(rules).entries()) {
    if (!isObject(value)) continue
    const path = `${basePath}[${index}]`
    checkReference(value.server, `${path}.server`, dnsTags, "missing_dns_server", issues)
    checkReferenceList(value.rule_set, `${path}.rule_set`, ruleSetTags, "missing_rule_set", issues)
    checkNestedDNSRules(value.rules, `${path}.rules`, dnsTags, ruleSetTags, issues)
  }
}

function checkOutboundEntries(
  config: SingBoxConfig,
  outboundTags: Map<string, NamedEntry>,
  dnsTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  for (const section of ["outbounds", "endpoints"]) {
    for (const entry of namedEntries(config[section], section)) {
      const path = entry.tagPath.replace(/\.tag$/, "")
      checkReference(entry.value.detour, `${path}.detour`, outboundTags, "missing_outbound", issues)
      checkDomainResolver(entry.value.domain_resolver, `${path}.domain_resolver`, dnsTags, issues)
      const type = stringValue(entry.value.type)
      const members = arrayValue(entry.value.outbounds)
      if ((type === "selector" || type === "urltest") && members.length === 0) {
        issues.push(issue("empty_group", `${path}.outbounds`, type, undefined, "warning"))
      }
      for (const [index, member] of members.entries()) {
        checkReference(member, `${path}.outbounds[${index}]`, outboundTags, "missing_outbound", issues)
      }
      checkReference(entry.value.default, `${path}.default`, outboundTags, "missing_outbound", issues)
    }
  }
}

function checkDNSSection(
  dns: JsonObject,
  outboundTags: Map<string, NamedEntry>,
  dnsTags: Map<string, NamedEntry>,
  ruleSetTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  for (const [index, value] of arrayValue(dns.servers).entries()) {
    if (!isObject(value)) continue
    const path = `dns.servers[${index}]`
    checkReference(value.detour, `${path}.detour`, outboundTags, "missing_outbound", issues)
    checkDomainResolver(value.domain_resolver, `${path}.domain_resolver`, dnsTags, issues)
    checkReference(value.address_resolver, `${path}.address_resolver`, dnsTags, "missing_dns_server", issues)
  }
  checkReference(dns.final, "dns.final", dnsTags, "missing_dns_server", issues)
  checkNestedDNSRules(dns.rules, "dns.rules", dnsTags, ruleSetTags, issues)
}

function checkRouteSection(
  route: JsonObject,
  outboundTags: Map<string, NamedEntry>,
  dnsTags: Map<string, NamedEntry>,
  ruleSetTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  checkReference(route.final, "route.final", outboundTags, "missing_outbound", issues)
  checkDomainResolver(route.default_domain_resolver, "route.default_domain_resolver", dnsTags, issues)
  checkNestedRouteRules(route.rules, "route.rules", outboundTags, dnsTags, ruleSetTags, issues)
  for (const entry of ruleSetTags.values()) {
    const path = entry.tagPath.replace(/\.tag$/, "")
    checkReference(entry.value.download_detour, `${path}.download_detour`, outboundTags, "missing_outbound", issues)
  }
  for (const section of ["geoip", "geosite"]) {
    const options = objectValue(route, section)
    if (isObject(options)) checkReference(options.download_detour, `route.${section}.download_detour`, outboundTags, "missing_outbound", issues)
  }
}

function checkExperimentalSection(
  experimental: JsonObject,
  outboundTags: Map<string, NamedEntry>,
  issues: ConfigPreflightIssue[],
) {
  const clashAPI = objectValue(experimental, "clash_api")
  if (isObject(clashAPI)) {
    checkReference(clashAPI.external_ui_download_detour, "experimental.clash_api.external_ui_download_detour", outboundTags, "missing_outbound", issues)
  }
}

export function preflightConfig(config: SingBoxConfig): ConfigPreflightIssue[] {
  const outboundNamespace = collectNamespace(config, ["outbounds", "endpoints"])
  const inboundNamespace = collectNamespace(config, ["inbounds"])
  const issues = [...outboundNamespace.issues, ...inboundNamespace.issues]
  const route = objectValue(config, "route")
  const ruleSetNamespace = isObject(route) ? collectRuleSets(route) : { tags: new Map<string, NamedEntry>(), issues: [] }
  issues.push(...ruleSetNamespace.issues)
  const dns = objectValue(config, "dns")
  const dnsNamespace = isObject(dns) ? collectEntries(namedEntries(dns.servers, "dns.servers")) : { tags: new Map<string, NamedEntry>(), issues: [] }
  issues.push(...dnsNamespace.issues)
  checkOutboundEntries(config, outboundNamespace.tags, dnsNamespace.tags, issues)
  if (isObject(route)) checkRouteSection(route, outboundNamespace.tags, dnsNamespace.tags, ruleSetNamespace.tags, issues)
  if (isObject(dns)) checkDNSSection(dns, outboundNamespace.tags, dnsNamespace.tags, ruleSetNamespace.tags, issues)
  const experimental = objectValue(config, "experimental")
  if (isObject(experimental)) checkExperimentalSection(experimental, outboundNamespace.tags, issues)
  return issues
}
