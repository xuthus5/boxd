import { DownloadIcon, RefreshCwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { ConfigApplyEventRow } from "@/features/dashboard/config-apply-event-row"
import { type ConfigRestoreHandler } from "@/features/dashboard/use-config-restore"
import { configHistoryFilters, isConfigHistoryFilter, type ConfigHistoryFilter } from "@/features/advanced/config-history-filter"
import type { ConfigApplyEvent, SingBoxConfig } from "@/lib/api/types"
import { summarizeConfigHistory } from "@/features/advanced/config-history-summary"

export interface ConfigHistoryViewProps {
  isFetching: boolean
  onRefresh: () => void
  onExport: () => void
  events: ConfigApplyEvent[]
  filtered: ConfigApplyEvent[]
  query: string
  filter: ConfigHistoryFilter
  now: number
  locale: string
  currentConfig?: SingBoxConfig
  currentConfigLoading: boolean
  restoringID: string | null
  onRestore: ConfigRestoreHandler
  hasFilters: boolean
  onQueryChange: (value: string) => void
  onFilterChange: (value: ConfigHistoryFilter) => void
  onClear: () => void
}

function HistoryPageHeader({ isFetching, onRefresh, onExport, canExport }: Pick<ConfigHistoryViewProps, "isFetching" | "onRefresh" | "onExport"> & { canExport: boolean }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
      <div className="flex flex-col gap-1">
        <h1 className="text-2xl font-semibold">{t("configHistory.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("configHistory.description")}</p>
      </div>
      <div className="flex flex-wrap gap-2 self-start sm:self-auto">
        <Button type="button" size="sm" variant="outline" className="h-8" disabled={isFetching} aria-busy={isFetching} onClick={onRefresh}>
          <RefreshCwIcon data-icon="inline-start" />
          {isFetching ? t("configHistory.refreshing") : t("configHistory.refresh")}
        </Button>
        <Button type="button" size="sm" variant="outline" className="h-8" disabled={!canExport || isFetching} onClick={onExport}>
          <DownloadIcon data-icon="inline-start" />
          {t("configHistory.export")}
        </Button>
      </div>
    </div>
  )
}

function HistorySearch({ query, active, onQueryChange, onClear }: {
  query: string
  active: boolean
  onQueryChange: (value: string) => void
  onClear: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div className="min-w-0 flex-1">
        <label className="sr-only" htmlFor="config-history-search">{t("configHistory.search")}</label>
        <Input id="config-history-search" value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder={t("configHistory.searchPlaceholder")} aria-label={t("configHistory.search")} className="h-8" />
      </div>
      {active ? <Button type="button" variant="ghost" size="sm" className="h-8 shrink-0" onClick={onClear}>{t("configHistory.clearFilters")}</Button> : null}
    </div>
  )
}

function HistoryFilterToggle({ filter, onFilterChange }: Pick<ConfigHistoryViewProps, "filter" | "onFilterChange">) {
  const { t } = useTranslation()
  const labels: Record<ConfigHistoryFilter, string> = {
    all: t("configHistory.filterAll"), applied: t("configHistory.filterApplied"), validated: t("configHistory.filterValidated"),
    failed: t("configHistory.filterFailed"), restorable: t("configHistory.filterRestorable"),
  }
  return (
    <ToggleGroup aria-label={t("configHistory.filter")} value={[filter]} onValueChange={(value) => {
      const next = value[0]
      if (next && isConfigHistoryFilter(next)) onFilterChange(next)
    }} variant="outline" size="sm" spacing={0} className="max-w-full flex-wrap justify-start">
      {configHistoryFilters.map((value) => <ToggleGroupItem key={value} value={value}>{labels[value]}</ToggleGroupItem>)}
    </ToggleGroup>
  )
}

function HistoryFilters({ query, filter, onQueryChange, onFilterChange, onClear }: Pick<ConfigHistoryViewProps, "query" | "filter" | "onQueryChange" | "onFilterChange" | "onClear">) {
  const active = Boolean(query.trim()) || filter !== "all"
  return <div className="flex flex-col gap-2"><HistorySearch query={query} active={active} onQueryChange={onQueryChange} onClear={onClear} /><HistoryFilterToggle filter={filter} onFilterChange={onFilterChange} /></div>
}

function HistoryEmpty({ hasEvents, hasFilters, onClear }: { hasEvents: boolean; hasFilters: boolean; onClear: () => void }) {
  const { t } = useTranslation()
  return (
    <Empty>
      <EmptyHeader>
        <EmptyTitle>{t(hasEvents ? "configHistory.noMatch" : "configHistory.empty")}</EmptyTitle>
        <EmptyDescription>{t(hasEvents ? "configHistory.noMatchDescription" : "configHistory.emptyDescription")}</EmptyDescription>
      </EmptyHeader>
      {hasEvents && hasFilters ? <EmptyContent><Button type="button" variant="outline" onClick={onClear}>{t("configHistory.clearFilters")}</Button></EmptyContent> : null}
    </Empty>
  )
}

function HistorySummary({ events }: { events: readonly ConfigApplyEvent[] }) {
  const { t } = useTranslation()
  const summary = summarizeConfigHistory(events)
  const items = [
    ["summaryTotal", summary.total],
    ["summaryApplied", summary.applied],
    ["summaryValidated", summary.validated],
    ["summaryFailed", summary.failed],
    ["summaryRestorable", summary.restorable],
    ["summaryCurrent", summary.current],
  ] as const
  return (
    <dl className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
      {items.map(([label, value]) => (
        <div key={label} className="rounded-md border bg-muted/30 px-2.5 py-1.5">
          <dt className="text-xs text-muted-foreground">{t(`configHistory.${label}`)}</dt>
          <dd className="text-lg font-semibold tabular-nums">{value}</dd>
        </div>
      ))}
    </dl>
  )
}

function HistoryRows({ events, now, locale, currentConfig, currentConfigLoading, restoringID, onRestore }: Pick<ConfigHistoryViewProps, "events" | "now" | "locale" | "currentConfig" | "currentConfigLoading" | "restoringID" | "onRestore">) {
  return <ul className="flex flex-col gap-1.5">{events.map((event) => <ConfigApplyEventRow key={event.id || `${event.applied_at}-${event.hash}`} event={event} now={now} locale={locale} currentConfig={currentConfig} currentConfigLoading={currentConfigLoading} onRestore={onRestore} restoring={restoringID !== null} fullError />)}</ul>
}

function HistoryContent({ events, filtered, ...props }: Pick<ConfigHistoryViewProps, "events" | "filtered" | "now" | "locale" | "currentConfig" | "currentConfigLoading" | "restoringID" | "onRestore" | "hasFilters" | "onClear">) {
  if (filtered.length === 0) return <HistoryEmpty hasEvents={events.length > 0} hasFilters={props.hasFilters} onClear={props.onClear} />
  return <HistoryRows events={filtered} now={props.now} locale={props.locale} currentConfig={props.currentConfig} currentConfigLoading={props.currentConfigLoading} restoringID={props.restoringID} onRestore={props.onRestore} />
}

function HistoryCard(props: Omit<ConfigHistoryViewProps, "isFetching" | "onRefresh">) {
  const { t } = useTranslation()
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5"><CardTitle>{t("configHistory.listTitle")}</CardTitle><CardDescription>{t("configHistory.listDescription")}</CardDescription></CardHeader>
      <CardContent className="flex flex-col gap-3">
        <HistorySummary events={props.events} />
        <HistoryFilters query={props.query} filter={props.filter} onQueryChange={props.onQueryChange} onFilterChange={props.onFilterChange} onClear={props.onClear} />
        <p className="text-xs text-muted-foreground" aria-live="polite">{t("configHistory.resultCount", { shown: props.filtered.length, total: props.events.length })}</p>
        <HistoryContent {...props} />
      </CardContent>
    </Card>
  )
}

export function ConfigHistoryView(props: ConfigHistoryViewProps) {
  return <div className="flex flex-col gap-3 sm:gap-4"><HistoryPageHeader isFetching={props.isFetching} onRefresh={props.onRefresh} onExport={props.onExport} canExport={props.filtered.length > 0} /><HistoryCard {...props} /></div>
}
