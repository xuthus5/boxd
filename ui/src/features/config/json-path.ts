/** Locate a JSON path (dot/bracket notation) inside a pretty-printed document. */

export interface JsonPathLocation {
  path: string
  index: number
  line: number
  column: number
}

function isWordChar(ch: string) {
  return /[A-Za-z0-9_]/.test(ch)
}

/** Find first occurrence of object key `"key"` at a reasonable structural depth. */
export function findJsonKeyIndex(source: string, key: string, from = 0): number {
  const needle = `"${key}"`
  let index = source.indexOf(needle, from)
  while (index !== -1) {
    let cursor = index + needle.length
    while (cursor < source.length && /\s/.test(source[cursor] ?? "")) cursor += 1
    if (source[cursor] === ":") return index
    index = source.indexOf(needle, index + 1)
  }
  return -1
}

export function parseJsonPath(path: string): Array<string | number> {
  const parts: Array<string | number> = []
  const text = path.trim().replace(/^\$\.?/, "")
  if (!text || text === "$") return parts
  const re = /([^[.\]]+)|\[(\d+)\]/g
  let match: RegExpExecArray | null
  while ((match = re.exec(text)) !== null) {
    if (match[1] !== undefined && match[1] !== "") parts.push(match[1])
    else if (match[2] !== undefined) parts.push(Number(match[2]))
  }
  return parts
}

export function indexToLineColumn(source: string, index: number): { line: number; column: number } {
  const safe = Math.max(0, Math.min(index, source.length))
  const prefix = source.slice(0, safe)
  const lines = prefix.split("\n")
  return { line: lines.length, column: (lines.at(-1)?.length ?? 0) + 1 }
}

/**
 * Best-effort location of a config path inside pretty JSON text.
 * Prefers the last path segment key occurrence when full walk is ambiguous.
 */
export function locateJsonPath(source: string, path: string): JsonPathLocation | null {
  const segments = parseJsonPath(path)
  if (!segments.length) {
    return { path, index: 0, line: 1, column: 1 }
  }
  let from = 0
  let lastIndex = -1
  for (const segment of segments) {
    if (typeof segment === "number") {
      // Array indices: jump to next `[` after previous match roughly.
      const bracket = source.indexOf("[", from)
      if (bracket === -1) break
      from = bracket
      lastIndex = bracket
      continue
    }
    const found = findJsonKeyIndex(source, segment, from)
    if (found === -1) {
      // fallback: search whole document for last segment later
      break
    }
    lastIndex = found
    from = found + segment.length
  }
  if (lastIndex < 0) {
    const last = segments.at(-1)
    if (typeof last === "string") {
      lastIndex = findJsonKeyIndex(source, last, 0)
    }
  }
  if (lastIndex < 0) return null
  const { line, column } = indexToLineColumn(source, lastIndex)
  return { path, index: lastIndex, line, column }
}

export function isLikelyJsonPath(path: string) {
  const text = path.trim()
  if (!text) return false
  if (text.includes(".") || text.includes("[")) return true
  return isWordChar(text[0] ?? "")
}
