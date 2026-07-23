import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type {
  ConnectionFacetField,
  ConnectionFacetFilters,
  ConnectionFacetSummarySection,
} from "@/features/observability/connection-facets"
import { cn } from "@/lib/utils"

const fieldLabelKey: Record<ConnectionFacetField, string> = {
  network: "observability.filterNetwork",
  protocol: "observability.filterProtocol",
  outbound: "observability.filterOutbound",
  rule: "observability.filterRule",
  process: "observability.filterProcess",
}

function shortLabel(value: string) {
  if (value.length <= 24) return value
  const parts = value.split("/")
  const leaf = parts.at(-1) || value
  if (leaf.length <= 24) return leaf
  return `${leaf.slice(0, 12)}…${leaf.slice(-8)}`
}

export function ConnectionFacetSummaryBar({
  sections,
  filters,
  onChange,
}: {
  sections: ConnectionFacetSummarySection[]
  filters: ConnectionFacetFilters
  onChange: (next: Partial<ConnectionFacetFilters>) => void
}) {
  const { t } = useTranslation()
  if (sections.length === 0) return null

  return (
    <div className="flex flex-col gap-2" data-slot="connection-facet-summary">
      <span className="text-sm text-muted-foreground">
        {t("observability.facetSummary")}
      </span>
      {sections.map((section) => {
        const selected = filters[section.field]
        return (
          <div
            key={section.field}
            className="flex flex-wrap items-center gap-2"
            data-slot={`connection-facet-summary-${section.field}`}
          >
            <span className="min-w-14 text-xs font-medium text-muted-foreground">
              {t(fieldLabelKey[section.field])}
            </span>
            {section.options.map((option) => {
              const active = selected === option.value
              return (
                <Button
                  key={`${section.field}:${option.value}`}
                  type="button"
                  size="sm"
                  variant={active ? "default" : "outline"}
                  className={cn("h-7 max-w-full gap-1.5 px-2.5")}
                  aria-pressed={active}
                  title={option.value}
                  onClick={() => onChange({
                    [section.field]: active ? "" : option.value,
                  })}
                >
                  <span className="truncate">{shortLabel(option.value)}</span>
                  <Badge variant={active ? "secondary" : "outline"} className="tabular-nums">
                    {option.count}
                  </Badge>
                </Button>
              )
            })}
          </div>
        )
      })}
    </div>
  )
}
