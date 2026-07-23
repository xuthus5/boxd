import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { VirtualList } from "@/components/virtual-list"
import { useAuth } from "@/features/auth/auth-context"
import {
  buildLogExportFilename,
  copyText,
  downloadTextFile,
  formatLogExport,
} from "@/features/observability/log-export"
import { reportExportError } from "@/features/observability/export-error-actions"
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
import { LogDesktopRow, LogMobileCard } from "@/features/observability/log-list-rows"
import { StreamErrorAlert } from "@/features/observability/stream-error-alert"
import { StreamStatusBadge } from "@/features/observability/stream-status-badge"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { useIsMobile } from "@/hooks/use-mobile"
import { useVirtualWindow } from "@/hooks/use-virtual-window"
import { usePreferences } from "@/features/preferences/preferences-provider"
import type { LogEvent } from "@/lib/api/types"

const DESKTOP_ROW_HEIGHT = 52
const MOBILE_CARD_HEIGHT = 132

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
      reportExportError(error, t, {
        scope: "logs",
        kind: "copy",
        count: items.length,
        fallback: t("observability.logsCopyFailed"),
      })
    }
  }

  const onDownload = () => {
    if (!canExport) return
    const filename = buildLogExportFilename(title)
    try {
      downloadTextFile(filename, exportText)
      toast.success(t("observability.logsExported", { count: items.length }))
    } catch (error) {
      reportExportError(error, t, {
        scope: "logs",
        kind: "export",
        count: items.length,
        filename,
        fallback: t("observability.logsExportFailed"),
      })
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

  const onClear = () => writeFilters({ tab: sharedTab() })
  const hasActiveFilter = Boolean(filter.trim()) || minimum !== "all" || Boolean(level) || Boolean(activePreset)

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex flex-wrap items-center gap-2 text-base sm:text-lg">
          {title}
          <StreamStatusBadge status={stream.status} paused={stream.paused} error={stream.error} />
        </CardTitle>
        <CardDescription>{t("observability.logDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        {stream.error ? (
          <StreamErrorAlert
            error={stream.error}
            path={path}
            status={stream.status}
            paused={stream.paused}
          />
        ) : null}
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
        {items.length === 0 ? (
          <Empty>
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
        ) : isMobile ? (
          <VirtualList
            className="h-[24rem] sm:h-[32rem]"
            items={items}
            itemHeight={MOBILE_CARD_HEIGHT}
            getKey={(item, index) => `${item.timestamp}-${item.level}-${index}`}
            aria-label={title}
            renderItem={(item) => (
              <div className="pb-1.5 sm:pb-2">
                <LogMobileCard item={item} />
              </div>
            )}
          />
        ) : (
          <LogDesktopVirtualTable items={items} title={title} />
        )}
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
        <span className="text-xs text-muted-foreground sm:text-sm">
          {t("observability.shownCount", { count: items.length })}
        </span>
      </CardFooter>
    </Card>
  )
}

function LogDesktopVirtualTable({ items, title }: { items: LogEvent[]; title: string }) {
  const { t } = useTranslation()
  const { parentRef, onScroll, window } = useVirtualWindow({
    count: items.length,
    itemHeight: DESKTOP_ROW_HEIGHT,
  })
  const slice = items.slice(window.startIndex, window.endIndex)
  return (
    <div
      ref={parentRef}
      className="h-[24rem] overflow-auto sm:h-[32rem]"
      onScroll={onScroll}
      role="region"
      aria-label={title}
    >
      <Table>
        <TableHeader className="sticky top-0 z-10 bg-background">
          <TableRow>
            <TableHead>{t("observability.time")}</TableHead>
            <TableHead>{t("dashboard.level")}</TableHead>
            <TableHead>{t("dashboard.message")}</TableHead>
            <TableHead className="w-28">{t("common.actions")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {window.startIndex > 0 ? (
            <TableRow aria-hidden="true">
              <td colSpan={4} style={{ height: window.offsetTop, padding: 0, border: 0 }} />
            </TableRow>
          ) : null}
          {slice.map((item, offset) => {
            const index = window.startIndex + offset
            return <LogDesktopRow key={`${item.timestamp}-${item.level}-${index}`} item={item} />
          })}
          {window.endIndex < items.length ? (
            <TableRow aria-hidden="true">
              <td
                colSpan={4}
                style={{
                  height: window.totalHeight - window.endIndex * DESKTOP_ROW_HEIGHT,
                  padding: 0,
                  border: 0,
                }}
              />
            </TableRow>
          ) : null}
        </TableBody>
      </Table>
    </div>
  )
}
