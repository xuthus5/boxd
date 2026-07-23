import type { APIEnvelope } from "@/lib/api/types"

export function rolledBackMessage(response: Pick<APIEnvelope<unknown>, "error">, fallback: string) {
  const detail = response.error?.message?.trim()
  if (!detail) return fallback
  if (fallback.includes(detail)) return fallback
  return `${fallback}: ${detail}`
}
