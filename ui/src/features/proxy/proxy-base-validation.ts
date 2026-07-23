import { serverTypes } from "@/features/proxy/outbound-form-model"
import type { JsonObject } from "@/features/proxy/proxy-form-model"
import { isNumericFieldRawValid } from "@/features/proxy/proxy-field-validation"

export type ProxyBaseField = "tag" | "type" | "server" | "server_port" | "listen" | "listen_port"

export function requiredTextValid(value: unknown): boolean {
  return typeof value === "string" ? value.trim().length > 0 : value !== undefined && value !== null && String(value).trim().length > 0
}

export function portValid(value: unknown): boolean {
  if (value === undefined || value === null || value === "") return false
  const raw = String(value).trim()
  if (!isNumericFieldRawValid("number", raw)) return false
  const port = Number(raw)
  return Number.isInteger(port) && port >= 1 && port <= 65535
}

export function outboundBaseFieldValid(object: JsonObject, field: ProxyBaseField): boolean {
  const type = String(object.type ?? "")
  switch (field) {
    case "tag":
      return requiredTextValid(object.tag)
    case "type":
      return requiredTextValid(type)
    case "server":
      return !serverTypes.has(type) || requiredTextValid(object.server)
    case "server_port":
      return !serverTypes.has(type) || portValid(object.server_port)
    default:
      return true
  }
}

export function inboundBaseFieldValid(object: JsonObject, field: ProxyBaseField): boolean {
  const type = String(object.type ?? "")
  switch (field) {
    case "tag":
      return requiredTextValid(object.tag)
    case "type":
      return requiredTextValid(type)
    case "listen_port":
      if (!type || type === "tun") return true
      return portValid(object.listen_port)
    default:
      return true
  }
}

export function collectOutboundBaseInvalid(object: JsonObject): ProxyBaseField[] {
  return (["tag", "type", "server", "server_port"] as const).filter((field) => !outboundBaseFieldValid(object, field))
}

export function collectInboundBaseInvalid(object: JsonObject): ProxyBaseField[] {
  return (["tag", "type", "listen_port"] as const).filter((field) => !inboundBaseFieldValid(object, field))
}
