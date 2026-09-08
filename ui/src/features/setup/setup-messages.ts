const translatedMessages = new Set([
  "container_port_mapping", "proxy_nodes_required", "tun_ipv4_only", "config_reset",
  "tun_permission_required", "tun_device_unavailable", "tun_unknown", "tun_unsupported",
  "ipv6_disabled", "ipv6_unknown", "ipv6_unsupported",
])

export function setupMessage(message: string, translate: (key: string) => string) {
  return translatedMessages.has(message) ? translate(`setup.messages.${message}`) : message
}

export function setupCapabilityLabel(feature: "tun" | "ipv6", available: boolean, reason?: string) {
  if (reason === `${feature}_unknown`) return `setup.${feature}Unknown`
  return `setup.${feature}${available ? "Available" : "Unavailable"}`
}
