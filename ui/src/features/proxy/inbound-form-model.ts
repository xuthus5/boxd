import { getPath, type FieldSpec, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"

export { getPath, setPath }
export type { FieldSpec, JsonObject }

export const inboundTypes = [
  "mixed", "socks", "http", "direct", "shadowsocks", "vmess", "vless", "trojan", "naive",
  "hysteria", "hysteria2", "tuic", "shadowtls", "anytls", "redirect", "tproxy", "tun",
] as const

export const listenFields: FieldSpec[] = [
  { path: "listen", label: "listenAddress" }, { path: "listen_port", label: "listenPort", kind: "number" },
  { path: "bind_interface", label: "bindInterface" }, { path: "routing_mark", label: "routingMark" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean" }, { path: "netns", label: "networkNamespace" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean" }, { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean" }, { path: "tcp_keep_alive", label: "tcpKeepAlive" },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval" }, { path: "udp_fragment", label: "udpFragment", kind: "select", options: ["true", "false"] },
  { path: "udp_timeout", label: "udpTimeout" }, { path: "detour", label: "detour" },
]

const network = { path: "network", label: "network", kind: "list" } satisfies FieldSpec
const protocolMap: Record<string, FieldSpec[]> = {
  mixed: [{ path: "users", label: "users", kind: "users" }, { path: "domain_resolver", label: "domainResolver" }, { path: "set_system_proxy", label: "setSystemProxy", kind: "boolean" }],
  socks: [{ path: "users", label: "users", kind: "users" }, { path: "domain_resolver", label: "domainResolver" }],
  http: [{ path: "users", label: "users", kind: "users" }, { path: "domain_resolver", label: "domainResolver" }, { path: "set_system_proxy", label: "setSystemProxy", kind: "boolean" }],
  direct: [network, { path: "override_address", label: "overrideAddress" }, { path: "override_port", label: "overridePort", kind: "number" }],
  shadowsocks: [network, { path: "method", label: "method" }, { path: "password", label: "password" }, { path: "users", label: "users", kind: "users" }, { path: "managed", label: "managed", kind: "boolean" }],
  vmess: [{ path: "users", label: "users", kind: "users" }],
  vless: [{ path: "users", label: "users", kind: "users" }],
  trojan: [{ path: "users", label: "users", kind: "users" }],
  naive: [{ path: "users", label: "users", kind: "users" }, network, { path: "quic_congestion_control", label: "congestionControl" }],
  hysteria: [{ path: "up", label: "uploadBandwidth" }, { path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down", label: "downloadBandwidth" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs", label: "obfuscation" }, { path: "users", label: "users", kind: "users" }, { path: "recv_window_conn", label: "receiveWindowConnection", kind: "number" }, { path: "recv_window_client", label: "receiveWindowClient", kind: "number" }, { path: "max_conn_client", label: "maxClientConnections", kind: "number" }, { path: "disable_mtu_discovery", label: "disableMTUDiscovery", kind: "boolean" }],
  hysteria2: [{ path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs.type", label: "obfuscationType", kind: "select", options: ["", "salamander"] }, { path: "obfs.password", label: "obfuscationPassword" }, { path: "users", label: "users", kind: "users" }, { path: "ignore_client_bandwidth", label: "ignoreClientBandwidth", kind: "boolean" }, { path: "masquerade", label: "masquerade" }, { path: "brutal_debug", label: "brutalDebug", kind: "boolean" }],
  tuic: [{ path: "users", label: "users", kind: "users" }, { path: "congestion_control", label: "congestionControl" }, { path: "auth_timeout", label: "authenticationTimeout" }, { path: "zero_rtt_handshake", label: "zeroRTTHandshake", kind: "boolean" }, { path: "heartbeat", label: "heartbeat" }],
  shadowtls: [{ path: "version", label: "version", kind: "number" }, { path: "password", label: "password" }, { path: "users", label: "users", kind: "users" }, { path: "handshake.server", label: "handshakeServer" }, { path: "handshake.server_port", label: "handshakePort", kind: "number" }, { path: "strict_mode", label: "strictMode", kind: "boolean" }, { path: "wildcard_sni", label: "wildcardSNI", kind: "select", options: ["", "off", "authed", "all"] }],
  anytls: [{ path: "users", label: "users", kind: "users" }, { path: "padding_scheme", label: "paddingScheme" }],
  tproxy: [network], redirect: [], tun: [],
}

export const tunFields: FieldSpec[] = [
  { path: "interface_name", label: "interfaceName" }, { path: "mtu", label: "mtu", kind: "number" }, { path: "address", label: "tunAddress", kind: "list" },
  { path: "stack", label: "stack", kind: "select", options: ["", "system", "gvisor", "mixed"] }, { path: "auto_route", label: "autoRoute", kind: "boolean" },
  { path: "iproute2_table_index", label: "ipRouteTableIndex", kind: "number" }, { path: "iproute2_rule_index", label: "ipRouteRuleIndex", kind: "number" },
  { path: "auto_redirect", label: "autoRedirect", kind: "boolean" }, { path: "auto_redirect_input_mark", label: "autoRedirectInputMark" },
  { path: "auto_redirect_output_mark", label: "autoRedirectOutputMark" }, { path: "auto_redirect_reset_mark", label: "autoRedirectResetMark" },
  { path: "auto_redirect_nfqueue", label: "autoRedirectNFQueue", kind: "number" }, { path: "auto_redirect_iproute2_fallback_rule_index", label: "autoRedirectFallbackRuleIndex", kind: "number" },
  { path: "loopback_address", label: "loopbackAddress", kind: "list" }, { path: "strict_route", label: "strictRoute", kind: "boolean" },
  { path: "route_address", label: "routeAddress", kind: "list" }, { path: "route_address_set", label: "routeAddressSet", kind: "list" },
  { path: "route_exclude_address", label: "routeExcludeAddress", kind: "list" }, { path: "route_exclude_address_set", label: "routeExcludeAddressSet", kind: "list" },
  { path: "include_interface", label: "includeInterface", kind: "list" }, { path: "exclude_interface", label: "excludeInterface", kind: "list" },
  { path: "include_uid", label: "includeUID", kind: "number-list" }, { path: "exclude_uid", label: "excludeUID", kind: "number-list" },
  { path: "include_uid_range", label: "includeUIDRange", kind: "list" }, { path: "exclude_uid_range", label: "excludeUIDRange", kind: "list" },
  { path: "include_android_user", label: "includeAndroidUser", kind: "number-list" }, { path: "include_package", label: "includePackage", kind: "list" }, { path: "exclude_package", label: "excludePackage", kind: "list" },
  { path: "udp_timeout", label: "udpTimeout" }, { path: "exclude_mptcp", label: "excludeMPTCP", kind: "boolean" },
]

export const tlsFields: FieldSpec[] = [
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean" }, { path: "tls.server_name", label: "serverName" },
  { path: "tls.insecure", label: "insecure", kind: "boolean" }, { path: "tls.alpn", label: "alpn", kind: "list" },
  { path: "tls.min_version", label: "minimumTLSVersion" }, { path: "tls.max_version", label: "maximumTLSVersion" },
  { path: "tls.cipher_suites", label: "cipherSuites", kind: "list" }, { path: "tls.curve_preferences", label: "curvePreferences", kind: "list" },
  { path: "tls.certificate", label: "certificate", kind: "textarea" }, { path: "tls.certificate_path", label: "certificatePath" },
  { path: "tls.key", label: "privateKey", kind: "textarea" }, { path: "tls.key_path", label: "keyPath" },
  { path: "tls.client_authentication", label: "clientAuthentication", kind: "select", options: ["no", "request", "require-any", "verify-if-given", "require-and-verify"] },
  { path: "tls.client_certificate", label: "clientCertificate", kind: "textarea" }, { path: "tls.client_certificate_path", label: "clientCertificatePath", kind: "list" },
  { path: "tls.client_certificate_public_key_sha256", label: "clientCertificateSHA256", kind: "list" }, { path: "tls.kernel_tx", label: "kernelTX", kind: "boolean" }, { path: "tls.kernel_rx", label: "kernelRX", kind: "boolean" },
  { path: "tls.acme.domain", label: "acmeDomain", kind: "list" }, { path: "tls.acme.data_directory", label: "acmeDataDirectory" },
  { path: "tls.acme.default_server_name", label: "acmeDefaultServerName" }, { path: "tls.acme.email", label: "acmeEmail" }, { path: "tls.acme.provider", label: "acmeProvider" },
  { path: "tls.acme.disable_http_challenge", label: "disableHTTPChallenge", kind: "boolean" }, { path: "tls.acme.disable_tls_alpn_challenge", label: "disableTLSALPNChallenge", kind: "boolean" },
  { path: "tls.acme.alternative_http_port", label: "alternativeHTTPPort", kind: "number" }, { path: "tls.acme.alternative_tls_port", label: "alternativeTLSPort", kind: "number" },
  { path: "tls.ech.enabled", label: "echEnabled", kind: "boolean" }, { path: "tls.ech.key", label: "echKey", kind: "list" }, { path: "tls.ech.key_path", label: "echKeyPath" },
  { path: "tls.reality.enabled", label: "realityEnabled", kind: "boolean" }, { path: "tls.reality.handshake.server", label: "realityHandshakeServer" },
  { path: "tls.reality.handshake.server_port", label: "realityHandshakePort", kind: "number" }, { path: "tls.reality.private_key", label: "realityPrivateKey" },
  { path: "tls.reality.short_id", label: "realityShortID", kind: "list" }, { path: "tls.reality.max_time_difference", label: "realityMaxTimeDifference" },
]

export const transportFields: FieldSpec[] = [
  { path: "transport.type", label: "transportType", kind: "select", options: ["", "http", "ws", "quic", "grpc", "httpupgrade"] },
]
const transportByType: Record<string, FieldSpec[]> = {
  http: [{ path: "transport.host", label: "transportHost", kind: "list" }, { path: "transport.path", label: "transportPath" }, { path: "transport.method", label: "transportMethod" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }],
  ws: [{ path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.max_early_data", label: "maxEarlyData", kind: "number" }, { path: "transport.early_data_header_name", label: "earlyDataHeaderName" }],
  quic: [],
  grpc: [{ path: "transport.service_name", label: "serviceName" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }, { path: "transport.permit_without_stream", label: "permitWithoutStream", kind: "boolean" }],
  httpupgrade: [{ path: "transport.host", label: "transportHost" }, { path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }],
}
export const multiplexFields: FieldSpec[] = [
  { path: "multiplex.enabled", label: "multiplexEnabled", kind: "boolean" }, { path: "multiplex.padding", label: "multiplexPadding", kind: "boolean" },
  { path: "multiplex.brutal.enabled", label: "brutalEnabled", kind: "boolean" }, { path: "multiplex.brutal.up_mbps", label: "uploadMbps", kind: "number" },
  { path: "multiplex.brutal.down_mbps", label: "downloadMbps", kind: "number" },
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
