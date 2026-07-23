import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>

function text(value: JsonValue | undefined) {
  return typeof value === "string" || typeof value === "number" ? String(value) : ""
}

export function matchesProxyItem(item: JsonObject, query: string) {
  if (!query) return true
  const transport = typeof item.transport === "object" && item.transport && !Array.isArray(item.transport)
    ? text(item.transport.type)
    : ""
  const haystack = [
    text(item.tag),
    text(item.type),
    text(item.listen),
    text(item.interface_name),
    text(item.listen_port),
    text(item.server),
    text(item.server_port),
    transport,
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export type ProxyListFilters = {
  query?: string
}

export function parseProxySearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): ProxyListFilters {
  const value = params.get("q")?.trim()
  return { query: value ? value : undefined }
}

export function toProxySearchParams(filters: ProxyListFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const query = filters.query?.trim()
  if (query) params.set("q", query)
  return params
}

export function buildProxyHref(
  configKey: "inbounds" | "outbounds",
  filters: ProxyListFilters = {},
): string {
  const base = configKey === "inbounds" ? "/proxy/inbounds" : "/proxy/outbounds"
  const qs = toProxySearchParams(filters).toString()
  return qs ? `${base}?${qs}` : base
}

