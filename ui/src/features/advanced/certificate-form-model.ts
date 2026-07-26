import { isJsonObject, type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

export const certificateStores = ["system", "mozilla", "chrome", "none"] as const

const listablePaths = ["certificate", "certificate_path", "certificate_directory_path"] as const

export const certificateStoreFields = [
  { path: "store", label: "store", kind: "select", options: certificateStores, section: "store" },
] as const satisfies readonly PolicyFieldSpec[]

export const certificateSourceFields = [
  { path: "certificate", label: "certificate", kind: "textarea", section: "sources" },
  { path: "certificate_path", label: "certificatePath", kind: "list", section: "sources" },
  { path: "certificate_directory_path", label: "certificateDirectoryPath", kind: "list", section: "sources" },
] as const satisfies readonly PolicyFieldSpec[]

export const certificateFields = [
  ...certificateStoreFields,
  ...certificateSourceFields,
] as const satisfies readonly PolicyFieldSpec[]

function isStringListable(value: JsonValue | undefined): boolean {
  return value === undefined || typeof value === "string" || Array.isArray(value) && value.every((item) => typeof item === "string")
}

function normalizeListable(value: JsonValue | undefined): JsonValue | undefined {
  if (typeof value === "string") return [value]
  return value
}

function cleanList(value: JsonValue | undefined, preserveWhitespace: boolean): JsonValue | undefined {
  if (!Array.isArray(value)) return value
  const cleaned = value.flatMap((item) => {
    if (typeof item !== "string" || !item.trim()) return []
    return [preserveWhitespace ? item : item.trim()]
  })
  return cleaned.length > 0 ? cleaned : undefined
}

function hasNonEmptyString(value: JsonValue | undefined): boolean {
  if (typeof value === "string") return Boolean(value.trim())
  return Array.isArray(value) && value.some((item) => typeof item === "string" && Boolean(item.trim()))
}

export function isCertificateStructureValid(value: JsonValue | null | undefined): value is JsonObject {
  if (!isJsonObject(value)) return false
  if (value.store !== undefined && typeof value.store !== "string") return false
  return listablePaths.every((path) => isStringListable(value[path]))
}

export function normalizeCertificateObject(value: JsonValue | undefined): JsonObject {
  if (!isJsonObject(value)) return { store: "system" }
  const normalized = { ...value }
  if (normalized.store === undefined || normalized.store === "") normalized.store = "system"
  for (const path of listablePaths) {
    const next = normalizeListable(normalized[path])
    if (next === undefined) delete normalized[path]
    else normalized[path] = next
  }
  return normalized
}

export function hasCertificateSources(object: JsonObject): boolean {
  return listablePaths.some((path) => hasNonEmptyString(object[path]))
}

export function prepareCertificateObject(object: JsonObject): JsonObject {
  const prepared = { ...normalizeCertificateObject(object) }
  if (prepared.store === "system" || prepared.store === "") delete prepared.store
  for (const path of listablePaths) {
    const next = cleanList(prepared[path], path === "certificate")
    if (next === undefined) delete prepared[path]
    else prepared[path] = next
  }
  return prepared
}

export function applyCertificateConfig(config: SingBoxConfig, object: JsonObject): SingBoxConfig {
  const next = { ...config }
  const certificate = prepareCertificateObject(object)
  if (Object.keys(certificate).length === 0) delete next.certificate
  else next.certificate = certificate
  return next
}
