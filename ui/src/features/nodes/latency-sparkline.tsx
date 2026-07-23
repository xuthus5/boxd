import { useMemo } from "react"

import type { LatencyPoint } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface LatencySparklineProps {
  points?: readonly LatencyPoint[]
  className?: string
  "aria-label"?: string
}

export function LatencySparkline({ points = [], className, "aria-label": ariaLabel }: LatencySparklineProps) {
  const path = useMemo(() => buildSparklinePath(points), [points])
  if (points.length < 2 || !path) {
    return <div className={cn("h-8 w-full rounded-md bg-muted/40", className)} aria-hidden />
  }
  return (
    <svg viewBox="0 0 100 32" preserveAspectRatio="none" className={cn("h-8 w-full overflow-visible", className)} role="img" aria-label={ariaLabel}>
      <path d={path} fill="none" stroke="currentColor" strokeWidth="2" vectorEffect="non-scaling-stroke" className="text-sky-600 dark:text-sky-400" />
    </svg>
  )
}

export function buildSparklinePath(points: readonly LatencyPoint[]): string | null {
  const values = points.map((point) => (point.success && typeof point.latency_ms === "number" ? point.latency_ms : null))
  const finite = values.filter((value): value is number => value !== null && Number.isFinite(value))
  if (finite.length < 2) return null
  const min = Math.min(...finite)
  const max = Math.max(...finite)
  const span = Math.max(max - min, 1)
  const coords: Array<[number, number]> = []
  values.forEach((value, index) => {
    if (value === null || !Number.isFinite(value)) return
    const x = (index / Math.max(points.length - 1, 1)) * 100
    const y = 28 - ((value - min) / span) * 24
    coords.push([x, y])
  })
  if (coords.length < 2) return null
  return coords.map(([x, y], index) => `${index === 0 ? "M" : "L"}${x.toFixed(2)} ${y.toFixed(2)}`).join(" ")
}
