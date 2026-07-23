import type { StreamConnectionStatus } from "@/features/observability/use-stream-buffer"

export function streamStatusLabelKey(status: StreamConnectionStatus, paused: boolean) {
  if (paused) return "observability.paused"
  if (status === "connecting") return "observability.streamConnecting"
  if (status === "reconnecting") return "observability.streamReconnecting"
  if (status === "open") return "observability.streamLive"
  return ""
}

export function streamStatusVariant(
  status: StreamConnectionStatus,
  paused: boolean,
): "default" | "secondary" | "destructive" | "outline" {
  if (paused) return "outline"
  if (status === "reconnecting") return "destructive"
  if (status === "connecting") return "outline"
  if (status === "open") return "default"
  return "secondary"
}

export function shouldShowStreamStatus(status: StreamConnectionStatus, paused: boolean) {
  return Boolean(streamStatusLabelKey(status, paused))
}
