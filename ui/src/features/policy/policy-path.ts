/** Parse route/dns item paths like route.rules[0].outbound into dialog targets. */

import type { JsonObject } from "@/features/policy/policy-form-model"
import type { RouteRuleMetadata } from "@/lib/api/types"

export type PolicySectionPath = "route" | "dns"
export type PolicyItemKind = "rules" | "rule_set" | "servers"

export interface PolicyItemPathTarget {
  section: PolicySectionPath
  kind: PolicyItemKind
  index: number
  /** Path relative to the item object (may be empty). */
  relativePath: string
}

export type PolicyDialogSelection =
  | { kind: "rule"; index: number; item: JsonObject; metadata?: RouteRuleMetadata; jumpPath: string }
  | { kind: "rule-set"; index: number; item: JsonObject; jumpPath: string }
  | { kind: "server"; index: number; item: JsonObject; jumpPath: string }

const KIND_BY_SECTION: Record<PolicySectionPath, readonly PolicyItemKind[]> = {
  route: ["rules", "rule_set"],
  dns: ["rules", "servers"],
}

function matchKind(path: string, section: PolicySectionPath, kind: PolicyItemKind): PolicyItemPathTarget | null {
  const text = path.trim()
  if (!text) return null
  const full = text.match(new RegExp(`^${section}\\.${kind}\\[(\\d+)\\](?:\\.(.*))?$`))
  if (full) {
    return { section, kind, index: Number(full[1]), relativePath: (full[2] ?? "").trim() }
  }
  const relative = text.match(new RegExp(`^${kind}\\[(\\d+)\\](?:\\.(.*))?$`))
  if (relative) {
    return { section, kind, index: Number(relative[1]), relativePath: (relative[2] ?? "").trim() }
  }
  return null
}

/** Resolve a full or section-relative path to a route/dns list item target. */
export function parsePolicyItemPath(
  path: string,
  section: PolicySectionPath,
): PolicyItemPathTarget | null {
  for (const kind of KIND_BY_SECTION[section]) {
    const hit = matchKind(path, section, kind)
    if (hit) return hit
  }
  return null
}

function itemAt(list: readonly JsonObject[] | undefined, index: number): JsonObject | null {
  if (!list || index < 0 || index >= list.length) return null
  const item = list[index]
  return item ?? null
}

/** Build a visual-editor selection for a route/dns list-item path, if in range. */
export function policyDialogSelectionFromPath(
  path: string,
  section: PolicySectionPath,
  lists: {
    rules?: readonly JsonObject[]
    ruleSets?: readonly JsonObject[]
    servers?: readonly JsonObject[]
    metadata?: readonly RouteRuleMetadata[]
  },
): PolicyDialogSelection | null {
  const target = parsePolicyItemPath(path, section)
  if (!target) return null
  const jumpPath = target.relativePath
  if (target.kind === "rules") {
    const item = itemAt(lists.rules, target.index)
    if (!item) return null
    const metadata = lists.metadata?.[target.index]
    return { kind: "rule", index: target.index, item, metadata, jumpPath }
  }
  if (target.kind === "rule_set") {
    const item = itemAt(lists.ruleSets, target.index)
    if (!item) return null
    return { kind: "rule-set", index: target.index, item, jumpPath }
  }
  const item = itemAt(lists.servers, target.index)
  if (!item) return null
  return { kind: "server", index: target.index, item, jumpPath }
}
