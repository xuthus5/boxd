import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { LogLevelSummary, LogSearchFilters } from "@/features/observability/log-filter-presets"
import { cn } from "@/lib/utils"

const toneClass: Record<string, string> = {
  error: "border-destructive/40 text-destructive",
  warn: "border-amber-500/40 text-amber-700 dark:text-amber-400",
  info: "border-sky-500/40 text-sky-700 dark:text-sky-400",
  debug: "border-muted-foreground/30 text-muted-foreground",
  unknown: "border-muted-foreground/30 text-muted-foreground",
}

export function LogLevelSummaryBar({
  summary,
  filters,
  onChange,
}: {
  summary: LogLevelSummary
  filters: Pick<LogSearchFilters, "level">
  onChange: (next: Pick<LogSearchFilters, "level">) => void
}) {
  const { t } = useTranslation()
  if (summary.total === 0 || summary.buckets.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2" data-slot="log-level-summary">
      <span className="text-sm text-muted-foreground">
        {t("observability.levelSummary", { count: summary.total })}
      </span>
      {summary.buckets.map((bucket) => {
        const active = (filters.level ?? "").toLowerCase() === bucket.level.toLowerCase()
        return (
          <Button
            key={bucket.level}
            type="button"
            size="sm"
            variant={active ? "default" : "outline"}
            className={cn("h-7 gap-1.5 px-2.5", !active && (toneClass[bucket.level] ?? toneClass.unknown))}
            aria-pressed={active}
            onClick={() => onChange({ level: active ? undefined : bucket.level })}
          >
            {bucket.level}
            <Badge variant={active ? "secondary" : "outline"} className="tabular-nums">
              {bucket.count}
            </Badge>
          </Button>
        )
      })}
    </div>
  )
}
