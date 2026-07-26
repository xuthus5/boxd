import { getPath, pruneInvisibleFields, type FieldSpec, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"

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

const tlsVersions = ["1.0", "1.1", "1.2", "1.3"]
const ssMethods = [
  "2022-blake3-aes-128-gcm", "2022-blake3-aes-256-gcm", "2022-blake3-chacha20-poly1305",
  "aes-128-gcm", "aes-192-gcm", "aes-256-gcm", "chacha20-ietf-poly1305", "xchacha20-ietf-poly1305", "none",
]

const tlsOn = { path: "tls.enabled", is: true } as const
const utlsOn = [tlsOn, { path: "tls.utls.enabled", is: true }] as const
const realityOn = [tlsOn, { path: "tls.reality.enabled", is: true }] as const
const echOn = [tlsOn, { path: "tls.ech.enabled", is: true }] as const
const fragmentOn = [tlsOn, { path: "tls.fragment", is: true }] as const
const muxOn = { path: "multiplex.enabled", is: true } as const
const brutalOn = [muxOn, { path: "multiplex.brutal.enabled", is: true }] as const
const udpOverTCPOn = { path: "udp_over_tcp.enabled", is: true } as const
const keepAliveOn = { path: "disable_tcp_keep_alive", falsy: true } as const

export const dialerFields: FieldSpec[] = [
  { path: "detour", label: "detour", kind: "ref", ref: "outbound", section: "bind" },
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "bind" },
  { path: "inet4_bind_address", label: "inet4BindAddress", section: "bind" },
  { path: "inet6_bind_address", label: "inet6BindAddress", section: "bind" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean", section: "bind" },
  { path: "protect_path", label: "protectPath", section: "bind" },
  { path: "routing_mark", label: "routingMark", section: "bind" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "bind" },
  { path: "netns", label: "networkNamespace", section: "bind" },
  { path: "connect_timeout", label: "connectTimeout", section: "bind" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "tcp" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "tcp" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "tcp" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "tcp", when: keepAliveOn },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "tcp", when: keepAliveOn },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean-select", options: ["true", "false"], section: "udp" },
  { path: "domain_resolver.server", label: "domainResolverServer", kind: "ref", ref: "dns-server", section: "dns" },
  { path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select", options: ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"], section: "dns" },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean", section: "dns" },
  { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number", section: "dns" },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet", section: "dns" },
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: ["default", "fallback", "hybrid"], section: "strategy" },
  { path: "network_type", label: "networkType", kind: "list", section: "strategy" },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list", section: "strategy" },
  { path: "fallback_delay", label: "fallbackDelay", section: "strategy" },
]

const network = { path: "network", label: "network", kind: "network-multi", section: "protocol" } satisfies FieldSpec
const udpOverTCP: FieldSpec[] = [
  { path: "udp_over_tcp.enabled", label: "udpOverTCPEnabled", kind: "boolean", section: "protocol" },
  { path: "udp_over_tcp.version", label: "udpOverTCPVersion", kind: "number", section: "protocol", when: udpOverTCPOn },
]

const protocolMap: Record<string, FieldSpec[]> = {
  direct: [], block: [], selector: [], urltest: [],
  socks: [
    { path: "version", label: "version", kind: "select", options: ["4", "4a", "5"], section: "auth" },
    { path: "username", label: "username", section: "auth" },
    { path: "password", label: "password", section: "auth" },
    network, ...udpOverTCP,
  ],
  http: [
    { path: "username", label: "username", section: "auth" },
    { path: "password", label: "password", section: "auth" },
    { path: "path", label: "httpPath", section: "protocol" },
    { path: "headers", label: "headers", kind: "json-object", section: "protocol" },
  ],
  shadowsocks: [
    { path: "method", label: "method", kind: "select", options: ssMethods, section: "auth" },
    { path: "password", label: "password", section: "auth" },
    { path: "plugin", label: "plugin", section: "protocol" },
    { path: "plugin_opts", label: "pluginOptions", section: "protocol" },
    network, ...udpOverTCP,
  ],
  vmess: [
    { path: "uuid", label: "uuid", section: "auth" },
    { path: "security", label: "security", kind: "select", options: ["auto", "none", "zero", "aes-128-gcm", "chacha20-poly1305"], section: "auth" },
    { path: "alter_id", label: "alterId", kind: "number", section: "auth" },
    { path: "global_padding", label: "globalPadding", kind: "boolean", section: "protocol" },
    { path: "authenticated_length", label: "authenticatedLength", kind: "boolean", section: "protocol" },
    network,
    { path: "packet_encoding", label: "packetEncoding", kind: "select", options: ["packetaddr", "xudp"], section: "protocol" },
  ],
  vless: [
    { path: "uuid", label: "uuid", section: "auth" },
    { path: "flow", label: "flow", kind: "select", options: ["xtls-rprx-vision"], section: "auth" },
    network,
    { path: "packet_encoding", label: "packetEncoding", kind: "select", options: ["packetaddr", "xudp"], section: "protocol" },
  ],
  trojan: [
    { path: "password", label: "password", section: "auth" },
    network,
  ],
  naive: [
    { path: "username", label: "username", section: "auth" },
    { path: "password", label: "password", section: "auth" },
    { path: "insecure_concurrency", label: "insecureConcurrency", kind: "number", section: "protocol" },
    { path: "extra_headers", label: "extraHeaders", kind: "json-object", section: "protocol" },
    { path: "stream_receive_window", label: "streamReceiveWindow", section: "protocol" },
    ...udpOverTCP,
    { path: "quic", label: "quic", kind: "boolean", section: "protocol" },
    { path: "quic_congestion_control", label: "quicCongestionControl", section: "protocol", when: { path: "quic", is: true } },
    { path: "quic_session_receive_window", label: "quicSessionReceiveWindow", section: "protocol", when: { path: "quic", is: true } },
  ],
  hysteria: [
    { path: "server_ports", label: "serverPorts", kind: "list", section: "protocol" },
    { path: "hop_interval", label: "hopInterval", section: "protocol" },
    { path: "up", label: "uploadBandwidth", section: "protocol" },
    { path: "up_mbps", label: "uploadMbps", kind: "number", section: "protocol" },
    { path: "down", label: "downloadBandwidth", section: "protocol" },
    { path: "down_mbps", label: "downloadMbps", kind: "number", section: "protocol" },
    { path: "obfs", label: "obfuscation", section: "protocol" },
    { path: "auth", label: "auth", section: "auth" },
    { path: "auth_str", label: "authString", section: "auth" },
    { path: "recv_window_conn", label: "receiveWindowConnection", kind: "number", section: "protocol" },
    { path: "recv_window", label: "receiveWindow", kind: "number", section: "protocol" },
    { path: "disable_mtu_discovery", label: "disableMTUDiscovery", kind: "boolean", section: "protocol" },
    network,
  ],
  hysteria2: [
    { path: "server_ports", label: "serverPorts", kind: "list", section: "protocol" },
    { path: "hop_interval", label: "hopInterval", section: "protocol" },
    { path: "up_mbps", label: "uploadMbps", kind: "number", section: "protocol" },
    { path: "down_mbps", label: "downloadMbps", kind: "number", section: "protocol" },
    { path: "obfs.type", label: "obfuscationType", kind: "select", options: ["salamander"], section: "protocol" },
    { path: "obfs.password", label: "obfuscationPassword", section: "protocol", when: { path: "obfs.type", is: "salamander" } },
    { path: "password", label: "password", section: "auth" },
    network,
    { path: "brutal_debug", label: "brutalDebug", kind: "boolean", section: "protocol" },
  ],
  tuic: [
    { path: "uuid", label: "uuid", section: "auth" },
    { path: "password", label: "password", section: "auth" },
    { path: "congestion_control", label: "congestionControl", kind: "select", options: ["cubic", "new_reno", "bbr"], section: "protocol" },
    { path: "udp_relay_mode", label: "udpRelayMode", kind: "select", options: ["native", "quic"], section: "protocol" },
    { path: "udp_over_stream", label: "udpOverStream", kind: "boolean", section: "protocol" },
    { path: "zero_rtt_handshake", label: "zeroRTTHandshake", kind: "boolean", section: "protocol" },
    { path: "heartbeat", label: "heartbeat", section: "protocol" },
    network,
  ],
  ssh: [
    { path: "user", label: "sshUser", section: "auth" },
    { path: "password", label: "password", section: "auth" },
    { path: "private_key", label: "privateKey", kind: "textarea", section: "auth" },
    { path: "private_key_path", label: "privateKeyPath", section: "auth" },
    { path: "private_key_passphrase", label: "privateKeyPassphrase", section: "auth" },
    { path: "host_key", label: "hostKey", kind: "list", section: "protocol" },
    { path: "host_key_algorithms", label: "hostKeyAlgorithms", kind: "list", section: "protocol" },
    { path: "client_version", label: "clientVersion", section: "protocol" },
  ],
  tor: [
    { path: "executable_path", label: "executablePath", section: "protocol" },
    { path: "extra_args", label: "extraArguments", kind: "list", section: "protocol" },
    { path: "data_directory", label: "dataDirectory", section: "protocol" },
    { path: "torrc", label: "torOptions", kind: "json-object", section: "protocol" },
  ],
  shadowtls: [
    { path: "version", label: "version", kind: "number", section: "protocol" },
    { path: "password", label: "password", section: "auth" },
  ],
  anytls: [
    { path: "password", label: "password", section: "auth" },
    { path: "idle_session_check_interval", label: "idleSessionCheckInterval", section: "protocol" },
    { path: "idle_session_timeout", label: "idleSessionTimeout", section: "protocol" },
    { path: "min_idle_session", label: "minimumIdleSession", kind: "number", section: "protocol" },
  ],
}

const groupMap: Record<string, FieldSpec[]> = {
  selector: [
    { path: "outbounds", label: "groupOutbounds", kind: "list", section: "group" },
    { path: "default", label: "groupDefault", section: "group" },
    { path: "interrupt_exist_connections", label: "interruptConnections", kind: "boolean", section: "group" },
  ],
  urltest: [
    { path: "outbounds", label: "groupOutbounds", kind: "list", section: "group" },
    { path: "url", label: "urlTestURL", section: "group" },
    { path: "interval", label: "urlTestInterval", section: "group" },
    { path: "tolerance", label: "urlTestTolerance", kind: "number", section: "group" },
    { path: "idle_timeout", label: "idleTimeout", section: "group" },
    { path: "interrupt_exist_connections", label: "interruptConnections", kind: "boolean", section: "group" },
  ],
}

export const outboundTLSFields: FieldSpec[] = [
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean", section: "tlsBasic" },
  { path: "tls.disable_sni", label: "disableSNI", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.server_name", label: "serverName", when: tlsOn, section: "tlsBasic" },
  { path: "tls.insecure", label: "insecure", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.alpn", label: "alpn", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.min_version", label: "minimumTLSVersion", kind: "select", options: tlsVersions, when: tlsOn, section: "tlsBasic" },
  { path: "tls.max_version", label: "maximumTLSVersion", kind: "select", options: tlsVersions, when: tlsOn, section: "tlsBasic" },
  { path: "tls.cipher_suites", label: "cipherSuites", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.curve_preferences", label: "curvePreferences", kind: "list", when: tlsOn, section: "tlsBasic" },
  { path: "tls.certificate", label: "certificate", kind: "textarea", when: tlsOn, section: "tlsCert" },
  { path: "tls.certificate_path", label: "certificatePath", when: tlsOn, section: "tlsCert" },
  { path: "tls.certificate_public_key_sha256", label: "certificateSHA256", kind: "list", when: tlsOn, section: "tlsCert" },
  { path: "tls.client_certificate", label: "clientCertificate", kind: "textarea", when: tlsOn, section: "tlsCert" },
  { path: "tls.client_certificate_path", label: "clientCertificatePath", when: tlsOn, section: "tlsCert" },
  { path: "tls.client_key", label: "clientKey", kind: "textarea", when: tlsOn, section: "tlsCert" },
  { path: "tls.client_key_path", label: "clientKeyPath", when: tlsOn, section: "tlsCert" },
  { path: "tls.fragment", label: "tlsFragment", kind: "boolean", when: tlsOn, section: "tlsFragment" },
  { path: "tls.fragment_fallback_delay", label: "fragmentFallbackDelay", when: [...fragmentOn], section: "tlsFragment" },
  { path: "tls.record_fragment", label: "recordFragment", kind: "boolean", when: tlsOn, section: "tlsFragment" },
  { path: "tls.kernel_tx", label: "kernelTX", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.kernel_rx", label: "kernelRX", kind: "boolean", when: tlsOn, section: "tlsBasic" },
  { path: "tls.ech.enabled", label: "echEnabled", kind: "boolean", when: tlsOn, section: "tlsEch" },
  { path: "tls.ech.config", label: "echConfig", kind: "list", when: [...echOn], section: "tlsEch" },
  { path: "tls.ech.config_path", label: "echConfigPath", when: [...echOn], section: "tlsEch" },
  { path: "tls.ech.query_server_name", label: "echQueryServerName", when: [...echOn], section: "tlsEch" },
  { path: "tls.utls.enabled", label: "utlsEnabled", kind: "boolean", when: tlsOn, section: "tlsUtls" },
  { path: "tls.utls.fingerprint", label: "utlsFingerprint", when: [...utlsOn], section: "tlsUtls" },
  { path: "tls.reality.enabled", label: "realityEnabled", kind: "boolean", when: tlsOn, section: "tlsReality" },
  { path: "tls.reality.public_key", label: "realityPublicKey", when: [...realityOn], section: "tlsReality" },
  { path: "tls.reality.short_id", label: "realityShortID", when: [...realityOn], section: "tlsReality" },
]

export const outboundMultiplexFields: FieldSpec[] = [
  { path: "multiplex.enabled", label: "multiplexEnabled", kind: "boolean", section: "multiplex" },
  { path: "multiplex.protocol", label: "multiplexProtocol", kind: "select", options: ["smux", "yamux", "h2mux"], when: muxOn, section: "multiplex" },
  { path: "multiplex.max_connections", label: "maxConnections", kind: "number", when: muxOn, section: "multiplex" },
  { path: "multiplex.min_streams", label: "minStreams", kind: "number", when: muxOn, section: "multiplex" },
  { path: "multiplex.max_streams", label: "maxStreams", kind: "number", when: muxOn, section: "multiplex" },
  { path: "multiplex.padding", label: "multiplexPadding", kind: "boolean", when: muxOn, section: "multiplex" },
  { path: "multiplex.brutal.enabled", label: "brutalEnabled", kind: "boolean", when: muxOn, section: "multiplex" },
  { path: "multiplex.brutal.up_mbps", label: "uploadMbps", kind: "number", when: [...brutalOn], section: "multiplex" },
  { path: "multiplex.brutal.down_mbps", label: "downloadMbps", kind: "number", when: [...brutalOn], section: "multiplex" },
]

const transportBase: FieldSpec[] = [{ path: "transport.type", label: "transportType", kind: "select", options: ["http", "ws", "quic", "grpc", "httpupgrade"], section: "transport" }]
const transportMap: Record<string, FieldSpec[]> = {
  http: [
    { path: "transport.host", label: "transportHost", kind: "list", section: "transport" },
    { path: "transport.path", label: "transportPath", section: "transport" },
    { path: "transport.method", label: "transportMethod", section: "transport" },
    { path: "transport.headers", label: "transportHeaders", kind: "json-object", section: "transport" },
    { path: "transport.idle_timeout", label: "idleTimeout", section: "transport" },
    { path: "transport.ping_timeout", label: "pingTimeout", section: "transport" },
  ],
  ws: [
    { path: "transport.path", label: "transportPath", section: "transport" },
    { path: "transport.headers", label: "transportHeaders", kind: "json-object", section: "transport" },
    { path: "transport.max_early_data", label: "maxEarlyData", kind: "number", section: "transport" },
    { path: "transport.early_data_header_name", label: "earlyDataHeaderName", section: "transport" },
  ],
  quic: [],
  grpc: [
    { path: "transport.service_name", label: "serviceName", section: "transport" },
    { path: "transport.idle_timeout", label: "idleTimeout", section: "transport" },
    { path: "transport.ping_timeout", label: "pingTimeout", section: "transport" },
    { path: "transport.permit_without_stream", label: "permitWithoutStream", kind: "boolean", section: "transport" },
  ],
  httpupgrade: [
    { path: "transport.host", label: "transportHost", section: "transport" },
    { path: "transport.path", label: "transportPath", section: "transport" },
    { path: "transport.headers", label: "transportHeaders", kind: "json-object", section: "transport" },
  ],
}

const credentialPaths = ["username", "password", "uuid", "flow", "security", "auth", "auth_str", "user", "private_key", "private_key_path", "private_key_passphrase", "obfs.password"]
const tlsCredentialPaths = ["tls.client_certificate", "tls.client_certificate_path", "tls.client_key", "tls.client_key_path"]
const knownProtocolFields = [...Object.values(protocolMap).flat(), ...Object.values(groupMap).flat()]
const knownTransportFields = Object.values(transportMap).flat()

export function protocolFields(type: string) { return protocolMap[type] ?? [] }
export function groupFields(type: string) { return groupMap[type] ?? [] }
export function transportTypeFields(type: string) { return [...transportBase, ...(transportMap[type] ?? [])] }

function removeFields(object: JsonObject, fields: FieldSpec[]) {
  return fields.reduce((next, field) => setPath(next, field.path, undefined), object)
}
function removePaths(object: JsonObject, paths: string[]) {
  return paths.reduce((next, path) => setPath(next, path, undefined), object)
}
function matchesField(value: unknown, field: FieldSpec) {
  if (value === undefined) return true
  if (field.kind === "boolean" || field.kind === "boolean-select") return typeof value === "boolean"
  if (field.kind === "number") return typeof value === "number"
  if (field.kind === "list" || field.kind === "network-multi") {
    return typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
  }
  /* c8 ignore next 2 - outbound fields currently never use number-list/users kinds */
  if (field.kind === "number-list") return typeof value === "number" || Array.isArray(value) && value.every((item) => typeof item === "number")
  if (field.kind === "json-object") return Boolean(value && typeof value === "object" && !Array.isArray(value))
  /* c8 ignore next */
  if (field.kind === "users") return Array.isArray(value) && value.every((item) => Boolean(item && typeof item === "object" && !Array.isArray(item)))
  return typeof value === "string" || typeof value === "number" || typeof value === "boolean"
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

export function managedOutboundFields(type: string, transportType = "") {
  const fields: FieldSpec[] = []
  if (dialerTypes.has(type)) fields.push(...dialerFields)
  if (groupTypes.has(type)) fields.push(...groupFields(type))
  else fields.push(...protocolFields(type))
  if (outboundTLSTypes.has(type)) fields.push(...outboundTLSFields)
  if (outboundTransportTypes.has(type)) fields.push(...transportTypeFields(transportType))
  if (outboundMultiplexTypes.has(type)) fields.push(...outboundMultiplexFields)
  return fields
}

export function applyOutboundFieldChange(object: JsonObject, next: JsonObject, typeHint = "") {
  const type = String(next.type ?? object.type ?? typeHint ?? "")
  const transportType = String(getPath(next, "transport.type") ?? getPath(object, "transport.type") ?? "")
  return pruneInvisibleFields(next, managedOutboundFields(type, transportType))
}
