/** Parse config paths like inbounds[0].listen_port into list index + item-relative path. */

export type ProxySection = "inbounds" | "outbounds"

export interface ProxyItemPathTarget {
  section: ProxySection
  index: number
  /** Path relative to the item object (may be empty). */
  relativePath: string
}

const TOP_LEVEL = /^(inbounds|outbounds|route|dns|endpoints|experimental|log)\b/

function matchIndexed(path: string, section: ProxySection): ProxyItemPathTarget | null {
  const text = path.trim()
  if (!text) return null
  const full = text.match(new RegExp(`^${section}\\[(\\d+)\\](?:\\.(.*))?$`))
  if (full) {
    return { section, index: Number(full[1]), relativePath: (full[2] ?? "").trim() }
  }
  const bare = text.match(/^\[(\d+)\](?:\.(.*))?$/)
  if (bare) {
    return { section, index: Number(bare[1]), relativePath: (bare[2] ?? "").trim() }
  }
  return null
}

/** Resolve a full or section-relative path to a list item target for the given proxy section. */
export function parseProxyItemPath(path: string, section: ProxySection): ProxyItemPathTarget | null {
  return matchIndexed(path, section)
}

/**
 * Candidate paths relative to the open proxy item editor.
 * Accepts full `inbounds[0].x`, bare relative `x`, or `inbounds.x`.
 * When `index < 0` (new item draft), any matching section index is accepted.
 */
export function proxyItemRelativePaths(
  path: string,
  section: ProxySection,
  index: number,
): string[] {
  const text = path.trim()
  if (!text) return []
  const target = parseProxyItemPath(text, section)
  if (target) {
    if (index >= 0 && target.index !== index) return []
    return target.relativePath ? [target.relativePath] : []
  }
  if (text.startsWith(`${section}.`)) {
    const relative = text.slice(section.length + 1).trim()
    return relative ? [relative] : []
  }
  if (!TOP_LEVEL.test(text)) return [text]
  return []
}
