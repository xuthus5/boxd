type JsonObject = Record<string, unknown>

export interface OutboundPreflightEntry {
  tag: string
  tagPath: string
  value: JsonObject
}

export interface OutboundTopologyIssue {
  severity: "error"
  code: "invalid_group_default" | "outbound_dependency_cycle"
  path: string
  reference: string
  relatedPath?: string
}

interface OutboundDependencyEdge {
  target: string
  path: string
}

type VisitState = "visiting" | "visited"

function stringValue(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined
  const result = value.trim()
  return result || undefined
}

function arrayValue(value: unknown): unknown[] {
  return Array.isArray(value) ? value : []
}

function entryPath(entry: OutboundPreflightEntry): string {
  return entry.tagPath.replace(/\.tag$/, "")
}

function dependencyEdges(
  entry: OutboundPreflightEntry,
  entries: ReadonlyMap<string, OutboundPreflightEntry>,
): OutboundDependencyEdge[] {
  const path = entryPath(entry)
  const edges: OutboundDependencyEdge[] = []
  const detour = stringValue(entry.value.detour)
  if (detour && entries.has(detour)) edges.push({ target: detour, path: `${path}.detour` })
  const type = stringValue(entry.value.type)
  if (type !== "selector" && type !== "urltest") return edges
  for (const [index, value] of arrayValue(entry.value.outbounds).entries()) {
    const target = stringValue(value)
    if (target && entries.has(target)) edges.push({ target, path: `${path}.outbounds[${index}]` })
  }
  return edges
}

function invalidDefaultIssue(
  entry: OutboundPreflightEntry,
  entries: ReadonlyMap<string, OutboundPreflightEntry>,
): OutboundTopologyIssue | undefined {
  if (stringValue(entry.value.type) !== "selector") return undefined
  const target = stringValue(entry.value.default)
  if (!target || !entries.has(target)) return undefined
  const members = new Set(arrayValue(entry.value.outbounds).map(stringValue).filter((value): value is string => Boolean(value)))
  if (members.has(target)) return undefined
  const path = entryPath(entry)
  return {
    severity: "error",
    code: "invalid_group_default",
    path: `${path}.default`,
    reference: target,
  }
}

function visitDependencies(
  tag: string,
  graph: ReadonlyMap<string, readonly OutboundDependencyEdge[]>,
  entries: ReadonlyMap<string, OutboundPreflightEntry>,
  states: Map<string, VisitState>,
  issues: OutboundTopologyIssue[],
) {
  states.set(tag, "visiting")
  for (const edge of graph.get(tag) ?? []) {
    const state = states.get(edge.target)
    if (!state) visitDependencies(edge.target, graph, entries, states, issues)
    else if (state === "visiting") {
      issues.push({
        severity: "error",
        code: "outbound_dependency_cycle",
        path: edge.path,
        reference: edge.target,
        relatedPath: entries.get(edge.target)?.tagPath ?? edge.target,
      })
    }
  }
  states.set(tag, "visited")
}

export function checkOutboundTopology(
  entries: ReadonlyMap<string, OutboundPreflightEntry>,
): OutboundTopologyIssue[] {
  const issues: OutboundTopologyIssue[] = []
  const graph = new Map<string, OutboundDependencyEdge[]>()
  for (const [tag, entry] of entries) {
    graph.set(tag, dependencyEdges(entry, entries))
    const defaultIssue = invalidDefaultIssue(entry, entries)
    if (defaultIssue) issues.push(defaultIssue)
  }
  const states = new Map<string, VisitState>()
  for (const tag of entries.keys()) {
    if (!states.has(tag)) visitDependencies(tag, graph, entries, states, issues)
  }
  return issues
}
