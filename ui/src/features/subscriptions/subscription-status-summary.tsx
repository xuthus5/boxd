import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type {
  SubscriptionFilter,
  SubscriptionListFilters,
  SubscriptionStatusSummary,
} from "@/features/subscriptions/subscription-list"
import { cn } from "@/lib/utils"

const BUCKETS: Exclude<SubscriptionFilter, "all">[] = ["error", "ok"]

const labelKey = {
  error: "subscriptions.filterError",
  ok: "subscriptions.filterOk",
} as const

const toneClass = {
  error: "border-destructive/40 text-destructive",
  ok: "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
} as const

export function SubscriptionStatusSummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: SubscriptionStatusSummary
  filters: SubscriptionListFilters
  onChange: (next: SubscriptionListFilters) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="subscription-status-summary">
      <span className="text-sm text-muted-foreground">
        {t("subscriptions.statusSummary", { count: summary.total })}
      </span>
      {BUCKETS.map((bucket) => {
        const count = summary[bucket]
        if (count === 0) return null
        const active = (filters.status ?? "all") === bucket
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
              status: active ? undefined : bucket,
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
