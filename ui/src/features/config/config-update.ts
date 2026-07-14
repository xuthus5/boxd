import type { JsonValue, SingBoxConfig } from "@/lib/api/types"

function configArray(config: SingBoxConfig, key: string) {
  const value = config[key]
  return Array.isArray(value) ? value : []
}

function isJsonObject(value: JsonValue): value is Record<string, JsonValue> {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

export function replaceConfigSection(config: SingBoxConfig, key: string, value: JsonValue) {
  return { ...config, [key]: value }
}

export function replaceConfigArrayItem(
  config: SingBoxConfig,
  options: { key: string; index: number; patch: Record<string, JsonValue> },
) {
  const { key, index, patch } = options
  const items = [...configArray(config, key)]
  const current = items[index]
  items[index] = isJsonObject(current) ? { ...current, ...patch } : patch
  return replaceConfigSection(config, key, items)
}

export function removeConfigArrayItem(config: SingBoxConfig, key: string, index: number) {
  const items = configArray(config, key).filter((_, itemIndex) => itemIndex !== index)
  return replaceConfigSection(config, key, items)
}

export function moveConfigArrayItem(
  config: SingBoxConfig,
  options: { key: string; from: number; to: number },
) {
  const { key, from, to } = options
  const items = [...configArray(config, key)]
  const [item] = items.splice(from, 1)
  if (item !== undefined) items.splice(to, 0, item)
  return replaceConfigSection(config, key, items)
}
