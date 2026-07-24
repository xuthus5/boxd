import type { SingBoxConfig } from "@/lib/api/types"

export const MAX_RAW_CONFIG_FILE_BYTES = 2 * 1024 * 1024

export type RawConfigFileErrorCode = "too_large" | "invalid_json" | "invalid_root" | "read_failed"

export class RawConfigFileError extends Error {
  readonly code: RawConfigFileErrorCode

  constructor(code: RawConfigFileErrorCode) {
    super(code)
    this.name = "RawConfigFileError"
    this.code = code
  }
}

function isConfigObject(value: unknown): value is SingBoxConfig {
  return Boolean(value && typeof value === "object" && !Array.isArray(value))
}

export function parseRawConfigText(text: string): SingBoxConfig {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    throw new RawConfigFileError("invalid_json")
  }
  if (!isConfigObject(parsed)) throw new RawConfigFileError("invalid_root")
  return parsed
}

export async function readRawConfigFile(file: Pick<File, "size" | "text">): Promise<SingBoxConfig> {
  if (file.size > MAX_RAW_CONFIG_FILE_BYTES) throw new RawConfigFileError("too_large")
  try {
    return parseRawConfigText(await file.text())
  } catch (error) {
    if (error instanceof RawConfigFileError) throw error
    throw new RawConfigFileError("read_failed")
  }
}

export function formatRawConfig(config: SingBoxConfig) {
  return `${JSON.stringify(config, null, 2)}\n`
}

export function rawConfigFileErrorMessageKey(code: RawConfigFileErrorCode) {
  switch (code) {
    case "too_large":
      return "advanced.importTooLarge"
    case "invalid_json":
      return "advanced.importInvalidJSON"
    case "invalid_root":
      return "advanced.importInvalidRoot"
    case "read_failed":
      return "advanced.importFailed"
  }
}
