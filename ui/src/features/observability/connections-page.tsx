import { Columns3Icon, Trash2Icon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ConfirmAction } from "@/components/confirm-action"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuth } from "@/features/auth/auth-context"
import { formatBytes } from "@/features/dashboard/format"
import {
  CONNECTION_COLUMNS,
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
  aggregateConnections,
  matchesConnection,
  summarizeConnections,
} from "@/features/observability/connection-stats"
import { downloadTextFile } from "@/features/observability/log-export"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent } from "@/lib/api/types"

export function ConnectionsPage() {
  const { t } = useTranslation()
  const token = useAuth().session!.token
  const stream = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const [closingId, setClosingId] = useState<string | "all" | null>(null)
  const [query, setQuery] = useState("")
  const [sort, setSort] = useState<ConnectionSortKey>("traffic")
  const [columns, setColumns] = useState<ConnectionColumnId[]>(() => loadConnectionColumns())
  const snapshot = stream.items.at(-1)
  const connections = useMemo(() => snapshot?.list ?? [], [snapshot?.list])
  const normalized = query.trim().toLowerCase()
  const filtered = useMemo(
    () => connections.filter((connection) => matchesConnection(connection, normalized)),
    [connections, normalized],
  )
  const sorted = useMemo(() => sortConnections(filtered, sort), [filtered, sort])
  const summary = useMemo(() => summarizeConnections(filtered), [filtered])
  const byOutbound = useMemo(() => aggregateConnections(filtered, "outbound"), [filtered])
  const byRule = useMemo(() => aggregateConnections(filtered, "rule"), [filtered])
  const busy = closingId !== null
  const canExport = sorted.length > 0
  const sortOptions: { label: string; value: ConnectionSortKey }[] = [
    { label: t("observability.sortByTraffic"), value: "traffic" },
    { label: t("observability.sortByDownload"), value: "download" },
    { label: t("observability.sortByUpload"), value: "upload" },
    { label: t("observability.sortByDuration"), value: "duration" },
    { label: t("observability.sortByTarget"), value: "target" },
    { label: t("observability.sortByOutbound"), value: "outbound" },
  ]

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

  const closeGroup = async (field: "outbound" | "rule", key: string) => {
    if (key === "—") return
    const filters = field === "outbound" ? { outbound: key } : { rule: key }
    setClosingId(`group:${field}:${key}`)
    try {
      const result = await api.stats.closeAll(filters)
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

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("observability.connections")}</h1>
        <ConfirmAction
          trigger={<Button className="w-full sm:w-auto" variant="destructive" disabled={busy}><Trash2Icon data-icon="inline-start" />{t("observability.closeAll")}</Button>}
          title={t("observability.closeAllTitle")}
          description={t("observability.closeAllDescription")}
          confirmLabel={t("observability.confirmClose")}
          confirmVariant="destructive"
          onConfirm={() => void run(api.stats.closeAll(), t("observability.closeAll"))}
        />
      </div>
      {stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}
      <Card>
        <CardHeader className="gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>
                {t("observability.liveConnections")}{" "}
                <Badge variant="secondary">{t("observability.shownCount", { count: filtered.length })}</Badge>
                {stream.paused ? <Badge variant="outline" className="ml-2">{t("observability.paused")}</Badge> : null}
              </CardTitle>
              <CardDescription>{t("observability.connectionsDescription")}</CardDescription>
            </div>
            <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
              <span>{t("dashboard.upload")}: {formatBytes(summary.upload)}</span>
              <span>{t("dashboard.download")}: {formatBytes(summary.download)}</span>
              <span>{t("observability.outboundCount", { count: summary.outbounds })}</span>
            </div>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
            <label className="sr-only" htmlFor="connections-search">{t("observability.searchConnections")}</label>
            <Input
              id="connections-search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("observability.searchConnectionsPlaceholder")}
              className="sm:max-w-xs"
              aria-label={t("observability.searchConnections")}
            />
            <Select items={sortOptions} value={sort} onValueChange={(value) => setSort(String(value) as ConnectionSortKey)}>
              <SelectTrigger aria-label={t("observability.sortConnections")} className="w-full sm:w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {sortOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <DropdownMenu>
              <DropdownMenuTrigger render={<Button variant="outline" />}>
                <Columns3Icon data-icon="inline-start" />
                {t("observability.columns")}
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-52">
                <DropdownMenuGroup>
                  <DropdownMenuLabel>{t("observability.columns")}</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {CONNECTION_COLUMNS.map((column) => (
                    <DropdownMenuCheckboxItem
                      key={column.id}
                      checked={columns.includes(column.id)}
                      disabled={column.required}
                      onCheckedChange={(checked) => onToggleColumn(column.id, checked === true)}
                    >
                      {t(column.labelKey)}
                    </DropdownMenuCheckboxItem>
                  ))}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
            <Button variant="outline" onClick={() => stream.setPaused(!stream.paused)}>
              {stream.paused ? t("observability.resume") : t("observability.pause")}
            </Button>
            <Button variant="outline" disabled={!canExport} onClick={onExport}>
              {t("observability.exportConnections")}
            </Button>
            {normalized && filtered.length > 0 ? (
              <ConfirmAction
                trigger={<Button variant="outline" disabled={busy}>{t("observability.closeFiltered")}</Button>}
                title={t("observability.closeFilteredTitle")}
                description={t("observability.closeFilteredDescription", { count: filtered.length })}
                confirmLabel={t("observability.confirmClose")}
                confirmVariant="destructive"
                onConfirm={() => {
                  const ids = filtered.map((item) => String(item.id))
                  void run(
                    Promise.all(ids.map((id) => api.stats.closeConnection(id))),
                    t("observability.closedCount", { count: ids.length }),
                    "filtered",
                  )
                }}
              />
            ) : null}
          </div>
        </CardHeader>
        <CardContent>
          {connections.length > 0 ? (
            <Tabs defaultValue="list">
              <TabsList>
                <TabsTrigger value="list">{t("observability.listView")}</TabsTrigger>
                <TabsTrigger value="outbound">{t("observability.byOutbound")}</TabsTrigger>
                <TabsTrigger value="rule">{t("observability.byRule")}</TabsTrigger>
              </TabsList>
              <TabsContent value="list" className="mt-3">
                <ConnectionListTable
                  connections={sorted}
                  columns={columns}
                  busy={busy}
                  onClose={(id) => void run(api.stats.closeConnection(id), t("observability.close"), id)}
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
