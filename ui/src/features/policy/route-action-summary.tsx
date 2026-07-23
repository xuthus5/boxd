import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { RouteActionSummary, RouteSearchFilters } from "@/features/policy/route-rule-filter"
import { cn } from "@/lib/utils"

export function RouteActionSummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: RouteActionSummary
  filters: RouteSearchFilters
  onChange: (next: RouteSearchFilters) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0 || summary.buckets.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="route-action-summary">
      <span className="text-sm text-muted-foreground">
        {t("policy.route.actionSummary", { count: summary.total })}
      </span>
      {summary.buckets.map((bucket) => {
        const active = (filters.action ?? "").toLowerCase() === bucket.action.toLowerCase()
        return (
          <Button
            key={bucket.action}
            type="button"
            size="sm"
            variant={active ? "default" : "outline"}
            className={cn("h-7 gap-1.5 px-2.5")}
            aria-pressed={active}
            onClick={() => onChange({
              query: filters.query,
              action: active ? undefined : bucket.action,
            })}
          >
            {bucket.action}
            <Badge variant={active ? "secondary" : "outline"} className="tabular-nums">
              {bucket.count}
            </Badge>
          </Button>
        )
      })}
    </div>
  )
}
