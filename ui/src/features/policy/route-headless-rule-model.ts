import {
  getPolicyPath,
  isJsonObject,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"

export const headlessRuleMatchFields = [
  { path: "type", label: "ruleType", kind: "select", options: ["default", "logical"], section: "basic" },
  { path: "query_type", label: "queryType", kind: "list", section: "basic" },
  { path: "network", label: "network", kind: "network-multi", section: "basic" },
  { path: "domain", label: "domain", kind: "list", section: "domain" },
  { path: "domain_suffix", label: "domainSuffix", kind: "list", section: "domain" },
  { path: "domain_keyword", label: "domainKeyword", kind: "list", section: "domain" },
  { path: "domain_regex", label: "domainRegex", kind: "list", section: "domain" },
  { path: "source_ip_cidr", label: "sourceIPCIDR", kind: "list", section: "domain" },
  { path: "ip_cidr", label: "ipCIDR", kind: "list", section: "domain" },
  { path: "source_port", label: "sourcePort", kind: "number-list", section: "process" },
  { path: "source_port_range", label: "sourcePortRange", kind: "list", section: "process" },
  { path: "port", label: "port", kind: "number-list", section: "process" },
  { path: "port_range", label: "portRange", kind: "list", section: "process" },
  { path: "process_name", label: "processName", kind: "list", section: "process" },
  { path: "process_path", label: "processPath", kind: "list", section: "process" },
  { path: "process_path_regex", label: "processPathRegex", kind: "list", section: "process" },
  { path: "package_name", label: "packageName", kind: "list", section: "process" },
  { path: "network_type", label: "networkType", kind: "list", section: "environment" },
  { path: "network_is_expensive", label: "networkIsExpensive", kind: "boolean", section: "environment" },
  { path: "network_is_constrained", label: "networkIsConstrained", kind: "boolean", section: "environment" },
  { path: "network_interface_address", label: "networkInterfaceAddress", kind: "json-object", section: "environment" },
  { path: "default_interface_address", label: "defaultInterfaceAddress", kind: "list", section: "environment" },
  { path: "wifi_ssid", label: "wifiSSID", kind: "list", section: "environment" },
  { path: "wifi_bssid", label: "wifiBSSID", kind: "list", section: "environment" },
  { path: "invert", label: "invert", kind: "boolean", section: "basic" },
] as const satisfies readonly PolicyFieldSpec[]

export const logicalHeadlessRuleFields = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"], required: true },
  { path: "rules", label: "logicalRules", kind: "json-array", required: true },
  headlessRuleMatchFields.at(-1)!,
] as const satisfies readonly PolicyFieldSpec[]

const defaultFields = headlessRuleMatchFields.filter((field) => field.path !== "type")
const fieldsByType: Record<string, readonly PolicyFieldSpec[]> = {
  default: defaultFields,
  logical: logicalHeadlessRuleFields,
}
const knownFields = [...new Map(
  Object.values(fieldsByType).flat().map((field) => [field.path, field]),
).values()]

function matchesField(value: JsonObject[string] | undefined, field: PolicyFieldSpec) {
  if (value === undefined) return true
  if (field.kind === "boolean") return typeof value === "boolean"
  if (field.kind === "number-list") {
    return typeof value === "number" && Number.isFinite(value)
      || Array.isArray(value) && value.every((item) => typeof item === "number" && Number.isFinite(item))
  }
  if (field.path === "query_type") {
    return typeof value === "string" || typeof value === "number"
      || Array.isArray(value) && value.every((item) => typeof item === "string" || typeof item === "number")
  }
  if (field.kind === "list" || field.kind === "network-multi") {
    return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  }
  if (field.kind === "json-object") return isJsonObject(value)
  if (field.kind === "json-array") return Array.isArray(value)
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean"
}

export function changeHeadlessRuleType(rule: JsonObject, type: string): JsonObject {
  const current = String(rule.type ?? "default")
  if (current === type) return rule
  const source = fieldsByType[current]
  const target = fieldsByType[type] ?? []
  const compatible = new Map(target.filter((field) => (
    !source || source.some((currentField) => currentField.path === field.path && currentField.kind === field.kind)
  )).map((field) => [field.path, field]))
  const next = knownFields.reduce((object, field) => {
    const targetField = compatible.get(field.path)
    return targetField && matchesField(getPolicyPath(object, field.path), targetField)
      ? object
      : setPolicyPath(object, field.path, undefined)
  }, rule)
  return setPolicyPath(next, "type", type === "default" ? undefined : type)
}

export function headlessRules(ruleSet: JsonObject): JsonObject[] {
  return Array.isArray(ruleSet.rules) ? ruleSet.rules.filter(isJsonObject) : []
}

export function setHeadlessRules(ruleSet: JsonObject, rules: readonly JsonObject[]): JsonObject {
  return { ...ruleSet, rules: [...rules] }
}

function hasMeaningfulValue(value: JsonObject[string] | undefined) {
  if (Array.isArray(value)) return value.length > 0
  if (isJsonObject(value)) return Object.keys(value).length > 0
  if (typeof value === "string") return value.length > 0
  if (typeof value === "number") return Number.isFinite(value)
  return value === true
}

export function isHeadlessRuleComplete(rule: JsonObject): boolean {
  const type = String(rule.type ?? "default")
  if (type === "logical") {
    return typeof rule.mode === "string" && rule.mode.length > 0
      && Array.isArray(rule.rules) && rule.rules.length > 0 && rule.rules.every(isJsonObject)
  }
  if (type !== "default") return true
  return defaultFields.some((field) => field.path !== "invert" && hasMeaningfulValue(getPolicyPath(rule, field.path)))
}

const summaryPaths = [
  "domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "ip_cidr",
  "source_port", "source_port_range", "port", "port_range", "process_name", "process_path",
  "process_path_regex", "package_name", "query_type", "network", "network_type",
  "network_is_expensive", "network_is_constrained", "network_interface_address",
  "default_interface_address", "wifi_ssid", "wifi_bssid", "invert",
]

export interface HeadlessRuleSummaryLabels {
  matchLabel: (path: string) => string
}

const defaultSummaryLabels: HeadlessRuleSummaryLabels = { matchLabel: (path) => path }

function summarizeValue(path: string, value: JsonObject[string] | undefined, labels: HeadlessRuleSummaryLabels) {
  if (Array.isArray(value)) return value.flatMap((item) => typeof item === "string" || typeof item === "number" ? [String(item)] : [])
  if (isJsonObject(value) && Object.keys(value).length > 0) return [labels.matchLabel(path)]
  if (typeof value === "string" || typeof value === "number") return [String(value)]
  return value === true ? [labels.matchLabel(path)] : []
}

export function summarizeHeadlessRule(rule: JsonObject, labels = defaultSummaryLabels) {
  const type = String(rule.type ?? "default")
  if (type === "logical") {
    const matches = typeof rule.mode === "string" && rule.mode ? [rule.mode] : []
    return { type, matches, childRules: Array.isArray(rule.rules) ? rule.rules.filter(isJsonObject).length : 0 }
  }
  return { type, matches: summaryPaths.flatMap((path) => summarizeValue(path, rule[path], labels)), childRules: 0 }
}
