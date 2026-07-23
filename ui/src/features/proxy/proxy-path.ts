/** Parse config paths like inbounds[0].listen_port into list index + item-relative path. */

export type ProxySection = "inbounds" | "outbounds"

export interface ProxyItemPathTarget {
  section: ProxySection
  index: number
  /** Path relative to the item object (may be empty). */
  relativePath: string
}

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
