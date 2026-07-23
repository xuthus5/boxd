import type { APIEnvelope } from "@/lib/api/types"
import { formatConfigErrorMessage } from "@/lib/api/config-error"

export function rolledBackMessage(response: Pick<APIEnvelope<unknown>, "error">, fallback: string) {
  const detail = response.error?.message?.trim()
  if (!detail) return fallback
  return formatConfigErrorMessage(detail)
}

export function saveErrorMessage(error: Error, fallback?: string) {
  const message = error.message?.trim() || fallback || "request failed"
  return formatConfigErrorMessage(message)
}
