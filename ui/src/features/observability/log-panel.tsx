import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Link, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { NetworkIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAuth } from "@/features/auth/auth-context"
import { logConnectionsHref } from "@/features/observability/connection-facets"
import {
  buildLogExportFilename,
  copyText,
  downloadTextFile,
  formatLogExport,
} from "@/features/observability/log-export"
import { LogFilters } from "@/features/observability/log-filters"
import {
  applyLogPreset,
  LOG_FILTER_PRESETS,
  matchesLogFilter,
  matchesLogLevel,
  parseLogSearchParams,
  resolveLogSeed,
  summarizeLogLevels,
  toLogSearchParams,
  type LogFilterPresetId,
  type LogSearchFilters,
  type LogThresholdParam,
} from "@/features/observability/log-filter-presets"
import { meetsLogThreshold, type LogThreshold } from "@/features/observability/log-level"
import { useIsMobile } from "@/hooks/use-mobile"
import { StreamStatusBadge } from "@/features/observability/stream-status-badge"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { usePreferences } from "@/features/preferences/preferences-provider"
import type { LogEvent } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function formatLogTimestamp(timestamp?: string) {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

export function LogPanel({ path, title }: { path: string; title: string }) {
  const { t } = useTranslation()
  const preferences = usePreferences()
  const stream = useStreamBuffer<LogEvent>(path, useAuth().session!.token)
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseLogSearchParams(searchParams), [searchParams])
  const seed = resolveLogSeed(filters)
  const filter = seed.filter
  const minimum = (seed.minimum ?? preferences.minimumLogLevel) as LogThreshold
  const level = seed.level
  const activePreset = seed.preset
  const levelSummary = useMemo(() => summarizeLogLevels(stream.items, filter), [filter, stream.items])
  const items = useMemo(
    () => stream.items.filter((item) => meetsLogThreshold(item.level, minimum)
      && matchesLogFilter(item.level, item.message, filter)
      && matchesLogLevel(item.level, level)),
    [filter, level, minimum, stream.items],
  )
  const isMobile = useIsMobile()

  const exportText = useMemo(() => formatLogExport(items), [items])
  const canExport = items.length > 0

  const writeFilters = (next: LogSearchFilters) => {
    setSearchParams(toLogSearchParams(next), { replace: true })
  }

  const sharedTab = (): LogSearchFilters["tab"] => filters.tab

  const onCopy = async () => {
    if (!canExport) return
    try {
      await copyText(exportText)
      toast.success(t("observability.logsCopied", { count: items.length }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("observability.logsCopyFailed"))
    }
  }

  const onDownload = () => {
    if (!canExport) return
    try {
      downloadTextFile(buildLogExportFilename(title), exportText)
      toast.success(t("observability.logsExported", { count: items.length }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("observability.logsExportFailed"))
    }
  }

  const onPreset = (id: LogFilterPresetId) => {
    const applied = applyLogPreset(LOG_FILTER_PRESETS.find((preset) => preset.id === id))
    writeFilters({
      tab: sharedTab(),
      query: applied.filter,
      minimum: applied.minimum ?? "all",
      level: undefined,
      preset: id,
    })
  }

  const onFilterChange = (value: string) => {
    writeFilters({
      tab: sharedTab(),
      query: value,
      minimum: filters.minimum ?? seed.minimum,
      level,
      preset: undefined,
    })
  }

  const onMinimumChange = (value: LogThreshold) => {
    writeFilters({
      tab: sharedTab(),
      query: filter,
      minimum: value as LogThresholdParam,
      level,
      preset: undefined,
    })
  }

  const onLevelChange = (next: Pick<LogSearchFilters, "level">) => {
    writeFilters({
      tab: sharedTab(),
      query: filter,
      minimum: minimum as LogThresholdParam,
      level: next.level,
      preset: undefined,
    })
  }

  const onClear = () => {
    writeFilters({ tab: sharedTab() })
  }

  const hasActiveFilter = Boolean(filter.trim()) || minimum !== "all" || Boolean(level) || Boolean(activePreset)

  return <Card size="sm">
    <CardHeader className="gap-1.5">
      <CardTitle className="flex flex-wrap items-center gap-2 text-base sm:text-lg">{title}<StreamStatusBadge status={stream.status} paused={stream.paused} /></CardTitle>
      <CardDescription>{t("observability.logDescription")}</CardDescription>
    </CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">
      {stream.error ? <Alert variant="destructive">
        <AlertTitle>{t("observability.streamError")}</AlertTitle>
        <AlertDescription>{stream.error}</AlertDescription>
      </Alert> : null}
      <LogFilters
        filter={filter}
        minimum={minimum}
        level={level}
        levelSummary={levelSummary}
        activePreset={activePreset}
        onFilterChange={onFilterChange}
        onMinimumChange={onMinimumChange}
        onLevelChange={onLevelChange}
        onPreset={onPreset}
        onClear={onClear}
      />
      <ScrollArea className="h-[24rem] sm:h-[32rem]">
        {items.length === 0
          ? <Empty>
            <EmptyHeader>
              <EmptyTitle>
                {hasActiveFilter ? t("observability.noMatchLogs") : t("observability.noLogs")}
              </EmptyTitle>
              <EmptyDescription>
                {hasActiveFilter ? t("observability.noMatchLogsDescription") : t("observability.waitLogs")}
              </EmptyDescription>
            </EmptyHeader>
            {hasActiveFilter ? (
              <EmptyContent>
                <Button type="button" variant="outline" onClick={onClear}>
                  {t("observability.clearLogFilter")}
                </Button>
              </EmptyContent>
            ) : null}
          </Empty>
          : isMobile
            ? <div className="flex flex-col gap-1.5 sm:gap-2">
              {items.map((item, index) => {
                const connectionsHref = logConnectionsHref(item.message)
                return (
                  <Card key={`${item.timestamp}-${item.level}-${index}`} size="sm">
                    <CardHeader className="min-w-0 gap-1">
                      <CardTitle className="flex flex-wrap items-center gap-2 text-sm font-medium">
                        <Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge>
                        <time className="text-muted-foreground" dateTime={item.timestamp || undefined}>
                          {formatLogTimestamp(item.timestamp)}
                        </time>
                      </CardTitle>
                      <CardDescription className="whitespace-normal break-words text-foreground">
                        {item.message}
                      </CardDescription>
                    </CardHeader>
                    {connectionsHref ? (
                      <CardContent>
                        <Link
                          to={connectionsHref}
                          aria-label={`${t("observability.viewConnections")}: ${item.message}`}
                          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
                        >
                          <NetworkIcon data-icon="inline-start" />
                          {t("observability.viewConnections")}
                        </Link>
                      </CardContent>
                    ) : null}
                  </Card>
                )
              })}
            </div>
            : <Table>
              <TableHeader><TableRow>
                <TableHead>{t("observability.time")}</TableHead>
                <TableHead>{t("dashboard.level")}</TableHead>
                <TableHead>{t("dashboard.message")}</TableHead>
                <TableHead className="w-28">{t("common.actions")}</TableHead>
              </TableRow></TableHeader>
              <TableBody>
                {items.map((item, index) => {
                  const connectionsHref = logConnectionsHref(item.message)
                  return (
                    <TableRow key={`${item.timestamp}-${item.level}-${index}`}>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        <time dateTime={item.timestamp || undefined}>{formatLogTimestamp(item.timestamp)}</time>
                      </TableCell>
                      <TableCell><Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge></TableCell>
                      <TableCell className="min-w-64 whitespace-normal break-words">{item.message}</TableCell>
                      <TableCell>
                        {connectionsHref ? (
                          <Link
                            to={connectionsHref}
                            aria-label={`${t("observability.viewConnections")}: ${item.message}`}
                            className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
                          >
                            <NetworkIcon data-icon="inline-start" />
                            {t("observability.viewConnections")}
                          </Link>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>}
      </ScrollArea>
    </CardContent>
    <CardFooter className="flex flex-wrap gap-1.5 sm:gap-2">
      <Button size="sm" className="h-8" variant="outline" onClick={() => stream.setPaused(!stream.paused)}>
        {stream.paused ? t("observability.resume") : t("observability.pause")}
      </Button>
      <Button size="sm" className="h-8" variant="outline" onClick={stream.clear}>{t("observability.clear")}</Button>
      <Button size="sm" className="h-8" variant="outline" disabled={!canExport} onClick={() => { void onCopy() }}>
        {t("observability.copyLogs")}
      </Button>
      <Button size="sm" className="h-8" variant="outline" disabled={!canExport} onClick={onDownload}>
        {t("observability.exportLogs")}
      </Button>
      <span className="text-xs text-muted-foreground sm:text-sm">{t("observability.shownCount", { count: items.length })}</span>
    </CardFooter>
  </Card>
}
