type JsonObject = Record<string, unknown>

export interface DNSPreflightEntry {
  tag: string
  tagPath: string
  value: JsonObject
}

export interface DNSTopologyIssue {
  severity: "error"
  code: "dns_dependency_cycle" | "invalid_dns_default" | "multiple_fakeip_dns_servers"
  path: string
  reference: string
  relatedPath?: string
}

interface DNSDependencyEdge {
  target: string
  path: string
}

interface FakeIPEntry {
  entry: DNSPreflightEntry
  path: string
}

type VisitState = "visiting" | "visited"

function isObject(value: unknown): value is JsonObject {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

function stringValue(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined
  const result = value.trim()
  return result || undefined
}

function entryPath(entry: DNSPreflightEntry): string {
  return entry.tagPath.replace(/\.tag$/, "")
}

function uniqueEntries(entries: readonly DNSPreflightEntry[]): Map<string, DNSPreflightEntry> {
  const result = new Map<string, DNSPreflightEntry>()
  for (const entry of entries) {
    if (!result.has(entry.tag)) result.set(entry.tag, entry)
  }
  return result
}

function fakeIPEntry(entry: DNSPreflightEntry): FakeIPEntry | undefined {
  const path = entryPath(entry)
  const type = stringValue(entry.value.type)
  if (type === "fakeip") return { entry, path: `${path}.type` }
  const legacy = !type || type === "legacy"
  if (legacy && stringValue(entry.value.address) === "fakeip") return { entry, path: `${path}.address` }
  return undefined
}

function resolverEdge(value: unknown, path: string): DNSDependencyEdge | undefined {
  if (typeof value === "string") {
    const target = stringValue(value)
    return target ? { target, path } : undefined
  }
  if (!isObject(value)) return undefined
  const target = stringValue(value.server)
  return target ? { target, path: `${path}.server` } : undefined
}

function dependencyEdges(
  entry: DNSPreflightEntry,
  entries: ReadonlyMap<string, DNSPreflightEntry>,
): DNSDependencyEdge[] {
  const path = entryPath(entry)
  const candidates = [
    resolverEdge(entry.value.domain_resolver, `${path}.domain_resolver`),
    resolverEdge(entry.value.address_resolver, `${path}.address_resolver`),
  ]
  return candidates.filter((edge): edge is DNSDependencyEdge => Boolean(edge && entries.has(edge.target)))
}

function visitDependencies(
  tag: string,
  graph: ReadonlyMap<string, readonly DNSDependencyEdge[]>,
  entries: ReadonlyMap<string, DNSPreflightEntry>,
  states: Map<string, VisitState>,
  issues: DNSTopologyIssue[],
) {
  states.set(tag, "visiting")
  for (const edge of graph.get(tag) ?? []) {
    const state = states.get(edge.target)
    if (!state) visitDependencies(edge.target, graph, entries, states, issues)
    else if (state === "visiting") {
      issues.push({
        severity: "error",
        code: "dns_dependency_cycle",
        path: edge.path,
        reference: edge.target,
        relatedPath: entries.get(edge.target)?.tagPath ?? edge.target,
      })
    }
  }
  states.set(tag, "visited")
}

function dependencyCycleIssues(entries: ReadonlyMap<string, DNSPreflightEntry>): DNSTopologyIssue[] {
  const graph = new Map<string, DNSDependencyEdge[]>()
  for (const [tag, entry] of entries) graph.set(tag, dependencyEdges(entry, entries))
  const states = new Map<string, VisitState>()
  const issues: DNSTopologyIssue[] = []
  for (const tag of entries.keys()) {
    if (!states.has(tag)) visitDependencies(tag, graph, entries, states, issues)
  }
  return issues
}

function invalidDefaultIssue(
  dns: JsonObject,
  entries: readonly DNSPreflightEntry[],
  entriesByTag: ReadonlyMap<string, DNSPreflightEntry>,
): DNSTopologyIssue | undefined {
  const final = stringValue(dns.final)
  const entry = final ? entriesByTag.get(final) : entries.find((item) => entryPath(item) === "dns.servers[0]")
  if (!entry) return undefined
  const fakeIP = fakeIPEntry(entry)
  if (!fakeIP) return undefined
  return {
    severity: "error",
    code: "invalid_dns_default",
    path: final ? "dns.final" : fakeIP.path,
    reference: entry.tag,
    relatedPath: final ? entry.tagPath : undefined,
  }
}

function multipleFakeIPIssues(entries: readonly DNSPreflightEntry[]): DNSTopologyIssue[] {
  const fakeIPEntries = entries.map(fakeIPEntry).filter((entry): entry is FakeIPEntry => Boolean(entry))
  const first = fakeIPEntries[0]
  if (!first) return []
  return fakeIPEntries.slice(1).map(({ entry, path }) => ({
    severity: "error",
    code: "multiple_fakeip_dns_servers",
    path,
    reference: entry.tag,
    relatedPath: first.entry.tagPath,
  }))
}

export function checkDNSTopology(
  dns: JsonObject,
  entries: readonly DNSPreflightEntry[],
): DNSTopologyIssue[] {
  const entriesByTag = uniqueEntries(entries)
  const issues = dependencyCycleIssues(entriesByTag)
  const defaultIssue = invalidDefaultIssue(dns, entries, entriesByTag)
  if (defaultIssue) issues.push(defaultIssue)
  issues.push(...multipleFakeIPIssues(entries))
  return issues
}
