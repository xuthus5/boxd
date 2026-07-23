import { useTranslation } from "react-i18next"

import {
  buildLatencyHealth,
  latencyHealthBarClass,
  type LatencyHealthTone,
} from "@/features/nodes/latency-health"
import type { LatencyPoint } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function toneLabelKey(tone: LatencyHealthTone) {
  switch (tone) {
    case "excellent":
      return "nodes.healthExcellent"
    case "good":
      return "nodes.healthGood"
    case "fair":
      return "nodes.healthFair"
    case "poor":
      return "nodes.healthPoor"
    case "failed":
      return "nodes.healthFailed"
    default:
      return "nodes.healthUnknown"
  }
}

export function LatencyHealthBar({
  points = [],
  className,
}: {
  points?: readonly LatencyPoint[]
  className?: string
}) {
  const { t } = useTranslation()
  const health = buildLatencyHealth(points)
  const label = health.count
    ? t("nodes.healthSummary", { percent: health.percent, success: health.success, total: health.count })
    : t("nodes.healthEmpty")
  return (
    <div className={cn("space-y-1", className)}>
      <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
        <span>{t(toneLabelKey(health.tone))}</span>
        <span>{label}</span>
      </div>
      <div
        className="h-1.5 w-full overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-label={t("nodes.healthBarLabel")}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={health.percent}
        aria-valuetext={label}
      >
        <div
          className={cn("h-full rounded-full transition-[width]", latencyHealthBarClass(health.tone))}
          style={{ width: `${health.percent}%` }}
        />
      </div>
    </div>
  )
}
