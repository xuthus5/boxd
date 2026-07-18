import { getPath, pruneInvisibleFields, type FieldSpec, type JsonObject, setPath, visibleFields } from "@/features/proxy/proxy-form-model"

export { getPath, pruneInvisibleFields, setPath, visibleFields }
export type { FieldSpec, JsonObject }

export const inboundTypes = [
  "mixed", "socks", "http", "direct", "shadowsocks", "vmess", "vless", "trojan", "naive",
  "hysteria", "hysteria2", "tuic", "shadowtls", "anytls", "redirect", "tproxy", "tun",
] as const

export const listenFields: FieldSpec[] = [
  { path: "listen", label: "listenAddress", kind: "listen-address", section: "bind" },
  { path: "listen_port", label: "listenPort", kind: "number", section: "bind" },
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "bind" },
  { path: "detour", label: "detour", kind: "ref", ref: "inbound", section: "bind" },
  { path: "routing_mark", label: "routingMark", section: "bind" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "bind" },
  { path: "netns", label: "networkNamespace", section: "bind" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "tcp" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "tcp" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "tcp" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "tcp", when: { path: "disable_tcp_keep_alive", falsy: true } },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "tcp", when: { path: "disable_tcp_keep_alive", falsy: true } },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "udp" },
  { path: "udp_timeout", label: "udpTimeout", section: "udp" },
]

const network = { path: "network", label: "network", kind: "network-multi", section: "protocol" } satisfies FieldSpec
const protocolMap: Record<string, FieldSpec[]> = {
  mixed: [{ path: "users", label: "users", kind: "users", section: "auth" }, { path: "domain_resolver", label: "domainResolver", kind: "ref", ref: "dns-server", section: "protocol" }, { path: "set_system_proxy", label: "setSystemProxy", kind: "boolean", section: "protocol" }],
  socks: [{ path: "users", label: "users", kind: "users", section: "auth" }, { path: "domain_resolver", label: "domainResolver", kind: "ref", ref: "dns-server", section: "protocol" }],
  http: [{ path: "users", label: "users", kind: "users", section: "auth" }, { path: "domain_resolver", label: "domainResolver", kind: "ref", ref: "dns-server", section: "protocol" }, { path: "set_system_proxy", label: "setSystemProxy", kind: "boolean", section: "protocol" }],
  direct: [network, { path: "override_address", label: "overrideAddress" }, { path: "override_port", label: "overridePort", kind: "number" }],
  shadowsocks: [network, { path: "method", label: "method", kind: "select", options: ["2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305", "aes-128-gcm", "aes-192-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305", "none"], section: "auth" }, { path: "password", label: "password", section: "auth" }, { path: "users", label: "users", kind: "users", section: "auth" }, { path: "managed", label: "managed", kind: "boolean", section: "protocol" }],
  vmess: [{ path: "users", label: "users", kind: "users", section: "auth" }],
  vless: [{ path: "users", label: "users", kind: "users", section: "auth" }],
  trojan: [{ path: "users", label: "users", kind: "users", section: "auth" }],
  naive: [{ path: "users", label: "users", kind: "users", section: "auth" }, network, { path: "quic_congestion_control", label: "congestionControl" }],
  hysteria: [{ path: "up", label: "uploadBandwidth" }, { path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down", label: "downloadBandwidth" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs", label: "obfuscation" }, { path: "users", label: "users", kind: "users", section: "auth" }, { path: "recv_window_conn", label: "receiveWindowConnection", kind: "number" }, { path: "recv_window_client", label: "receiveWindowClient", kind: "number" }, { path: "max_conn_client", label: "maxClientConnections", kind: "number" }, { path: "disable_mtu_discovery", label: "disableMTUDiscovery", kind: "boolean" }],
  hysteria2: [{ path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs.type", label: "obfuscationType", kind: "select", options: ["", "salamander"] }, { path: "obfs.password", label: "obfuscationPassword", when: { path: "obfs.type", is: "salamander" } }, { path: "users", label: "users", kind: "users", section: "auth" }, { path: "ignore_client_bandwidth", label: "ignoreClientBandwidth", kind: "boolean" }, { path: "masquerade", label: "masquerade" }, { path: "brutal_debug", label: "brutalDebug", kind: "boolean" }],
  tuic: [{ path: "users", label: "users", kind: "users", section: "auth" }, { path: "congestion_control", label: "congestionControl" }, { path: "auth_timeout", label: "authenticationTimeout" }, { path: "zero_rtt_handshake", label: "zeroRTTHandshake", kind: "boolean" }, { path: "heartbeat", label: "heartbeat" }],
  shadowtls: [{ path: "version", label: "version", kind: "number" }, { path: "password", label: "password" }, { path: "users", label: "users", kind: "users", section: "auth" }, { path: "handshake.server", label: "handshakeServer" }, { path: "handshake.server_port", label: "handshakePort", kind: "number" }, { path: "strict_mode", label: "strictMode", kind: "boolean" }, { path: "wildcard_sni", label: "wildcardSNI", kind: "select", options: ["", "off", "authed", "all"] }],
  anytls: [{ path: "users", label: "users", kind: "users", section: "auth" }, { path: "padding_scheme", label: "paddingScheme" }],
  tproxy: [network], redirect: [], tun: [],
}

export const tunFields: FieldSpec[] = [
  { path: "interface_name", label: "interfaceName", section: "tunBasic" }, { path: "mtu", label: "mtu", kind: "number", section: "tunBasic" }, { path: "address", label: "tunAddress", kind: "list", section: "tunBasic" },
  { path: "stack", label: "stack", kind: "select", options: ["", "system", "gvisor", "mixed"], section: "tunBasic" }, { path: "auto_route", label: "autoRoute", kind: "boolean", section: "tunRoute" },
  { path: "iproute2_table_index", label: "ipRouteTableIndex", kind: "number", when: { path: "auto_route", is: true }, section: "tunRoute" }, { path: "iproute2_rule_index", label: "ipRouteRuleIndex", kind: "number", section: "tunRoute", when: { path: "auto_route", is: true } },
  { path: "auto_redirect", label: "autoRedirect", kind: "boolean", section: "tunRedirect" }, { path: "auto_redirect_input_mark", label: "autoRedirectInputMark", when: { path: "auto_redirect", is: true }, section: "tunRedirect" },
  { path: "auto_redirect_output_mark", label: "autoRedirectOutputMark", when: { path: "auto_redirect", is: true }, section: "tunRedirect" }, { path: "auto_redirect_reset_mark", label: "autoRedirectResetMark", when: { path: "auto_redirect", is: true }, section: "tunRedirect" },
  { path: "auto_redirect_nfqueue", label: "autoRedirectNFQueue", kind: "number", when: { path: "auto_redirect", is: true }, section: "tunRedirect" }, { path: "auto_redirect_iproute2_fallback_rule_index", label: "autoRedirectFallbackRuleIndex", kind: "number", when: { path: "auto_redirect", is: true }, section: "tunRedirect" },
  { path: "loopback_address", label: "loopbackAddress", kind: "list", section: "tunRoute" }, { path: "strict_route", label: "strictRoute", kind: "boolean", section: "tunRoute" },
  { path: "route_address", label: "routeAddress", kind: "list", section: "tunRoute" }, { path: "route_address_set", label: "routeAddressSet", kind: "list", section: "tunRoute" },
  { path: "route_exclude_address", label: "routeExcludeAddress", kind: "list", section: "tunRoute" }, { path: "route_exclude_address_set", label: "routeExcludeAddressSet", kind: "list", section: "tunRoute" },
  { path: "include_interface", label: "includeInterface", kind: "ref", ref: "network-interface-multi", section: "tunFilter" }, { path: "exclude_interface", label: "excludeInterface", kind: "ref", ref: "network-interface-multi", section: "tunFilter" },
  { path: "include_uid", label: "includeUID", kind: "number-list", section: "tunFilter" }, { path: "exclude_uid", label: "excludeUID", kind: "number-list", section: "tunFilter" },
  { path: "include_uid_range", label: "includeUIDRange", kind: "list", section: "tunFilter" }, { path: "exclude_uid_range", label: "excludeUIDRange", kind: "list", section: "tunFilter" },
  { path: "include_android_user", label: "includeAndroidUser", kind: "number-list", section: "tunFilter" }, { path: "include_package", label: "includePackage", kind: "list", section: "tunFilter" }, { path: "exclude_package", label: "excludePackage", kind: "list", section: "tunFilter" },
  { path: "udp_timeout", label: "udpTimeout", section: "tunBasic" }, { path: "exclude_mptcp", label: "excludeMPTCP", kind: "boolean", section: "tunFilter" },
]

const tlsOn = { path: "tls.enabled", is: true } as const
const clientAuthOn = [tlsOn, { path: "tls.client_authentication", is: ["request", "require-any", "verify-if-given", "require-and-verify"] }] as const
const acmeOn = [tlsOn, { path: "tls.acme.domain" }] as const
const echOn = [tlsOn, { path: "tls.ech.enabled", is: true }] as const
const realityOn = [tlsOn, { path: "tls.reality.enabled", is: true }] as const

export const tlsFields: FieldSpec[] = [
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean", section: "tlsBasic" },
  { path: "tls.server_name", label: "serverName", when: tlsOn, section: "tlsBasic" },
  { path: "tls.insecure", label: "insecure", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.alpn", label: "alpn", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.min_version", label: "minimumTLSVersion", when: tlsOn, section: "tlsBasic" },
  { path: "tls.max_version", label: "maximumTLSVersion", when: tlsOn, section: "tlsBasic" },
  { path: "tls.cipher_suites", label: "cipherSuites", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.curve_preferences", label: "curvePreferences", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.certificate", label: "certificate", kind: "textarea", when: tlsOn, section: "tlsCert" },
  { path: "tls.certificate_path", label: "certificatePath", when: tlsOn, section: "tlsCert" },
  { path: "tls.key", label: "privateKey", kind: "textarea", when: tlsOn, section: "tlsCert" },
  { path: "tls.key_path", label: "keyPath", when: tlsOn, section: "tlsCert" },
  { path: "tls.client_authentication", label: "clientAuthentication", kind: "select", options: ["no", "request", "require-any", "verify-if-given", "require-and-verify"], when: tlsOn, section: "tlsClient" },
  { path: "tls.client_certificate", label: "clientCertificate", kind: "textarea", when: [...clientAuthOn], section: "tlsClient" },
  { path: "tls.client_certificate_path", label: "clientCertificatePath", kind: "list", when: [...clientAuthOn], section: "tlsClient" },
  { path: "tls.client_certificate_public_key_sha256", label: "clientCertificateSHA256", kind: "list", when: [...clientAuthOn], section: "tlsClient" },
  { path: "tls.kernel_tx", label: "kernelTX", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.kernel_rx", label: "kernelRX", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.acme.domain", label: "acmeDomain", kind: "list", when: tlsOn, section: "tlsAcme" },
  { path: "tls.acme.data_directory", label: "acmeDataDirectory", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.default_server_name", label: "acmeDefaultServerName", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.email", label: "acmeEmail", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.provider", label: "acmeProvider", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.disable_http_challenge", label: "disableHTTPChallenge", kind: "boolean", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.disable_tls_alpn_challenge", label: "disableTLSALPNChallenge", kind: "boolean", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.alternative_http_port", label: "alternativeHTTPPort", kind: "number", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.acme.alternative_tls_port", label: "alternativeTLSPort", kind: "number", when: [...acmeOn], section: "tlsAcme" },
  { path: "tls.ech.enabled", label: "echEnabled", kind: "boolean", when: tlsOn, section: "tlsEch" },
  { path: "tls.ech.key", label: "echKey", kind: "list", when: [...echOn], section: "tlsEch" },
  { path: "tls.ech.key_path", label: "echKeyPath", when: [...echOn], section: "tlsEch" },
  { path: "tls.reality.enabled", label: "realityEnabled", kind: "boolean", when: tlsOn, section: "tlsReality" },
  { path: "tls.reality.handshake.server", label: "realityHandshakeServer", when: [...realityOn], section: "tlsReality" },
  { path: "tls.reality.handshake.server_port", label: "realityHandshakePort", kind: "number", when: [...realityOn], section: "tlsReality" },
  { path: "tls.reality.private_key", label: "realityPrivateKey", when: [...realityOn], section: "tlsReality" },
  { path: "tls.reality.short_id", label: "realityShortID", kind: "list", when: [...realityOn], section: "tlsReality" },
  { path: "tls.reality.max_time_difference", label: "realityMaxTimeDifference", when: [...realityOn], section: "tlsReality" },
]

export const transportFields: FieldSpec[] = [
  { path: "transport.type", label: "transportType", kind: "select", options: ["", "http", "ws", "quic", "grpc", "httpupgrade"], section: "transport" },
]
const transportByType: Record<string, FieldSpec[]> = {
  http: [{ path: "transport.host", label: "transportHost", kind: "list" }, { path: "transport.path", label: "transportPath" }, { path: "transport.method", label: "transportMethod" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }],
  ws: [{ path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.max_early_data", label: "maxEarlyData", kind: "number" }, { path: "transport.early_data_header_name", label: "earlyDataHeaderName" }],
  quic: [],
  grpc: [{ path: "transport.service_name", label: "serviceName" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }, { path: "transport.permit_without_stream", label: "permitWithoutStream", kind: "boolean" }],
  httpupgrade: [{ path: "transport.host", label: "transportHost" }, { path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }],
}
export const multiplexFields: FieldSpec[] = [
  { path: "multiplex.enabled", label: "multiplexEnabled", kind: "boolean", section: "multiplex" },
  { path: "multiplex.padding", label: "multiplexPadding", kind: "boolean", when: { path: "multiplex.enabled", is: true }, section: "multiplex" },
  { path: "multiplex.brutal.enabled", label: "brutalEnabled", kind: "boolean", when: { path: "multiplex.enabled", is: true }, section: "multiplex" },
  { path: "multiplex.brutal.up_mbps", label: "uploadMbps", kind: "number", when: [{ path: "multiplex.enabled", is: true }, { path: "multiplex.brutal.enabled", is: true }], section: "multiplex" },
  { path: "multiplex.brutal.down_mbps", label: "downloadMbps", kind: "number", when: [{ path: "multiplex.enabled", is: true }, { path: "multiplex.brutal.enabled", is: true }], section: "multiplex" },
]

export const tlsTypes = new Set(["http", "mixed", "vmess", "vless", "trojan", "naive", "hysteria", "hysteria2", "tuic", "anytls"])
export const transportTypes = new Set(["vmess", "vless", "trojan"])
export const multiplexTypes = new Set(["shadowsocks", "vmess", "vless", "trojan"])
export function protocolFields(type: string) { return protocolMap[type] ?? [] }
export function transportTypeFields(type: string) { return [...transportFields, ...(transportByType[type] ?? [])] }

function removeFields(object: JsonObject, fields: FieldSpec[]) {
  return fields.reduce((next, field) => setPath(next, field.path, undefined), object)
}

export function changeTransportType(object: JsonObject, type: string) {
  const previous = String(getPath(object, "transport.type") ?? "")
  let next = removeFields(object, (transportByType[previous] ?? []).filter((field) => !(transportByType[type] ?? []).some((candidate) => candidate.path === field.path && candidate.kind === field.kind)))
  next = setPath(next, "transport.type", type || undefined)
  return next
}

export function changeInboundType(object: JsonObject, type: string) {
  const previous = String(object.type ?? "")
  if (previous === type) return object
  let next = { ...object }
  if (previous === "tun" && type !== "tun") next = removeFields(next, tunFields.filter((field) => field.path !== "udp_timeout"))
  if (previous !== "tun" && type === "tun") next = removeFields(next, listenFields.filter((field) => field.path !== "udp_timeout"))
  next = removeFields(next, protocolFields(previous).filter((field) => field.path === "users" || !protocolFields(type).some((candidate) => candidate.path === field.path && candidate.kind === field.kind)))
  if (tlsTypes.has(previous) && !tlsTypes.has(type)) next = setPath(next, "tls", undefined)
  if (transportTypes.has(previous) && !transportTypes.has(type)) next = setPath(next, "transport", undefined)
  if (multiplexTypes.has(previous) && !multiplexTypes.has(type)) next = setPath(next, "multiplex", undefined)
  return { ...next, type }
}

export function managedInboundFields(type: string, transportType = "") {
  const fields: FieldSpec[] = [...listenFields, ...protocolFields(type), ...tunFields]
  if (tlsTypes.has(type)) fields.push(...tlsFields)
  if (transportTypes.has(type)) fields.push(...transportTypeFields(transportType))
  if (multiplexTypes.has(type)) fields.push(...multiplexFields)
  return fields
}

export function applyInboundFieldChange(object: JsonObject, next: JsonObject, typeHint = "") {
  const type = String(next.type ?? object.type ?? typeHint ?? "")
  const transportType = String(getPath(next, "transport.type") ?? getPath(object, "transport.type") ?? "")
  return pruneInvisibleFields(next, managedInboundFields(type, transportType))
}

export type UserFieldKey = "name" | "username" | "password" | "uuid" | "flow" | "alterId"

export function userSchema(type: string): UserFieldKey[] {
  switch (type) {
    case "mixed":
    case "socks":
    case "http":
    case "naive":
    case "anytls":
      return ["username", "password"]
    case "shadowsocks":
    case "trojan":
    case "shadowtls":
      return ["name", "password"]
    case "vmess":
      return ["name", "uuid", "alterId"]
    case "vless":
      return ["name", "uuid", "flow"]
    case "hysteria":
    case "hysteria2":
      return ["name", "password"]
    case "tuic":
      return ["name", "uuid", "password"]
    default:
      return ["name", "password"]
  }
}
