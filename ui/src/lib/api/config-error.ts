/** Parse sing-box / boxd config validation messages into UI-friendly pieces. */

export interface ParsedConfigError {
  message: string
  path?: string
  summary: string
}

// Path token: letters/digits/underscore/dot/brackets/quotes/hyphen (no char-class escapes needed).
const pathToken = String.raw`[A-Za-z0-9_."'\[\]-]+`

const pathPatterns = [
  /decode config at ([^\s:]+)/i,
  new RegExp(String.raw`decode (${pathToken})[:\s]`, "i"),
  new RegExp(String.raw`field\s+(${pathToken})`, "i"),
  new RegExp(String.raw`(?:at|path)\s+(${pathToken})`, "i"),
  new RegExp(String.raw`\x60(${pathToken})\x60`, "i"),
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
  const leading = text.match(/^([A-Za-z_][\w.[\]-]*)\s*[:：]/)
  if (leading?.[1] && (leading[1].includes(".") || leading[1].includes("["))) {
    return leading[1]
  }
  return undefined
}

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

export function parseConfigError(message: string): ParsedConfigError {
  const cleaned = message.trim() || "invalid sing-box config"
  const path = extractConfigPath(cleaned)
  if (!path) return { message: cleaned, summary: cleaned }
  const escaped = escapeRegExp(path)
  const withoutPath = cleaned
    .replace(new RegExp(String.raw`^${escaped}\s*[:：]\s*`), "")
    .replace(new RegExp(String.raw`(?:at|path|field|decode(?: config at)?)\s+["'\`]?${escaped}["'\`]?\s*[:：]?\s*`, "i"), "")
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
