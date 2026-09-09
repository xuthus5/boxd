import {
  isJsonObject,
  pruneInvisiblePolicyFields,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

const cacheOn = { path: "cache_file.enabled", is: true as const }
const statsOn = { path: "v2ray_api.stats.enabled", is: true as const }
export const experimentalCacheFields = [
  { path: "cache_file.store_dns", label: "storeDNS", kind: "boolean", when: cacheOn },
  { path: "cache_file.enabled", label: "cacheEnabled", kind: "boolean" },
  { path: "cache_file.path", label: "cachePath", when: cacheOn },
  { path: "cache_file.cache_id", label: "cacheID", when: cacheOn },
  { path: "cache_file.store_fakeip", label: "storeFakeIP", kind: "boolean", when: cacheOn },
] as const satisfies readonly PolicyFieldSpec[]

export const experimentalClashFields = [
  { path: "clash_api.external_controller", label: "externalController" },
  { path: "clash_api.external_ui", label: "externalUI" },
  { path: "clash_api.external_ui_download_url", label: "externalUIDownloadURL" },
  { path: "clash_api.external_ui_download_detour", label: "externalUIDownloadDetour", kind: "ref", ref: "outbound" },
  { path: "clash_api.secret", label: "clashSecret" },
  { path: "clash_api.default_mode", label: "defaultMode", kind: "select", options: ["rule", "global", "direct"] },
  { path: "clash_api.access_control_allow_origin", label: "accessControlAllowOrigin", kind: "list" },
  { path: "clash_api.access_control_allow_private_network", label: "accessControlAllowPrivateNetwork", kind: "boolean" },
] as const satisfies readonly PolicyFieldSpec[]

export const experimentalV2RayFields = [
  { path: "v2ray_api.listen", label: "v2rayListen" },
  { path: "v2ray_api.stats.enabled", label: "statsEnabled", kind: "boolean" },
  { path: "v2ray_api.stats.inbounds", label: "statsInbounds", kind: "ref-multi", ref: "inbound", when: statsOn },
  { path: "v2ray_api.stats.outbounds", label: "statsOutbounds", kind: "ref-multi", ref: "outbound", when: statsOn },
  { path: "v2ray_api.stats.users", label: "statsUsers", kind: "list", when: statsOn },
] as const satisfies readonly PolicyFieldSpec[]

export const experimentalFields = [
  ...experimentalCacheFields,
  ...experimentalClashFields,
  ...experimentalV2RayFields,
] as const satisfies readonly PolicyFieldSpec[]

export function isExperimentalStructureValid(value: JsonValue | null | undefined): value is JsonObject {
  return isJsonObject(value ?? undefined)
}

export function normalizeExperimentalObject(value: JsonValue | undefined): JsonObject {
  return isJsonObject(value) ? value : {}
}

export function prepareExperimentalObject(object: JsonObject): JsonObject {
  return pruneInvisiblePolicyFields(object, experimentalFields)
}
