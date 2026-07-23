import { parseConfigError, type ParsedConfigError } from "@/lib/api/config-error"
import { rolledBackMessage, saveErrorMessage } from "@/lib/api/status"
import type { APIEnvelope } from "@/lib/api/types"

export interface ConfigSaveErrorState {
  message: string
  path?: string
}

export function configSaveErrorFromMessage(message: string): ConfigSaveErrorState {
  const parsed = parseConfigError(message)
  return {
    message: parsed.summary || parsed.message,
    path: parsed.path,
  }
}

export function configSaveErrorFromError(error: unknown, fallback = "request failed"): ConfigSaveErrorState {
  if (error instanceof Error) {
    return configSaveErrorFromMessage(saveErrorMessage(error, fallback))
  }
  return configSaveErrorFromMessage(String(error || fallback))
}

export function configSaveErrorFromRollback(
  response: Pick<APIEnvelope<unknown>, "error">,
  fallback: string,
): ConfigSaveErrorState {
  return configSaveErrorFromMessage(rolledBackMessage(response, fallback))
}

export function describeConfigSaveError(state: ConfigSaveErrorState | null | undefined): ParsedConfigError | null {
  if (!state?.message) return null
  return {
    message: state.message,
    path: state.path,
    summary: state.message,
  }
}

/** Top-level config section from a JSON path like inbounds[0].listen_port */
export function configSectionFromPath(path?: string): string | undefined {
  if (!path) return undefined
  const match = path.trim().match(/^([A-Za-z_][\w-]*)/)
  return match?.[1]
}
