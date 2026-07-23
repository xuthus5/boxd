import { Trash2Icon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ConfirmAction } from "@/components/confirm-action"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuth } from "@/features/auth/auth-context"
import { formatBytes } from "@/features/dashboard/format"
import {
  loadConnectionColumns,
  saveConnectionColumns,
  toggleConnectionColumn,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import {
  buildConnectionExportFilename,
  formatConnectionExport,
  sortConnections,
  type ConnectionSortKey,
} from "@/features/observability/connection-export"
import { ConnectionGroupTable } from "@/features/observability/connection-group-table"
import { ConnectionListTable } from "@/features/observability/connection-list-table"
import {
  connectionFiltersActive,
  filterConnectionsByFacets,
  listConnectionFacets,
  parseConnectionSearchParams,
  summarizeConnectionFacets,
  toConnectionSearchParams,
  type ConnectionFacetFilters,
  type ConnectionView,
} from "@/features/observability/connection-facets"
import { ConnectionFacetSummaryBar } from "@/features/observability/connection-facet-summary"
import { ConnectionToolbar } from "@/features/observability/connection-toolbar"
import {
  aggregateConnections,
  summarizeConnections,
} from "@/features/observability/connection-stats"
import { downloadTextFile } from "@/features/observability/log-export"
import { StreamStatusBadge } from "@/features/observability/stream-status-badge"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent } from "@/lib/api/types"

export function ConnectionsPage() {
  const { t } = useTranslation()
  const token = useAuth().session!.token
  const [searchParams, setSearchParams] = useSearchParams()
  const stream = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const filters = useMemo(() => parseConnectionSearchParams(searchParams), [searchParams])
  const view: ConnectionView = filters.view ?? "list"
  const sort: ConnectionSortKey = filters.sort ?? "traffic"
  const [closingId, setClosingId] = useState<string | "all" | null>(null)
  const [columns, setColumns] = useState<ConnectionColumnId[]>(() => loadConnectionColumns())
  const connections = useMemo(() => stream.items.at(-1)?.list ?? [], [stream.items])
  const filtered = useMemo(() => filterConnectionsByFacets(connections, filters), [connections, filters])
  const facetOptions = useMemo(() => ({
    network: listConnectionFacets(connections, "network"),
    protocol: listConnectionFacets(connections, "protocol"),
    inbound: listConnectionFacets(connections, "inbound"),
    outbound: listConnectionFacets(connections, "outbound"),
    rule: listConnectionFacets(connections, "rule"),
    process: listConnectionFacets(connections, "process"),
  }), [connections])
  const facetSummary = useMemo(() => summarizeConnectionFacets(connections, filters), [connections, filters])
  const facetsActive = connectionFiltersActive(filters)
  const sorted = useMemo(() => sortConnections(filtered, sort), [filtered, sort])
  const summary = useMemo(() => summarizeConnections(filtered), [filtered])
  const byOutbound = useMemo(() => aggregateConnections(filtered, "outbound"), [filtered])
  const byRule = useMemo(() => aggregateConnections(filtered, "rule"), [filtered])
  const byProcess = useMemo(() => aggregateConnections(filtered, "process"), [filtered])
  const busy = closingId !== null
  const canExport = sorted.length > 0
  const sortOptions = useMemo(() => ([
    { label: t("observability.sortByTraffic"), value: "traffic" as const },
    { label: t("observability.sortByDownload"), value: "download" as const },
    { label: t("observability.sortByUpload"), value: "upload" as const },
    { label: t("observability.sortByDuration"), value: "duration" as const },
    { label: t("observability.sortByTarget"), value: "target" as const },
    { label: t("observability.sortByOutbound"), value: "outbound" as const },
  ]), [t])

  const patchFilters = (patch: Partial<ConnectionFacetFilters>) => {
    const next: ConnectionFacetFilters = {
      query: patch.query !== undefined ? patch.query : filters.query,
      network: patch.network !== undefined ? patch.network || undefined : filters.network,
      protocol: patch.protocol !== undefined ? patch.protocol || undefined : filters.protocol,
      inbound: patch.inbound !== undefined ? patch.inbound || undefined : filters.inbound,
      outbound: patch.outbound !== undefined ? patch.outbound || undefined : filters.outbound,
      rule: patch.rule !== undefined ? patch.rule || undefined : filters.rule,
      process: patch.process !== undefined ? patch.process || undefined : filters.process,
      view: "view" in patch ? patch.view : filters.view,
      sort: "sort" in patch ? patch.sort : filters.sort,
    }
    setSearchParams(toConnectionSearchParams(next), { replace: true })
  }

  const onSortChange = (value: ConnectionSortKey) => {
    patchFilters({ sort: value === "traffic" ? undefined : value })
  }

  const onViewChange = (value: string) => {
    const nextView = value as ConnectionView
    patchFilters({ view: nextView === "list" ? undefined : nextView })
  }

  const onExport = () => {
    if (!canExport) return
    try {
      downloadTextFile(buildConnectionExportFilename(), formatConnectionExport(sorted))
      toast.success(t("observability.connectionsExported", { count: sorted.length }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("observability.connectionsExportFailed"))
    }
  }

  const run = async (request: Promise<unknown>, message: string, id: string | "all" = "all") => {
    setClosingId(id)
    try {
      await request
      toast.success(message)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error))
    } finally {
      setClosingId(null)
    }
  }

  const closeGroup = async (field: "outbound" | "rule" | "process", key: string) => {
    if (key === "—") return
    setClosingId(`group:${field}:${key}`)
    try {
      const result = await api.stats.closeAll({ [field]: key })
      toast.success(t("observability.closedCount", { count: result.closed }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error))
    } finally {
      setClosingId(null)
    }
  }

  const closeFiltered = async () => {
    setClosingId("filtered")
    try {
      const result = await api.stats.closeAll({ ids: filtered.map((item) => item.id) })
      toast.success(t("observability.closedCount", { count: result.closed }))
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error))
    } finally {
      setClosingId(null)
    }
  }

  const onToggleColumn = (id: ConnectionColumnId, enabled: boolean) => {
    setColumns((current) => saveConnectionColumns(toggleConnectionColumn(current, id, enabled)))
  }

  const clearFacets = () => {
    setSearchParams(toConnectionSearchParams({ view: filters.view, sort: filters.sort }), { replace: true })
  }
  const emptyActionLabel = facetsActive ? t("observability.clearFacets") : undefined
  const emptyAction = facetsActive ? clearFacets : undefined

  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("observability.connections")}</h1>
        <ConfirmAction
          trigger={<Button size="sm" className="h-8 w-full sm:w-auto" variant="destructive" disabled={busy}><Trash2Icon data-icon="inline-start" />{t("observability.closeAll")}</Button>}
          title={t("observability.closeAllTitle")}
          description={t("observability.closeAllDescription")}
          confirmLabel={t("observability.confirmClose")}
          confirmVariant="destructive"
          onConfirm={() => void run(api.stats.closeAll(), t("observability.closeAll"))}
        />
      </div>
      {stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}
      <Card size="sm">
        <CardHeader className="gap-1.5 sm:gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>
                {t("observability.liveConnections")}{" "}
                <Badge variant="secondary">{t("observability.shownCount", { count: filtered.length })}</Badge>
                <StreamStatusBadge status={stream.status} paused={stream.paused} className="ml-2" />
              </CardTitle>
              <CardDescription>{t("observability.connectionsDescription")}</CardDescription>
            </div>
            <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground sm:text-sm">
              <span>{t("dashboard.upload")}: {formatBytes(summary.upload)}</span>
              <span>{t("dashboard.download")}: {formatBytes(summary.download)}</span>
              <span>{t("observability.outboundCount", { count: summary.outbounds })}</span>
            </div>
          </div>
          {connections.length > 0
            ? <ConnectionFacetSummaryBar sections={facetSummary} filters={filters} onChange={patchFilters} />
            : null}
          <ConnectionToolbar
            query={filters.query ?? ""}
            network={filters.network ?? ""}
            protocol={filters.protocol ?? ""}
            inbound={filters.inbound ?? ""}
            outbound={filters.outbound ?? ""}
            rule={filters.rule ?? ""}
            process={filters.process ?? ""}
            sort={sort}
            columns={columns}
            networkOptions={facetOptions.network}
            protocolOptions={facetOptions.protocol}
            inboundOptions={facetOptions.inbound}
            outboundOptions={facetOptions.outbound}
            ruleOptions={facetOptions.rule}
            processOptions={facetOptions.process}
            sortOptions={sortOptions}
            facetsActive={facetsActive}
            filteredCount={filtered.length}
            busy={busy}
            canExport={canExport}
            paused={stream.paused}
            onQueryChange={(value) => patchFilters({ query: value })}
            onNetworkChange={(value) => patchFilters({ network: value })}
            onProtocolChange={(value) => patchFilters({ protocol: value })}
            onInboundChange={(value) => patchFilters({ inbound: value })}
            onOutboundChange={(value) => patchFilters({ outbound: value })}
            onRuleChange={(value) => patchFilters({ rule: value })}
            onProcessChange={(value) => patchFilters({ process: value })}
            onSortChange={onSortChange}
            onToggleColumn={onToggleColumn}
            onClearFacets={clearFacets}
            onTogglePause={() => stream.setPaused(!stream.paused)}
            onExport={onExport}
            onCloseFiltered={closeFiltered}
          />
        </CardHeader>
        <CardContent>
          {connections.length > 0 ? (
            <Tabs value={view} onValueChange={onViewChange}>
              <TabsList>
                <TabsTrigger value="list">{t("observability.listView")}</TabsTrigger>
                <TabsTrigger value="outbound">{t("observability.byOutbound")}</TabsTrigger>
                <TabsTrigger value="rule">{t("observability.byRule")}</TabsTrigger>
                <TabsTrigger value="process">{t("observability.byProcess")}</TabsTrigger>
              </TabsList>
              <TabsContent value="list" className="mt-3">
                <ConnectionListTable
                  connections={sorted}
                  columns={columns}
                  busy={busy}
                  onClose={(id) => void run(api.stats.closeConnection(id), t("observability.close"), id)}
                  emptyActionLabel={emptyActionLabel}
                  onEmptyAction={emptyAction}
                />
              </TabsContent>
              <TabsContent value="outbound" className="mt-3">
                <ConnectionGroupTable
                  groups={byOutbound}
                  field="outbound"
                  busy={busy}
                  onCloseGroup={closeGroup}
                  emptyTitle={t("observability.noMatch")}
                  emptyDescription={t("observability.noMatchDescription")}
                  emptyActionLabel={emptyActionLabel}
                  onEmptyAction={emptyAction}
                />
              </TabsContent>
              <TabsContent value="rule" className="mt-3">
                <ConnectionGroupTable
                  groups={byRule}
                  field="rule"
                  busy={busy}
                  onCloseGroup={closeGroup}
                  emptyTitle={t("observability.noMatch")}
                  emptyDescription={t("observability.noMatchDescription")}
                  emptyActionLabel={emptyActionLabel}
                  onEmptyAction={emptyAction}
                />
              </TabsContent>
              <TabsContent value="process" className="mt-3">
                <ConnectionGroupTable
                  groups={byProcess}
                  field="process"
                  busy={busy}
                  onCloseGroup={closeGroup}
                  emptyTitle={t("observability.noMatch")}
                  emptyDescription={t("observability.noMatchDescription")}
                  emptyActionLabel={emptyActionLabel}
                  onEmptyAction={emptyAction}
                />
              </TabsContent>
            </Tabs>
          ) : (
            <Empty><EmptyHeader><EmptyTitle>{t("observability.noConnections")}</EmptyTitle><EmptyDescription>{t("observability.noConnectionsDescription")}</EmptyDescription></EmptyHeader></Empty>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
