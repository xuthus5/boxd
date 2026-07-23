import type { StreamConnectionStatus } from "@/features/observability/use-stream-buffer"

export function streamStatusLabelKey(
  status: StreamConnectionStatus,
  paused: boolean,
  hasError = false,
) {
  if (paused) return "observability.paused"
  if (status === "connecting") return "observability.streamConnecting"
  if (status === "reconnecting") return "observability.streamReconnecting"
  if (status === "open") return "observability.streamLive"
  if (hasError) return "observability.streamDisconnected"
  return ""
}

export function streamStatusVariant(
  status: StreamConnectionStatus,
  paused: boolean,
  hasError = false,
): "default" | "secondary" | "destructive" | "outline" {
  if (paused) return "outline"
  if (status === "reconnecting") return "destructive"
  if (hasError && status === "closed") return "destructive"
  if (status === "connecting") return "outline"
  if (status === "open") return "default"
  return "secondary"
}

export function shouldShowStreamStatus(
  status: StreamConnectionStatus,
  paused: boolean,
  hasError = false,
) {
  return Boolean(streamStatusLabelKey(status, paused, hasError))
}
