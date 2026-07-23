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
