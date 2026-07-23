import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  type NodeListFilters,
  type NodeStabilityBucket,
  type NodeStabilitySummary,
} from "@/features/nodes/nodes-filter"
import { cn } from "@/lib/utils"

const BUCKETS: NodeStabilityBucket[] = ["stable", "fair", "unstable", "failed", "unknown"]

const labelKey: Record<NodeStabilityBucket, string> = {
  stable: "nodes.filterStable",
  fair: "nodes.filterFair",
  unstable: "nodes.filterUnstable",
  failed: "nodes.filterFailed",
  unknown: "nodes.filterUnknown",
}

const toneClass: Record<NodeStabilityBucket, string> = {
  stable: "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
  fair: "border-amber-500/40 text-amber-700 dark:text-amber-400",
  unstable: "border-orange-500/40 text-orange-700 dark:text-orange-400",
  failed: "border-destructive/40 text-destructive",
  unknown: "border-muted-foreground/30 text-muted-foreground",
}

export function NodeStabilitySummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: NodeStabilitySummary
  filters: NodeListFilters
  onChange: (next: NodeListFilters) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="node-stability-summary">
      <span className="text-sm text-muted-foreground">
        {t("nodes.stabilitySummary", { count: summary.total })}
      </span>
      {BUCKETS.map((bucket) => {
        const count = summary[bucket]
        if (count === 0) return null
        const active = filters.stability === bucket
        return (
          <Button
            key={bucket}
            type="button"
            size="sm"
            variant={active ? "default" : "outline"}
            className={cn("h-7 gap-1.5 px-2.5", !active && toneClass[bucket])}
            aria-pressed={active}
            onClick={() => onChange({
              query: filters.query,
              stability: active ? undefined : bucket,
              sort: filters.sort,
            })}
          >
            {t(labelKey[bucket])}
            <Badge variant={active ? "secondary" : "outline"} className="tabular-nums">
              {count}
            </Badge>
          </Button>
        )
      })}
    </div>
  )
}
