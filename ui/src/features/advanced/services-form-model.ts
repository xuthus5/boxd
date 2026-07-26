import {
  getPolicyPath,
  isJsonObject,
  pruneInvisiblePolicyFields,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

export const serviceTypes = ["ccm", "derp", "ocm", "resolved", "ssm-api"] as const
export type ServiceType = (typeof serviceTypes)[number]

const listenerTypes = ["ccm", "derp", "ocm", "resolved", "ssm-api"] as const
const tlsTypes = ["ccm", "derp", "ocm", "ssm-api"] as const
const listenerWhen = { path: "type", is: listenerTypes }
const tlsWhen = { path: "type", is: tlsTypes }

export const serviceIdentityFields = [
  { path: "type", label: "type", kind: "select", options: serviceTypes, required: true, section: "basic" },
  { path: "tag", label: "tag", section: "basic" },
] as const satisfies readonly PolicyFieldSpec[]

export const serviceListenFields = [
  { path: "listen", label: "listen", required: true, section: "listen", when: listenerWhen },
  { path: "listen_port", label: "listenPort", kind: "number", required: true, section: "listen", when: listenerWhen },
  { path: "bind_interface", label: "bindInterface", section: "listen", when: listenerWhen },
  { path: "routing_mark", label: "routingMark", section: "listen", when: listenerWhen },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "listen", when: listenerWhen },
  { path: "netns", label: "networkNamespace", section: "listen", when: listenerWhen },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "listen", when: listenerWhen },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "listen", when: listenerWhen },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "listen", when: listenerWhen },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "listen", when: listenerWhen },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "listen", when: listenerWhen },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "listen", when: listenerWhen },
  { path: "udp_timeout", label: "udpTimeout", section: "listen", when: listenerWhen },
] as const satisfies readonly PolicyFieldSpec[]

export const serviceDetourFields = [
  { path: "detour", label: "detour", section: "listen", when: listenerWhen },
] as const satisfies readonly PolicyFieldSpec[]

export const serviceTLSFields = [
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean", section: "tls", when: tlsWhen },
  { path: "tls.server_name", label: "serverName", section: "tls", when: tlsWhen },
  { path: "tls.insecure", label: "insecure", kind: "boolean", section: "tls", when: tlsWhen },
  { path: "tls.alpn", label: "alpn", kind: "list", section: "tls", when: tlsWhen },
  { path: "tls.certificate", label: "certificate", kind: "textarea", section: "tls", when: tlsWhen },
  { path: "tls.certificate_path", label: "certificatePath", section: "tls", when: tlsWhen },
  { path: "tls.key", label: "key", kind: "textarea", section: "tls", when: tlsWhen },
  { path: "tls.key_path", label: "keyPath", section: "tls", when: tlsWhen },
  { path: "tls.client_authentication", label: "clientAuthentication", kind: "select", options: ["no", "request", "require-any", "verify-if-given", "require-and-verify"], section: "tls", when: tlsWhen },
  { path: "tls.client_certificate", label: "clientCertificate", kind: "textarea", section: "tls", when: tlsWhen },
  { path: "tls.client_certificate_path", label: "clientCertificatePath", kind: "list", section: "tls", when: tlsWhen },
] as const satisfies readonly PolicyFieldSpec[]

export const ccmFields = [
  { path: "credential_path", label: "credentialPath", section: "ccm", when: { path: "type", is: ["ccm", "ocm"] } },
  { path: "usages_path", label: "usagesPath", section: "ccm", when: { path: "type", is: ["ccm", "ocm"] } },
  { path: "users", label: "users", kind: "json-array", section: "ccm", when: { path: "type", is: ["ccm", "ocm"] } },
  { path: "headers", label: "headers", kind: "json-object", section: "ccm", when: { path: "type", is: ["ccm", "ocm"] } },
] as const satisfies readonly PolicyFieldSpec[]

export const derpFields = [
  { path: "config_path", label: "configPath", required: true, section: "derp", when: { path: "type", is: "derp" } },
  { path: "verify_client_endpoint", label: "verifyClientEndpoint", kind: "list", section: "derp", when: { path: "type", is: "derp" } },
  { path: "verify_client_url", label: "verifyClientURL", kind: "json-array", section: "derp", when: { path: "type", is: "derp" } },
  { path: "home", label: "home", section: "derp", when: { path: "type", is: "derp" } },
  { path: "mesh_with", label: "meshWith", kind: "json-array", section: "derp", when: { path: "type", is: "derp" } },
  { path: "mesh_psk", label: "meshPSK", section: "derp", when: { path: "type", is: "derp" } },
  { path: "mesh_psk_file", label: "meshPSKFile", section: "derp", when: { path: "type", is: "derp" } },
  { path: "stun", label: "stun", kind: "json-object", section: "derp", when: { path: "type", is: "derp" } },
] as const satisfies readonly PolicyFieldSpec[]

export const ssmAPIFields = [
  { path: "servers", label: "servers", kind: "json-object", required: true, section: "ssmAPI", when: { path: "type", is: "ssm-api" } },
  { path: "cache_path", label: "cachePath", section: "ssmAPI", when: { path: "type", is: "ssm-api" } },
] as const satisfies readonly PolicyFieldSpec[]

export const serviceFields = [
  ...serviceIdentityFields,
  ...serviceListenFields,
  ...serviceDetourFields,
  ...serviceTLSFields,
  ...ccmFields,
  ...derpFields,
  ...ssmAPIFields,
] as const satisfies readonly PolicyFieldSpec[]

export function isServiceType(value: JsonValue | undefined): value is ServiceType {
  return typeof value === "string" && serviceTypes.includes(value as ServiceType)
}

export function inferServiceType(object: JsonObject): ServiceType | undefined {
  return isServiceType(object.type) ? object.type : undefined
}

export function isServiceObject(value: JsonValue | undefined): value is JsonObject {
  return isJsonObject(value) && typeof value.type === "string"
}

export function isServicesStructureValid(value: JsonValue | null | undefined): value is JsonObject[] {
  return Array.isArray(value) && value.every(isServiceObject)
}

export function normalizeServices(value: JsonValue | undefined): JsonObject[] {
  return isServicesStructureValid(value) ? value.map((item) => normalizeServiceObject(item)) : []
}

export function createServiceDraft(type: ServiceType = "resolved"): JsonObject {
  if (type === "resolved") {
    return { type, tag: "", listen: "127.0.0.53", listen_port: 53 }
  }
  return { type, tag: "", listen: "", listen_port: 0 }
}

export function normalizeServiceObject(value: JsonValue | undefined): JsonObject {
  if (!isServiceObject(value)) return createServiceDraft()
  const normalized = { ...value }
  if (normalized.type === "resolved") {
    if (normalized.listen === undefined || normalized.listen === "") normalized.listen = "127.0.0.53"
    if (normalized.listen_port === undefined || normalized.listen_port === 0) normalized.listen_port = 53
  }
  return normalized
}

export function isServiceReady(value: JsonValue | undefined): value is JsonObject {
  if (!isServiceObject(value)) return false
  const type = inferServiceType(value)
  if (!type) return false
  const normalized = normalizeServiceObject(value)
  if (typeof normalized.listen !== "string" || !normalized.listen.trim()) return false
  if (typeof normalized.listen_port !== "number" || !Number.isInteger(normalized.listen_port)
    || normalized.listen_port < 1 || normalized.listen_port > 65535) return false
  if (type === "derp" && (typeof normalized.config_path !== "string" || !normalized.config_path.trim())) return false
  if (type === "ssm-api" && !hasServers(normalized.servers)) return false
  return true
}

function hasServers(value: JsonValue | undefined): boolean {
  return isJsonObject(value) && Object.entries(value).some(([path, tag]) => Boolean(path.trim() && typeof tag === "string" && tag.trim()))
}

function cleanKnownEmptyFields(object: JsonObject): JsonObject {
  let next = object
  for (const field of serviceFields) {
    const value = getPolicyPath(next, field.path)
    const empty = value === "" || Array.isArray(value) && value.length === 0
    if (empty) next = setPolicyPath(next, field.path, undefined)
  }
  return next
}

export function prepareServiceObject(object: JsonObject): JsonObject {
  const normalized = normalizeServiceObject(object)
  return cleanKnownEmptyFields(pruneInvisiblePolicyFields(normalized, serviceFields))
}

export function prepareServices(items: readonly JsonObject[]): JsonObject[] {
  return items.map((item) => prepareServiceObject(item))
}

export function applyServicesConfig(config: SingBoxConfig, items: readonly JsonObject[]): SingBoxConfig {
  const next = { ...config }
  const services = prepareServices(items)
  if (services.length === 0) delete next.services
  else next.services = services
  return next
}

export function changeServiceType(object: JsonObject, type: string): JsonObject {
  const nextType = isServiceType(type) ? type : "resolved"
  const next = createServiceDraft(nextType)
  if (typeof object.tag === "string" && object.tag) next.tag = object.tag
  return next
}

export function summarizeService(object: JsonObject) {
  const normalized = normalizeServiceObject(object)
  const type = typeof normalized.type === "string" ? normalized.type : "unknown"
  const listen = typeof normalized.listen === "string" ? normalized.listen : ""
  const port = typeof normalized.listen_port === "number" && normalized.listen_port > 0
    ? normalized.listen_port : undefined
  const detail = listen ? port ? `${listen}:${port}` : listen : undefined
  const servers = isJsonObject(normalized.servers) ? Object.keys(normalized.servers).length : 0
  return { type, detail, meta: servers }
}
