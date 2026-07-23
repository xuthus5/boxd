/** Parse sing-box / boxd config validation messages into UI-friendly pieces. */

export interface ParsedConfigError {
  message: string
  path?: string
  summary: string
}

const pathPatterns = [
  /decode config at ([^\s:]+)/i,
  /decode ([a-z0-9_.\[\]"-]+)[:\s]/i,
  /field\s+([a-z0-9_.\[\]"-]+)/i,
  /(?:at|path)\s+([a-z0-9_.\[\]"-]+)/i,
  /`([a-z0-9_.\[\]"-]+)`/i,
]

function normalizePath(raw: string) {
  return raw.replace(/^["'`]+|["'`]+$/g, "").replace(/^\./, "")
}

export function extractConfigPath(message: string): string | undefined {
  const text = message.trim()
  if (!text) return undefined
  for (const pattern of pathPatterns) {
    const match = text.match(pattern)
    if (!match?.[1]) continue
    const path = normalizePath(match[1])
    if (path.includes(".") || path.includes("[") || /^(inbounds|outbounds|route|dns|experimental|log|ntp)/i.test(path)) {
      return path
    }
  }
  // common: "inbounds[0].listen_port: ..."
  const leading = text.match(/^([a-zA-Z_][\w.\[\]-]*)\s*[:：]/)
  if (leading?.[1] && (leading[1].includes(".") || leading[1].includes("["))) {
    return leading[1]
  }
  return undefined
}

export function parseConfigError(message: string): ParsedConfigError {
  const cleaned = message.trim() || "invalid sing-box config"
  const path = extractConfigPath(cleaned)
  if (!path) return { message: cleaned, summary: cleaned }
  const withoutPath = cleaned
    .replace(new RegExp(`^${path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s*[:：]\\s*`), "")
    .replace(new RegExp(`(?:at|path|field|decode(?: config at)?)\\s+["'\`]?${path.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}["'\`]?\\s*[:：]?\\s*`, "i"), "")
    .trim()
  const detail = withoutPath && withoutPath !== cleaned ? withoutPath : cleaned
  return {
    message: cleaned,
    path,
    summary: `${path}: ${detail}`,
  }
}

export function formatConfigErrorMessage(message: string): string {
  return parseConfigError(message).summary
}
