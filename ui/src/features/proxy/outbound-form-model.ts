import { getPath, type FieldSpec, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"

export const outboundTypes = [
  "direct", "block", "selector", "urltest", "socks", "http", "shadowsocks", "vmess", "vless",
  "trojan", "naive", "hysteria", "hysteria2", "tuic", "ssh", "tor", "shadowtls", "anytls",
] as const

export const serverTypes = new Set(["socks", "http", "shadowsocks", "vmess", "vless", "trojan", "naive", "hysteria", "hysteria2", "tuic", "ssh", "shadowtls", "anytls"])
export const dialerTypes = new Set(["direct", ...serverTypes, "tor"])
export const outboundTLSTypes = new Set(["http", "vmess", "vless", "trojan", "naive", "hysteria", "hysteria2", "tuic", "shadowtls", "anytls"])
export const outboundTransportTypes = new Set(["vmess", "vless", "trojan"])
export const outboundMultiplexTypes = new Set(["shadowsocks", "vmess", "vless", "trojan"])
export const groupTypes = new Set(["selector", "urltest"])

export const dialerFields: FieldSpec[] = [
  { path: "detour", label: "detour" }, { path: "bind_interface", label: "bindInterface", kind: "network-interface" },
  { path: "inet4_bind_address", label: "inet4BindAddress" }, { path: "inet6_bind_address", label: "inet6BindAddress" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean" }, { path: "protect_path", label: "protectPath" },
  { path: "routing_mark", label: "routingMark" }, { path: "reuse_addr", label: "reuseAddress", kind: "boolean" },
  { path: "netns", label: "networkNamespace" }, { path: "connect_timeout", label: "connectTimeout" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean" }, { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean" }, { path: "tcp_keep_alive", label: "tcpKeepAlive" },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval" }, { path: "udp_fragment", label: "udpFragment", kind: "select", options: ["true", "false"] },
  { path: "domain_resolver.server", label: "domainResolverServer" }, { path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select", options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean" }, { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number" },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet" }, { path: "network_strategy", label: "networkStrategy", kind: "select", options: ["default", "fallback", "hybrid"] },
  { path: "network_type", label: "networkType", kind: "list" }, { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list" },
  { path: "fallback_delay", label: "fallbackDelay" },
]

const network = { path: "network", label: "network", kind: "list" } satisfies FieldSpec
const udpOverTCP: FieldSpec[] = [{ path: "udp_over_tcp.enabled", label: "udpOverTCPEnabled", kind: "boolean" }, { path: "udp_over_tcp.version", label: "udpOverTCPVersion", kind: "number" }]
const protocolMap: Record<string, FieldSpec[]> = {
  direct: [], block: [], selector: [], urltest: [],
  socks: [{ path: "version", label: "version", kind: "select", options: ["4", "4a", "5"] }, { path: "username", label: "username" }, { path: "password", label: "password" }, network, ...udpOverTCP],
  http: [{ path: "username", label: "username" }, { path: "password", label: "password" }, { path: "path", label: "httpPath" }, { path: "headers", label: "headers", kind: "json-object" }],
  shadowsocks: [{ path: "method", label: "method" }, { path: "password", label: "password" }, { path: "plugin", label: "plugin" }, { path: "plugin_opts", label: "pluginOptions" }, network, ...udpOverTCP],
  vmess: [{ path: "uuid", label: "uuid" }, { path: "security", label: "security" }, { path: "alter_id", label: "alterId", kind: "number" }, { path: "global_padding", label: "globalPadding", kind: "boolean" }, { path: "authenticated_length", label: "authenticatedLength", kind: "boolean" }, network, { path: "packet_encoding", label: "packetEncoding", kind: "select", options: ["packetaddr", "xudp"] }],
  vless: [{ path: "uuid", label: "uuid" }, { path: "flow", label: "flow" }, network, { path: "packet_encoding", label: "packetEncoding", kind: "select", options: ["packetaddr", "xudp"] }],
  trojan: [{ path: "password", label: "password" }, network],
  naive: [{ path: "username", label: "username" }, { path: "password", label: "password" }, { path: "insecure_concurrency", label: "insecureConcurrency", kind: "number" }, { path: "extra_headers", label: "extraHeaders", kind: "json-object" }, { path: "stream_receive_window", label: "streamReceiveWindow" }, ...udpOverTCP, { path: "quic", label: "quic", kind: "boolean" }, { path: "quic_congestion_control", label: "quicCongestionControl" }, { path: "quic_session_receive_window", label: "quicSessionReceiveWindow" }],
  hysteria: [{ path: "server_ports", label: "serverPorts", kind: "list" }, { path: "hop_interval", label: "hopInterval" }, { path: "up", label: "uploadBandwidth" }, { path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down", label: "downloadBandwidth" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs", label: "obfuscation" }, { path: "auth", label: "auth" }, { path: "auth_str", label: "authString" }, { path: "recv_window_conn", label: "receiveWindowConnection", kind: "number" }, { path: "recv_window", label: "receiveWindow", kind: "number" }, { path: "disable_mtu_discovery", label: "disableMTUDiscovery", kind: "boolean" }, network],
  hysteria2: [{ path: "server_ports", label: "serverPorts", kind: "list" }, { path: "hop_interval", label: "hopInterval" }, { path: "up_mbps", label: "uploadMbps", kind: "number" }, { path: "down_mbps", label: "downloadMbps", kind: "number" }, { path: "obfs.type", label: "obfuscationType", kind: "select", options: ["salamander"] }, { path: "obfs.password", label: "obfuscationPassword" }, { path: "password", label: "password" }, network, { path: "brutal_debug", label: "brutalDebug", kind: "boolean" }],
  tuic: [{ path: "uuid", label: "uuid" }, { path: "password", label: "password" }, { path: "congestion_control", label: "congestionControl" }, { path: "udp_relay_mode", label: "udpRelayMode" }, { path: "udp_over_stream", label: "udpOverStream", kind: "boolean" }, { path: "zero_rtt_handshake", label: "zeroRTTHandshake", kind: "boolean" }, { path: "heartbeat", label: "heartbeat" }, network],
  ssh: [{ path: "user", label: "sshUser" }, { path: "password", label: "password" }, { path: "private_key", label: "privateKey", kind: "textarea" }, { path: "private_key_path", label: "privateKeyPath" }, { path: "private_key_passphrase", label: "privateKeyPassphrase" }, { path: "host_key", label: "hostKey", kind: "list" }, { path: "host_key_algorithms", label: "hostKeyAlgorithms", kind: "list" }, { path: "client_version", label: "clientVersion" }],
  tor: [{ path: "executable_path", label: "executablePath" }, { path: "extra_args", label: "extraArguments", kind: "list" }, { path: "data_directory", label: "dataDirectory" }, { path: "torrc", label: "torOptions", kind: "json-object" }],
  shadowtls: [{ path: "version", label: "version", kind: "number" }, { path: "password", label: "password" }],
  anytls: [{ path: "password", label: "password" }, { path: "idle_session_check_interval", label: "idleSessionCheckInterval" }, { path: "idle_session_timeout", label: "idleSessionTimeout" }, { path: "min_idle_session", label: "minimumIdleSession", kind: "number" }],
}

const groupMap: Record<string, FieldSpec[]> = {
  selector: [{ path: "outbounds", label: "groupOutbounds", kind: "list" }, { path: "default", label: "groupDefault" }, { path: "interrupt_exist_connections", label: "interruptConnections", kind: "boolean" }],
  urltest: [{ path: "outbounds", label: "groupOutbounds", kind: "list" }, { path: "url", label: "urlTestURL" }, { path: "interval", label: "urlTestInterval" }, { path: "tolerance", label: "urlTestTolerance", kind: "number" }, { path: "idle_timeout", label: "idleTimeout" }, { path: "interrupt_exist_connections", label: "interruptConnections", kind: "boolean" }],
}

export const outboundTLSFields: FieldSpec[] = [
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean" }, { path: "tls.disable_sni", label: "disableSNI", kind: "boolean" }, { path: "tls.server_name", label: "serverName" }, { path: "tls.insecure", label: "insecure", kind: "boolean" },
  { path: "tls.alpn", label: "alpn", kind: "list" }, { path: "tls.min_version", label: "minimumTLSVersion" }, { path: "tls.max_version", label: "maximumTLSVersion" }, { path: "tls.cipher_suites", label: "cipherSuites", kind: "list" }, { path: "tls.curve_preferences", label: "curvePreferences", kind: "list" },
  { path: "tls.certificate", label: "certificate", kind: "textarea" }, { path: "tls.certificate_path", label: "certificatePath" }, { path: "tls.certificate_public_key_sha256", label: "certificateSHA256", kind: "list" },
  { path: "tls.client_certificate", label: "clientCertificate", kind: "textarea" }, { path: "tls.client_certificate_path", label: "clientCertificatePath" }, { path: "tls.client_key", label: "clientKey", kind: "textarea" }, { path: "tls.client_key_path", label: "clientKeyPath" },
  { path: "tls.fragment", label: "tlsFragment", kind: "boolean" }, { path: "tls.fragment_fallback_delay", label: "fragmentFallbackDelay" }, { path: "tls.record_fragment", label: "recordFragment", kind: "boolean" }, { path: "tls.kernel_tx", label: "kernelTX", kind: "boolean" }, { path: "tls.kernel_rx", label: "kernelRX", kind: "boolean" },
  { path: "tls.ech.enabled", label: "echEnabled", kind: "boolean" }, { path: "tls.ech.config", label: "echConfig", kind: "list" }, { path: "tls.ech.config_path", label: "echConfigPath" }, { path: "tls.ech.query_server_name", label: "echQueryServerName" },
  { path: "tls.utls.enabled", label: "utlsEnabled", kind: "boolean" }, { path: "tls.utls.fingerprint", label: "utlsFingerprint" },
  { path: "tls.reality.enabled", label: "realityEnabled", kind: "boolean" }, { path: "tls.reality.public_key", label: "realityPublicKey" }, { path: "tls.reality.short_id", label: "realityShortID" },
]

export const outboundMultiplexFields: FieldSpec[] = [
  { path: "multiplex.enabled", label: "multiplexEnabled", kind: "boolean" }, { path: "multiplex.protocol", label: "multiplexProtocol", kind: "select", options: ["smux", "yamux", "h2mux"] },
  { path: "multiplex.max_connections", label: "maxConnections", kind: "number" }, { path: "multiplex.min_streams", label: "minStreams", kind: "number" }, { path: "multiplex.max_streams", label: "maxStreams", kind: "number" }, { path: "multiplex.padding", label: "multiplexPadding", kind: "boolean" },
  { path: "multiplex.brutal.enabled", label: "brutalEnabled", kind: "boolean" }, { path: "multiplex.brutal.up_mbps", label: "uploadMbps", kind: "number" }, { path: "multiplex.brutal.down_mbps", label: "downloadMbps", kind: "number" },
]

const transportBase: FieldSpec[] = [{ path: "transport.type", label: "transportType", kind: "select", options: ["http", "ws", "quic", "grpc", "httpupgrade"] }]
const transportMap: Record<string, FieldSpec[]> = {
  http: [{ path: "transport.host", label: "transportHost", kind: "list" }, { path: "transport.path", label: "transportPath" }, { path: "transport.method", label: "transportMethod" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }],
  ws: [{ path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }, { path: "transport.max_early_data", label: "maxEarlyData", kind: "number" }, { path: "transport.early_data_header_name", label: "earlyDataHeaderName" }],
  quic: [], grpc: [{ path: "transport.service_name", label: "serviceName" }, { path: "transport.idle_timeout", label: "idleTimeout" }, { path: "transport.ping_timeout", label: "pingTimeout" }, { path: "transport.permit_without_stream", label: "permitWithoutStream", kind: "boolean" }],
  httpupgrade: [{ path: "transport.host", label: "transportHost" }, { path: "transport.path", label: "transportPath" }, { path: "transport.headers", label: "transportHeaders", kind: "json-object" }],
}

const credentialPaths = ["username", "password", "uuid", "flow", "security", "auth", "auth_str", "user", "private_key", "private_key_path", "private_key_passphrase", "obfs.password"]
const tlsCredentialPaths = ["tls.client_certificate", "tls.client_certificate_path", "tls.client_key", "tls.client_key_path"]
const knownProtocolFields = [...Object.values(protocolMap).flat(), ...Object.values(groupMap).flat()]
const knownTransportFields = Object.values(transportMap).flat()
export function protocolFields(type: string) { return protocolMap[type] ?? [] }
export function groupFields(type: string) { return groupMap[type] ?? [] }
export function transportTypeFields(type: string) { return [...transportBase, ...(transportMap[type] ?? [])] }

function removeFields(object: JsonObject, fields: FieldSpec[]) { return fields.reduce((next, field) => setPath(next, field.path, undefined), object) }
function removePaths(object: JsonObject, paths: string[]) { return paths.reduce((next, path) => setPath(next, path, undefined), object) }
function matchesField(value: unknown, field: FieldSpec) {
  if (value === undefined) return true
  if (field.kind === "boolean") return typeof value === "boolean"
  if (field.kind === "number") return typeof value === "number"
  if (field.kind === "list") return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  /* c8 ignore next 2 - outbound fields currently never use number-list/users kinds */
  if (field.kind === "number-list") return typeof value === "number" || Array.isArray(value) && value.every((item) => typeof item === "number")
  if (field.kind === "json-object") return Boolean(value && typeof value === "object" && !Array.isArray(value))
  /* c8 ignore next */
  if (field.kind === "users") return Array.isArray(value) && value.every((item) => Boolean(item && typeof item === "object" && !Array.isArray(item)))
  return typeof value === "string"
}
function removeIncompatibleFields(object: JsonObject, known: FieldSpec[], target: FieldSpec[]) {
  const paths = [...new Set(known.map((field) => field.path))]
  return paths.reduce((next, path) => {
    const field = target.find((candidate) => candidate.path === path)
    return field && matchesField(getPath(next, path), field) ? next : setPath(next, path, undefined)
  }, object)
}

export function changeOutboundTransportType(object: JsonObject, type: string) {
  const next = removeIncompatibleFields(object, knownTransportFields, transportMap[type] ?? [])
  return setPath(next, "transport.type", type || undefined)
}

export function changeOutboundType(object: JsonObject, type: string) {
  const previous = String(object.type ?? "")
  if (previous === type) return object
  let next = { ...object }
  if (!serverTypes.has(type)) next = setPath(setPath(next, "server", undefined), "server_port", undefined)
  if (!dialerTypes.has(type)) next = removeFields(next, dialerFields)
  next = removeIncompatibleFields(next, knownProtocolFields, [...protocolFields(type), ...groupFields(type)])
  next = removePaths(next, credentialPaths)
  next = removePaths(next, tlsCredentialPaths)
  if (!outboundTLSTypes.has(type)) next = setPath(next, "tls", undefined)
  if (!outboundTransportTypes.has(type)) next = setPath(next, "transport", undefined)
  if (!outboundMultiplexTypes.has(type)) next = setPath(next, "multiplex", undefined)
  return { ...next, type }
}
