import { outboundTLS114Fields, resolver114Fields } from "@/features/config/kernel114-fields"
import { endpointDNSFields } from "@/features/policy/dns114-fields"
import type { PolicyFieldSpec } from "@/features/policy/policy-form-model"

const domainStrategies = ["prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only"] as const
const networkStrategies = ["default", "fallback", "hybrid"] as const
const keepAliveOn = { path: "disable_tcp_keep_alive", falsy: true } as const
const resolverOn = { path: "domain_resolver.server" } as const
const tlsOn = { path: "tls.enabled", is: true } as const

const domainResolverFields = [
  ...resolver114Fields,
  { path: "domain_resolver.server", label: "domainResolverServer", kind: "ref", ref: "dns-server", section: "resolver" },
  { path: "domain_resolver.strategy", label: "domainResolverStrategy", kind: "select", options: domainStrategies, section: "resolver", when: resolverOn },
  { path: "domain_resolver.disable_cache", label: "domainResolverDisableCache", kind: "boolean", section: "resolver", when: resolverOn },
  { path: "domain_resolver.rewrite_ttl", label: "domainResolverRewriteTTL", kind: "number", section: "resolver", when: resolverOn },
  { path: "domain_resolver.client_subnet", label: "domainResolverClientSubnet", section: "resolver", when: resolverOn },
] as const satisfies readonly PolicyFieldSpec[]

const dialerFields = [
  { path: "detour", label: "detour", kind: "ref", ref: "outbound", section: "bind" },
  { path: "bind_interface", label: "bindInterface", kind: "network-interface", section: "bind" },
  { path: "inet4_bind_address", label: "inet4BindAddress", section: "bind" },
  { path: "inet6_bind_address", label: "inet6BindAddress", section: "bind" },
  { path: "bind_address_no_port", label: "bindAddressNoPort", kind: "boolean", section: "bind" },
  { path: "protect_path", label: "protectPath", section: "bind" },
  { path: "routing_mark", label: "routingMark", section: "bind" },
  { path: "reuse_addr", label: "reuseAddress", kind: "boolean", section: "bind" },
  { path: "netns", label: "networkNamespace", section: "bind" },
  { path: "connect_timeout", label: "connectTimeout", section: "tcp" },
  { path: "tcp_fast_open", label: "tcpFastOpen", kind: "boolean", section: "tcp" },
  { path: "tcp_multi_path", label: "tcpMultiPath", kind: "boolean", section: "tcp" },
  { path: "disable_tcp_keep_alive", label: "disableTCPKeepAlive", kind: "boolean", section: "tcp" },
  { path: "tcp_keep_alive", label: "tcpKeepAlive", section: "tcp", when: keepAliveOn },
  { path: "tcp_keep_alive_interval", label: "tcpKeepAliveInterval", section: "tcp", when: keepAliveOn },
  { path: "udp_fragment", label: "udpFragment", kind: "boolean", section: "udp" },
  ...domainResolverFields,
  { path: "network_strategy", label: "networkStrategy", kind: "select", options: networkStrategies, section: "strategy" },
  { path: "network_type", label: "networkType", kind: "list", section: "strategy" },
  { path: "fallback_network_type", label: "fallbackNetworkType", kind: "list", section: "strategy" },
  { path: "fallback_delay", label: "fallbackDelay", section: "strategy" },
] as const satisfies readonly PolicyFieldSpec[]

const remoteFields = [
  { path: "server", label: "server", section: "remote" },
  { path: "server_port", label: "serverPort", kind: "number", section: "remote" },
  ...dialerFields,
] as const satisfies readonly PolicyFieldSpec[]

const tlsFields = [
  ...outboundTLS114Fields,
  { path: "tls.enabled", label: "tlsEnabled", kind: "boolean", section: "tls" },
  { path: "tls.disable_sni", label: "tlsDisableSNI", kind: "boolean", section: "tls", when: tlsOn },
  { path: "tls.server_name", label: "tlsServerName", section: "tls", when: tlsOn },
  { path: "tls.insecure", label: "tlsInsecure", kind: "boolean", section: "tls", when: tlsOn },
  { path: "tls.alpn", label: "tlsALPN", kind: "list", section: "tls", when: tlsOn },
  { path: "tls.certificate", label: "tlsCertificate", kind: "list", section: "tls", when: tlsOn },
  { path: "tls.certificate_path", label: "tlsCertificatePath", section: "tls", when: tlsOn },
] as const satisfies readonly PolicyFieldSpec[]

const httpFields = [
  { path: "path", label: "httpPath", section: "http" },
  { path: "method", label: "method", section: "http" },
  { path: "headers", label: "headers", kind: "json-object", section: "http" },
] as const satisfies readonly PolicyFieldSpec[]

export const dnsServerFields: Record<string, readonly PolicyFieldSpec[]> = {
  local: [...dialerFields, { path: "neighbor_domain", label: "neighborDomain", kind: "list", section: "local" }, { path: "prefer_go", label: "preferGo", kind: "boolean", section: "local" }],
  hosts: [
    { path: "path", label: "path", kind: "list", section: "hosts" },
    { path: "predefined", label: "predefined", kind: "json-object", section: "hosts" },
  ],
  udp: remoteFields,
  tcp: remoteFields,
  tls: [...remoteFields, ...tlsFields],
  quic: [...remoteFields, ...tlsFields],
  https: [...remoteFields, ...tlsFields, ...httpFields],
  h3: [...remoteFields, ...tlsFields, ...httpFields],
  dhcp: [
    ...dialerFields,
    { path: "prefer_go", label: "preferGo", kind: "boolean", section: "local" },
    { path: "interface", label: "interface", kind: "network-interface", section: "local" },
  ],
  mdns: [...dialerFields, { path: "interface", label: "interface", kind: "list", section: "local" },
    { path: "prefer_go", label: "preferGo", kind: "boolean", section: "local" },
    { path: "neighbor_domain", label: "neighborDomain", kind: "list", section: "local" }],
  tailscale: endpointDNSFields,
  openvpn: endpointDNSFields,
  openconnect: endpointDNSFields,
  resolved: [{ path: "service", label: "resolvedService", section: "special" },
    { path: "accept_default_resolvers", label: "acceptDefaultResolvers", kind: "boolean", section: "special" }],
  fakeip: [
    { path: "inet4_range", label: "fakeIPIPv4Range", section: "fakeip" },
    { path: "inet6_range", label: "fakeIPIPv6Range", section: "fakeip" },
  ],
}
