import {
  isJsonObject,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

const maxPort = 0xFFFF
const maxByte = 0xFF

export const wireGuardPeerFields = [
  { path: "public_key", label: "peerPublicKey", required: true, section: "peerIdentity" },
  { path: "pre_shared_key", label: "peerPreSharedKey", section: "peerIdentity" },
  { path: "address", label: "peerAddress", section: "peerNetwork" },
  { path: "port", label: "peerPort", kind: "number", section: "peerNetwork" },
  { path: "allowed_ips", label: "peerAllowedIPs", kind: "list", required: true, section: "peerNetwork" },
  { path: "persistent_keepalive_interval", label: "peerKeepalive", kind: "number", section: "peerNetwork" },
  { path: "reserved", label: "peerReserved", kind: "number-list", section: "peerAdvanced" },
] as const satisfies readonly PolicyFieldSpec[]

function stringList(value: JsonValue | undefined): string[] | null {
  if (typeof value === "string") return value.trim() ? [value.trim()] : []
  if (!Array.isArray(value) || !value.every((item) => typeof item === "string")) return null
  return value.map((item) => item.trim()).filter(Boolean)
}

function validInteger(value: unknown, minimum: number, maximum: number): value is number {
  return typeof value === "number" && Number.isInteger(value) && value >= minimum && value <= maximum
}

function validReserved(value: JsonValue | undefined): boolean {
  if (value === undefined) return true
  if (!Array.isArray(value)) return false
  return value.length === 0 || value.length === 3 && value.every((item) => validInteger(item, 0, maxByte))
}

function parseInteger(raw: string, minimum: number, maximum: number): number | null {
  const token = raw.trim()
  if (!token || !/^\d+$/.test(token)) return null
  const value = Number(token)
  return validInteger(value, minimum, maximum) ? value : null
}

function parseReserved(raw: string): number[] | null {
  const tokens = raw.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)
  if (tokens.length === 0) return []
  if (tokens.length !== 3) return null
  const values = tokens.map((token) => parseInteger(token, 0, maxByte))
  return values.some((value) => value === null) ? null : values as number[]
}

export function wireGuardPeers(value: JsonValue | undefined): JsonObject[] {
  return Array.isArray(value) ? value.filter(isJsonObject) : []
}

export function isWireGuardPeerList(value: JsonValue | undefined): value is JsonObject[] {
  return Array.isArray(value) && value.every(isJsonObject)
}

export function createWireGuardPeerDraft(): JsonObject {
  return { public_key: "", allowed_ips: [] }
}

export function isWireGuardPeerReady(peer: JsonObject): boolean {
  const publicKey = typeof peer.public_key === "string" && Boolean(peer.public_key.trim())
  const allowedIPs = stringList(peer.allowed_ips)
  const addressValid = peer.address === undefined || typeof peer.address === "string"
  const portValid = peer.port === undefined || validInteger(peer.port, 1, maxPort)
  const keepaliveValid = peer.persistent_keepalive_interval === undefined
    || validInteger(peer.persistent_keepalive_interval, 0, maxPort)
  return publicKey && Boolean(allowedIPs?.length) && addressValid && portValid && keepaliveValid && validReserved(peer.reserved)
}

export function isWireGuardPeerListReady(value: JsonValue | undefined): value is JsonObject[] {
  return isWireGuardPeerList(value) && value.length > 0 && value.every(isWireGuardPeerReady)
}

export function prepareWireGuardPeer(peer: JsonObject): JsonObject {
  let next = { ...peer }
  const allowedIPs = stringList(peer.allowed_ips)
  if (allowedIPs) next = setPolicyPath(next, "allowed_ips", allowedIPs.length ? allowedIPs : undefined)
  if (typeof peer.reserved === "string") {
    const reserved = parseReserved(peer.reserved)
    if (reserved?.length) next = setPolicyPath(next, "reserved", reserved)
    else if (reserved) next = setPolicyPath(next, "reserved", undefined)
  } else if (Array.isArray(peer.reserved) && validReserved(peer.reserved)) {
    next = setPolicyPath(next, "reserved", peer.reserved.length ? peer.reserved : undefined)
  }
  return next
}

export function transformWireGuardPeerField(
  object: JsonObject,
  field: PolicyFieldSpec,
  raw: string,
): JsonObject | null | undefined {
  if (field.path === "port") {
    if (!raw.trim()) return setPolicyPath(object, field.path, undefined)
    const value = parseInteger(raw, 1, maxPort)
    return value === null ? null : setPolicyPath(object, field.path, value)
  }
  if (field.path === "persistent_keepalive_interval") {
    if (!raw.trim()) return setPolicyPath(object, field.path, undefined)
    const value = parseInteger(raw, 0, maxPort)
    return value === null ? null : setPolicyPath(object, field.path, value)
  }
  if (field.path === "reserved") {
    const value = parseReserved(raw)
    return value === null ? null : setPolicyPath(object, field.path, value.length ? value : undefined)
  }
  return undefined
}

export function summarizeWireGuardPeer(peer: JsonObject): {
  endpoint: string | undefined
  publicKey: string
  allowedIPs: number
  keepalive: number | undefined
} {
  const address = typeof peer.address === "string" ? peer.address.trim() : ""
  const port = validInteger(peer.port, 1, maxPort) ? `:${peer.port}` : ""
  const endpoint = address || port ? `${address}${port}` : undefined
  const key = typeof peer.public_key === "string" ? peer.public_key : ""
  const publicKey = key.length > 8 ? `${key.slice(0, 8)}…` : key
  const allowedIPs = stringList(peer.allowed_ips)?.length ?? 0
  const keepalive = validInteger(peer.persistent_keepalive_interval, 0, maxPort)
    ? peer.persistent_keepalive_interval
    : undefined
  return { endpoint, publicKey, allowedIPs, keepalive }
}
