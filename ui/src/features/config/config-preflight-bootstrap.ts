type JsonObject = Record<string, unknown>

interface BootstrapPreflightEntry {
  tag: string
  tagPath: string
  value: JsonObject
}

interface BootstrapTopologyInput {
  route?: JsonObject
  dns?: JsonObject
  outboundEntries: ReadonlyMap<string, BootstrapPreflightEntry>
  dnsEntries: readonly BootstrapPreflightEntry[]
}

interface BootstrapContext {
  graph: BootstrapGraph
  outbounds: ReadonlyMap<string, BootstrapPreflightEntry>
  dnsEntries: ReadonlyMap<string, BootstrapPreflightEntry>
  routeResolver?: string
  dnsResolver?: string
}

export interface BootstrapTopologyIssue {
  severity: "error"
  code: "dns_dependency_cycle"
  path: string
  reference: string
  relatedPath?: string
}

type BootstrapGraph = Map<string, Set<string>>

const dnsNodePrefix = "dns:"
const domainNodePrefix = "domain:"
const outboundNodePrefix = "outbound:"

function isObject(value: unknown): value is JsonObject {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function stringValue(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined
  const result = value.trim()
  return result || undefined
}

function arrayValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function entryPath(entry: BootstrapPreflightEntry): string {
  return entry.tagPath.replace(/\.tag$/, "")
}

function resolverTag(value: unknown): string | undefined {
  if (typeof value === "string") return stringValue(value)
  return isObject(value) ? stringValue(value.server) : undefined
}

function resolverWithSingleFallback(
  value: unknown,
  dnsEntries: ReadonlyMap<string, BootstrapPreflightEntry>,
): string | undefined {
  const resolver = resolverTag(value)
  if (resolver) return resolver
  if (dnsEntries.size === 1) return dnsEntries.keys().next().value
  return undefined
}

function isIPv4Address(value: string): boolean {
  const parts = value.split(".")
  return parts.length === 4 && parts.every((part) => (
    /^(0|[1-9]\d{0,2})$/.test(part) && Number(part) <= 255
  ))
}

function isIPv6Address(value: string): boolean {
  const address = value.split("%", 1)[0]
  if (!address.includes(":") || !/^[0-9a-f:.]+$/i.test(address)) return false
  try {
    return Boolean(new URL(`http://[${address}]/`).hostname)
  } catch {
    return false
  }
}

function isDomainName(value: unknown): boolean {
  const address = stringValue(value)
  return Boolean(address && !isIPv4Address(address) && !isIPv6Address(address))
}

function uniqueEntries(entries: readonly BootstrapPreflightEntry[]): Map<string, BootstrapPreflightEntry> {
  const result = new Map<string, BootstrapPreflightEntry>()
  for (const entry of entries) {
    if (!result.has(entry.tag)) result.set(entry.tag, entry)
  }
  return result
}

function wireGuardPeerHasDomain(value: unknown): boolean {
  return arrayValue(value).some((peer) => isObject(peer) && isDomainName(peer.address))
}

function tailscaleControlUsesDomain(value: unknown): boolean {
  const controlURL = stringValue(value)
  if (!controlURL) return true
  try {
    return isDomainName(new URL(controlURL).hostname.replace(/^\[|\]$/g, ""))
  } catch {
    return false
  }
}

function outboundHasDomainRemote(entry: BootstrapPreflightEntry): boolean {
  const type = stringValue(entry.value.type)?.toLowerCase()
  if (type === "wireguard") return wireGuardPeerHasDomain(entry.value.peers)
  if (type === "tailscale") return tailscaleControlUsesDomain(entry.value.control_url)
  return isDomainName(entry.value.server)
}

function resolvesServerOnDetour(entry: BootstrapPreflightEntry): boolean {
  const type = stringValue(entry.value.type)?.toLowerCase()
  return type === "naive" || type === "tailscale" || type === "wireguard"
}

function dnsNode(tag: string): string {
  return dnsNodePrefix + tag
}

function domainNode(tag: string): string {
  return domainNodePrefix + tag
}

function outboundNode(tag: string): string {
  return outboundNodePrefix + tag
}

function addEdge(graph: BootstrapGraph, source: string, target: string) {
  const edges = graph.get(source) ?? new Set<string>()
  edges.add(target)
  graph.set(source, edges)
}

function addDNSReference(context: BootstrapContext, source: string, resolver: string | undefined) {
  if (resolver && context.dnsEntries.has(resolver)) addEdge(context.graph, source, dnsNode(resolver))
}

function addOutboundReference(context: BootstrapContext, source: string, value: unknown) {
  const target = stringValue(value)
  if (target && context.outbounds.has(target)) addEdge(context.graph, source, outboundNode(target))
}

function addDomainReference(context: BootstrapContext, source: string, value: unknown) {
  const target = stringValue(value)
  if (target && context.outbounds.has(target)) addEdge(context.graph, source, domainNode(target))
}

function addGroupDependencies(context: BootstrapContext, entry: BootstrapPreflightEntry) {
  const type = stringValue(entry.value.type)?.toLowerCase()
  if (type !== "selector" && type !== "urltest") return
  for (const member of arrayValue(entry.value.outbounds)) {
    addOutboundReference(context, outboundNode(entry.tag), member)
    addDomainReference(context, domainNode(entry.tag), member)
  }
}

function addConfiguredRemoteDependency(context: BootstrapContext, entry: BootstrapPreflightEntry) {
  if (!outboundHasDomainRemote(entry)) return
  const source = outboundNode(entry.tag)
  const explicit = resolverTag(entry.value.domain_resolver)
  if (explicit) {
    addDNSReference(context, source, explicit)
    return
  }
  const detour = stringValue(entry.value.detour)
  if (detour && !resolvesServerOnDetour(entry)) {
    addDomainReference(context, source, detour)
    return
  }
  addDNSReference(context, source, context.routeResolver)
}

function addDomainDialDependencies(context: BootstrapContext, entry: BootstrapPreflightEntry) {
  const source = domainNode(entry.tag)
  addEdge(context.graph, source, outboundNode(entry.tag))
  const type = stringValue(entry.value.type)?.toLowerCase()
  if (type === "direct") {
    addDNSReference(context, source, resolverTag(entry.value.domain_resolver) ?? context.routeResolver)
  } else if (type === "tailscale" || type === "wireguard") {
    addDNSReference(context, source, context.dnsResolver)
  }
}

function addOutboundDependencies(context: BootstrapContext) {
  for (const entry of context.outbounds.values()) {
    addOutboundReference(context, outboundNode(entry.tag), entry.value.detour)
    addGroupDependencies(context, entry)
    addConfiguredRemoteDependency(context, entry)
    addDomainDialDependencies(context, entry)
  }
}

function addDNSDependencies(context: BootstrapContext) {
  for (const entry of context.dnsEntries.values()) {
    const source = dnsNode(entry.tag)
    addDNSReference(context, source, resolverTag(entry.value.domain_resolver))
    addDNSReference(context, source, resolverTag(entry.value.address_resolver))
    addOutboundReference(context, source, entry.value.detour)
  }
}

function createContext(input: BootstrapTopologyInput): BootstrapContext {
  const dnsEntries = uniqueEntries(input.dnsEntries)
  return {
    graph: new Map(),
    outbounds: input.outboundEntries,
    dnsEntries,
    routeResolver: resolverWithSingleFallback(input.route?.default_domain_resolver, dnsEntries),
    dnsResolver: resolverWithSingleFallback(input.dns?.final, dnsEntries),
  }
}

function reaches(graph: BootstrapGraph, start: string, target: string): boolean {
  const stack = [start]
  const visited = new Set<string>()
  while (stack.length > 0) {
    const current = stack.pop()!
    if (current === target) return true
    if (visited.has(current)) continue
    visited.add(current)
    stack.push(...(graph.get(current) ?? []))
  }
  return false
}

export function checkDNSOutboundBootstrapTopology(
  input: BootstrapTopologyInput,
): BootstrapTopologyIssue[] {
  const context = createContext(input)
  addOutboundDependencies(context)
  addDNSDependencies(context)
  const issues: BootstrapTopologyIssue[] = []
  for (const entry of context.dnsEntries.values()) {
    const detour = stringValue(entry.value.detour)
    if (!detour || !context.outbounds.has(detour)) continue
    if (!reaches(context.graph, outboundNode(detour), dnsNode(entry.tag))) continue
    issues.push({
      severity: "error",
      code: "dns_dependency_cycle",
      path: `${entryPath(entry)}.detour`,
      reference: detour,
      relatedPath: context.outbounds.get(detour)?.tagPath,
    })
  }
  return issues
}
