import { endpointDialerFields } from "@/features/advanced/endpoints-form-model"
import { http2Fields, quicFields } from "@/features/config/kernel114-fields"
import { isJsonObject, type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

export type SharedResourceSection = "network_namespaces" | "certificate_providers" | "http_clients"

const resourceTypes: Record<SharedResourceSection, readonly string[]> = {
  network_namespaces: ["default", "unshare"],
  certificate_providers: ["acme", "tailscale", "cloudflare-origin-ca"],
  http_clients: [],
}

const certificateFields: Record<string, readonly PolicyFieldSpec[]> = {
  acme: [
    { path: "domain", label: "certificateDomains", kind: "list" },
    { path: "data_directory", label: "dataDirectory" },
    { path: "default_server_name", label: "defaultServerName" },
    { path: "email", label: "acmeEmail" },
    { path: "provider", label: "acmeProvider" },
    { path: "account_key", label: "acmeAccountKey" },
    { path: "key_type", label: "certificateKeyType", kind: "select", options: ["ed25519", "p256", "p384", "rsa2048", "rsa4096"] },
    { path: "profile", label: "acmeProfile" },
    { path: "dns01_challenge", label: "dns01Challenge", kind: "json-object" },
    { path: "external_account", label: "externalAccount", kind: "json-object" },
    { path: "disable_http_challenge", label: "disableHTTPChallenge", kind: "boolean" },
    { path: "disable_tls_alpn_challenge", label: "disableTLSALPNChallenge", kind: "boolean" },
    { path: "alternative_http_port", label: "alternativeHTTPPort", kind: "number" },
    { path: "alternative_tls_port", label: "alternativeTLSPort", kind: "number" },
    { path: "http_client", label: "httpClient", kind: "json-value" },
  ],
  tailscale: [{ path: "endpoint", label: "endpoint", required: true }],
  "cloudflare-origin-ca": [
    { path: "domain", label: "certificateDomains", kind: "list" },
    { path: "data_directory", label: "dataDirectory" },
    { path: "api_token", label: "apiToken" },
    { path: "origin_ca_key", label: "originCAKey" },
    { path: "request_type", label: "certificateRequestType", kind: "select", options: ["origin-rsa", "origin-ecc"] },
    { path: "requested_validity", label: "requestedValidity", kind: "number" },
    { path: "http_client", label: "httpClient", kind: "json-value" },
  ],
}

const httpClientFields: readonly PolicyFieldSpec[] = [
  { path: "engine", label: "httpEngine", kind: "select", options: ["go", "apple"] },
  { path: "version", label: "httpVersion", kind: "select", options: ["1", "2", "3"] },
  { path: "disable_version_fallback", label: "disableVersionFallback", kind: "boolean" },
  { path: "headers", label: "httpHeaders", kind: "json-object" },
  { path: "tls", label: "httpTLS", kind: "json-object" },
  ...endpointDialerFields,
]

export function sharedResourceFields(section: SharedResourceSection, item: JsonObject): readonly PolicyFieldSpec[] {
  const identity: PolicyFieldSpec[] = [{ path: "tag", label: "tag", required: true }]
  if (resourceTypes[section].length) identity.push({ path: "type", label: "type", kind: "select", options: resourceTypes[section], required: true })
  if (section === "network_namespaces") return [...identity, item.type === "unshare"
    ? { path: "pid_file", label: "pidFile" }
    : { path: "path", label: "namespacePath", required: true }]
  if (section === "certificate_providers") return [...identity, ...(certificateFields[String(item.type)] ?? [])]
  const transport = item.version === 1 ? [] : item.version === 3 ? quicFields : http2Fields
  return [...identity, ...httpClientFields, ...transport]
}

export function createSharedResource(section: SharedResourceSection): JsonObject {
  return section === "http_clients" ? { tag: "", engine: "go", version: 2 } : { type: resourceTypes[section][0], tag: "" }
}

export function parseSharedResources(raw: string): JsonObject[] | null {
  try {
    const value: JsonValue = JSON.parse(raw)
    return Array.isArray(value) && value.every(isJsonObject) ? value : null
  } catch {
    return null
  }
}

export function isSharedResourceReady(section: SharedResourceSection, item: JsonObject): boolean {
  if (typeof item.tag !== "string" || !item.tag.trim()) return false
  if (section === "http_clients") return true
  if (!resourceTypes[section].includes(String(item.type))) return false
  if (section === "network_namespaces" && item.type === "default") return typeof item.path === "string" && Boolean(item.path.trim())
  if (section === "certificate_providers" && item.type === "tailscale") return typeof item.endpoint === "string" && Boolean(item.endpoint.trim())
  return true
}

export function updateSharedResource(section: SharedResourceSection, previous: JsonObject, next: JsonObject): JsonObject {
  if (previous.type !== next.type && section !== "http_clients") return { type: next.type, tag: next.tag ?? "" }
  if (section === "http_clients" && previous.version !== next.version) {
    const result = { ...next }
    const supported = new Set(sharedResourceFields(section, result).map((field) => field.path))
    for (const field of quicFields) if (!supported.has(field.path)) delete result[field.path]
    return result
  }
  return next
}
