import {
  inferDNSServerType,
  summarizeDNSRule,
  summarizeDNSServer,
} from "@/features/policy/dns-form-model"
import type { JsonObject } from "@/features/policy/policy-form-model"

function text(value: unknown) {
  return typeof value === "string" || typeof value === "number" ? String(value) : ""
}

export function matchesDNSServer(item: JsonObject, query: string) {
  if (!query) return true
  const summary = summarizeDNSServer(item)
  const haystack = [
    text(item.tag),
    text(item.type),
    inferDNSServerType(item),
    summary.type,
    summary.detail,
    text(item.server),
    text(item.server_port),
    text(item.address),
    text(item.detour),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function matchesDNSRule(item: JsonObject, query: string) {
  if (!query) return true
  const summary = summarizeDNSRule(item)
  const haystack = [
    summary.action,
    ...summary.matches,
    text(item.type),
    text(item.action),
    text(item.server),
    text(item.mode),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export type DNSSearchFilters = {
  servers?: string
  rules?: string
}

function readParam(params: { get(name: string): string | null }, key: string): string | undefined {
  const value = params.get(key)?.trim()
  return value ? value : undefined
}

export function parseDNSSearchParams(
  params: URLSearchParams | { get(name: string): string | null },
): DNSSearchFilters {
  return {
    servers: readParam(params, "sq"),
    rules: readParam(params, "rq"),
  }
}

export function toDNSSearchParams(filters: DNSSearchFilters = {}): URLSearchParams {
  const params = new URLSearchParams()
  const servers = filters.servers?.trim()
  const rules = filters.rules?.trim()
  if (servers) params.set("sq", servers)
  if (rules) params.set("rq", rules)
  return params
}

export function buildDNSHref(filters: DNSSearchFilters = {}): string {
  const qs = toDNSSearchParams(filters).toString()
  return qs ? `/policy/dns?${qs}` : "/policy/dns"
}

