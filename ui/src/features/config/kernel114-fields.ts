import type { FieldWhen } from "@/features/proxy/proxy-form-model"

export interface KernelField {
  path: string
  label: string
  kind?: "text" | "number" | "boolean" | "list" | "select" | "json-value" | "json-object"
  options?: string[]
  section?: string
  when?: FieldWhen
}

const natBehaviors = ["endpoint_independent", "address_dependent", "address_and_port_dependent"]

export const udpNATFields: KernelField[] = [
  { path: "udp_mapping", label: "udpMapping", kind: "select", options: natBehaviors, section: "udp" },
  { path: "udp_filtering", label: "udpFiltering", kind: "select", options: natBehaviors, section: "udp" },
  { path: "udp_nat_max", label: "udpNATMax", kind: "number", section: "udp" },
]

export const http2Fields: KernelField[] = [
  { path: "idle_timeout", label: "idleTimeout", section: "protocol" },
  { path: "keep_alive_period", label: "keepAlivePeriod", section: "protocol" },
  { path: "stream_receive_window", label: "streamReceiveWindow", section: "protocol" },
  { path: "connection_receive_window", label: "connectionReceiveWindow", section: "protocol" },
  { path: "max_concurrent_streams", label: "maxConcurrentStreams", kind: "number", section: "protocol" },
]

export const quicFields: KernelField[] = [
  ...http2Fields,
  { path: "initial_packet_size", label: "initialPacketSize", kind: "number", section: "protocol" },
  { path: "disable_path_mtu_discovery", label: "disablePathMTUDiscovery", kind: "boolean", section: "protocol" },
]

export const resolver114Fields: KernelField[] = [
  { path: "domain_resolver.timeout", label: "resolverTimeout", section: "dns", when: { path: "domain_resolver.server" } },
  { path: "domain_resolver.disable_optimistic_cache", label: "disableOptimisticCache", kind: "boolean", section: "dns", when: { path: "domain_resolver.server" } },
]

export const tls114Fields: KernelField[] = [
  { path: "tls.handshake_timeout", label: "handshakeTimeout", section: "tlsBasic", when: { path: "tls.enabled", is: true } },
]

export const outboundTLS114Fields: KernelField[] = [
  ...tls114Fields,
  { path: "tls.engine", label: "tlsEngine", kind: "select", options: ["go", "apple", "windows"], section: "tlsBasic", when: { path: "tls.enabled", is: true } },
  { path: "tls.spoof", label: "tlsSpoof", section: "tlsFragment", when: { path: "tls.enabled", is: true } },
  { path: "tls.spoof_method", label: "tlsSpoofMethod", kind: "select", options: ["wrong-sequence", "wrong-checksum", "wrong-ack", "wrong-md5", "wrong-timestamp"], section: "tlsFragment", when: { path: "tls.spoof" } },
]

export const inboundTLS114Fields: KernelField[] = [
  ...tls114Fields,
  { path: "tls.certificate_provider", label: "certificateProvider", kind: "json-value", section: "tlsCert", when: { path: "tls.enabled", is: true } },
]
