export type LatencyTone = "excellent" | "good" | "fair" | "poor" | "failed" | "unknown"

export function latencyTone(latencyMs: number | undefined, success = true): LatencyTone {
  if (!success) return "failed"
  if (latencyMs === undefined || !Number.isFinite(latencyMs)) return "unknown"
  if (latencyMs < 80) return "excellent"
  if (latencyMs < 150) return "good"
  if (latencyMs < 300) return "fair"
  return "poor"
}

export function latencyBadgeVariant(tone: LatencyTone): "default" | "secondary" | "outline" | "destructive" {
  switch (tone) {
    case "excellent":
      return "default"
    case "good":
      return "secondary"
    case "fair":
      return "outline"
    case "poor":
    case "failed":
      return "destructive"
    default:
      return "outline"
  }
}

export function latencyToneClass(tone: LatencyTone): string {
  switch (tone) {
    case "excellent":
      return "border-transparent bg-emerald-600 text-white dark:bg-emerald-500"
    case "good":
      return "border-transparent bg-sky-600 text-white dark:bg-sky-500"
    case "fair":
      return "border-amber-500/40 text-amber-700 dark:text-amber-300"
    case "poor":
      return "border-transparent bg-orange-600 text-white dark:bg-orange-500"
    case "failed":
      return ""
    default:
      return ""
  }
}
