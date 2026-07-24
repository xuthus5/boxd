import {
  configSaveErrorFromError,
  type ConfigSaveErrorState,
} from "@/features/config/config-save-error"
import { reportConfigSaveErrorToast } from "@/features/config/config-save-error-actions"

/** Densify dry-run validate failures when callers omit a custom reporter. */
export function reportConfigValidateError(error: unknown): ConfigSaveErrorState {
  const next = configSaveErrorFromError(error)
  reportConfigSaveErrorToast(next)
  return next
}
