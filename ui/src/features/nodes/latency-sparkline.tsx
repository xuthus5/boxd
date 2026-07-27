import { useMemo } from "react"

import { buildSparklinePath } from "@/features/nodes/latency-sparkline-model"
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
