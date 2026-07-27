import type { LatencyPoint } from "@/lib/api/types"

export function buildSparklinePath(points: readonly LatencyPoint[]): string | null {
  const values = points.map((point) => (point.success && typeof point.latency_ms === "number" ? point.latency_ms : null))
  const finite = values.filter((value): value is number => value !== null && Number.isFinite(value))
  if (finite.length < 2) return null
  const min = Math.min(...finite)
  const max = Math.max(...finite)
  const span = Math.max(max - min, 1)
  const coordinates: Array<[number, number]> = []
  values.forEach((value, index) => {
    if (value === null || !Number.isFinite(value)) return
    const x = (index / Math.max(points.length - 1, 1)) * 100
    const y = 28 - ((value - min) / span) * 24
    coordinates.push([x, y])
  })
  if (coordinates.length < 2) return null
  return coordinates.map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(2)} ${y.toFixed(2)}`).join(" ")
}
