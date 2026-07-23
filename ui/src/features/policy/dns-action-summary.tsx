import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { DNSActionSummary, DNSSearchFilters } from "@/features/policy/dns-filter"
import { cn } from "@/lib/utils"

export function DNSActionSummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: DNSActionSummary
  filters: DNSSearchFilters
  onChange: (next: DNSSearchFilters) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0 || summary.buckets.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="dns-action-summary">
      <span className="text-sm text-muted-foreground">
        {t("policy.dns.actionSummary", { count: summary.total })}
      </span>
      {summary.buckets.map((bucket) => {
        const active = (filters.ruleAction ?? "").toLowerCase() === bucket.action.toLowerCase()
        return (
          <Button
            key={bucket.action}
            type="button"
            size="sm"
            variant={active ? "default" : "outline"}
            className={cn("h-7 gap-1.5 px-2.5")}
            aria-pressed={active}
            onClick={() => onChange({
              servers: filters.servers,
              rules: filters.rules,
              serverType: filters.serverType,
              ruleAction: active ? undefined : bucket.action,
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
