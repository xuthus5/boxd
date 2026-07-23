import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { ProxyListFilters, ProxyTypeSummary } from "@/features/proxy/proxy-filter"
import { cn } from "@/lib/utils"

export function ProxyTypeSummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: ProxyTypeSummary
  filters: ProxyListFilters
  onChange: (next: ProxyListFilters) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0 || summary.buckets.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="proxy-type-summary">
      <span className="text-sm text-muted-foreground">
        {t("proxy.typeSummary", { count: summary.total })}
      </span>
      {summary.buckets.map((bucket) => {
        const active = (filters.type ?? "").toLowerCase() === bucket.type.toLowerCase()
        return (
          <Button
            key={bucket.type}
            type="button"
            size="sm"
            variant={active ? "default" : "outline"}
            className={cn("h-7 gap-1.5 px-2.5")}
            aria-pressed={active}
            onClick={() => onChange({
              query: filters.query,
              type: active ? undefined : bucket.type,
            })}
          >
            {bucket.type}
            <Badge variant={active ? "secondary" : "outline"} className="tabular-nums">
              {bucket.count}
            </Badge>
          </Button>
        )
      })}
    </div>
  )
}
