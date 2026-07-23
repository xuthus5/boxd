/** Maps backend config apply source codes to i18n keys. */
const sourceKeys: Record<string, string> = {
  update: "sourceUpdate",
  raw: "sourceRaw",
  dns_defaults: "sourceDNSDefaults",
  route_defaults: "sourceRouteDefaults",
  outbounds_defaults: "sourceOutboundsDefaults",
  inbounds_defaults: "sourceInboundsDefaults",
  experimental_defaults: "sourceExperimentalDefaults",
  rule_sets_defaults: "sourceRuleSetsDefaults",
}

export function configApplySourceKey(source: string): string {
  return sourceKeys[source] ?? "sourceUnknown"
}

export function shortConfigHash(hash: string, length = 8): string {
  const value = hash.trim()
  if (!value) return "—"
  return value.slice(0, length)
}
