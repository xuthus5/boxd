/** Log message → connections/DNS deep-link helpers. */

import { buildDNSHref } from "@/features/policy/dns-filter"

/** Build a log search query from a connection target (host:port or host). */
export function connectionTargetLogQuery(target: string | undefined): string {
  const raw = target?.trim() ?? ""
  if (!raw) return ""
  if (raw.startsWith("[")) {
    const close = raw.indexOf("]")
    if (close > 0) return raw.slice(1, close)
  }
  // host:port -> host (only when a single trailing numeric port exists)
  const match = raw.match(/^(.*):(\d+)$/)
  if (match && !match[1].includes(":")) return match[1]
  return raw
}

const IPV4_HOST_PORT = /\b((?:\d{1,3}\.){3}\d{1,3})(?::(\d{1,5}))?\b/g
const BRACKETED_IPV6 = /\[([0-9a-fA-F:]+)\](?::(\d{1,5}))?/g
const DOMAIN_HOST_PORT = /\b((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,})(?::(\d{1,5}))?\b/g

function isLikelyIPv4(value: string): boolean {
  const parts = value.split(".")
  if (parts.length !== 4) return false
  return parts.every((part) => {
    if (!/^\d{1,3}$/.test(part)) return false
    const n = Number(part)
    return n >= 0 && n <= 255
  })
}

function isLikelyPort(value: string | undefined): boolean {
  if (!value) return true
  if (!/^\d{1,5}$/.test(value)) return false
  const n = Number(value)
  return n >= 1 && n <= 65535
}

/** Extract a connection-search host from a sing-box style log message. */
export function logConnectionQuery(message: string | undefined): string {
  const raw = message?.trim() ?? ""
  if (!raw) return ""

  // Prefer destination hosts from "connection to ..."; fall back to "connection from ...".
  const connectionTo = raw.match(
    /\b(?:inbound|outbound)\s+connection\s+to\s+(\[?[\w.:-]+\]?)/i,
  )
  if (connectionTo?.[1]) return connectionTargetLogQuery(connectionTo[1])

  const connectionFrom = raw.match(
    /\b(?:inbound|outbound)\s+connection\s+from\s+(\[?[\w.:-]+\]?)/i,
  )
  if (connectionFrom?.[1]) {
    const host = connectionTargetLogQuery(connectionFrom[1])
    if (host && !host.startsWith("127.") && host !== "0.0.0.0" && host !== "::1") return host
  }

  // DNS-style: "lookup example.com" / "query example.com"
  const dnsMatch = raw.match(/\b(?:lookup|query|resolve)\s+([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})\b/i)
  if (dnsMatch?.[1]) return dnsMatch[1]

  // First bracketed IPv6 host
  BRACKETED_IPV6.lastIndex = 0
  const ipv6 = BRACKETED_IPV6.exec(raw)
  if (ipv6?.[1] && isLikelyPort(ipv6[2])) return ipv6[1]

  // First plausible domain
  DOMAIN_HOST_PORT.lastIndex = 0
  let domainHit: RegExpExecArray | null
  while ((domainHit = DOMAIN_HOST_PORT.exec(raw)) !== null) {
    const host = domainHit[1]
    if (!isLikelyPort(domainHit[2])) continue
    // Skip version-like tokens such as v1.2.3
    if (/^v?\d+(\.\d+)+$/.test(host)) continue
    return host
  }

  // First public-looking IPv4 (skip loopback/wildcard noise)
  IPV4_HOST_PORT.lastIndex = 0
  let ipHit: RegExpExecArray | null
  while ((ipHit = IPV4_HOST_PORT.exec(raw)) !== null) {
    const host = ipHit[1]
    if (!isLikelyIPv4(host) || !isLikelyPort(ipHit[2])) continue
    if (host.startsWith("127.") || host === "0.0.0.0") continue
    return host
  }

  return ""
}

function connectionsQueryHref(query: string): string {
  const params = new URLSearchParams()
  params.set("q", query)
  return `/observability/connections?${params}`
}

/** Build a connections deep-link from a log message when a host can be extracted. */
export function logConnectionsHref(message: string | undefined): string {
  const query = logConnectionQuery(message)
  return query ? connectionsQueryHref(query) : ""
}

/** Extract a DNS server tag from common sing-box / panel log phrasing. */
export function logDNSServerTag(message: string | undefined): string {
  const raw = message?.trim() ?? ""
  if (!raw) return ""
  const bracket = raw.match(/\bdns\/[a-zA-Z0-9._-]+\[([a-zA-Z0-9._-]+)\]/i)
  if (bracket?.[1]) return bracket[1]
  const patterns = [
    /\bdns\/([a-zA-Z0-9._-]+)/i,
    /\bserver(?:\s+tag)?[=:\s]+["']?([a-zA-Z0-9._-]+)/i,
    /\busing\s+([a-zA-Z0-9._-]+)\s+for\s+(?:dns|query|lookup)/i,
  ]
  for (const pattern of patterns) {
    const match = raw.match(pattern)
    const tag = match?.[1]
    if (!tag) continue
    if (/^(dns|udp|tcp|tls|https|quic)$/i.test(tag)) continue
    return tag
  }
  return ""
}

/** Build a DNS policy deep-link from a log message when it looks DNS-related. */
export function logDNSHref(message: string | undefined): string {
  const raw = message?.trim() ?? ""
  if (!raw) return ""
  const lower = raw.toLowerCase()
  const dnsRelated = /\b(dns|lookup|resolve|query|fakeip|nameserver)\b/i.test(lower)
  if (!dnsRelated) return ""
  const serverTag = logDNSServerTag(raw)
  if (serverTag) return buildDNSHref({ servers: serverTag })
  const host = logConnectionQuery(raw)
  if (host) return buildDNSHref({ rules: host })
  if (/\b(reject|block)\b/i.test(lower)) return buildDNSHref({ ruleAction: "reject" })
  return buildDNSHref()
}
