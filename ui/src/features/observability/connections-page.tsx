import { Trash2Icon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ConfirmAction } from "@/components/confirm-action"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAuth } from "@/features/auth/auth-context"
import { formatBytes } from "@/features/dashboard/format"
import {
  buildConnectionExportFilename,
  formatConnectionExport,
  sortConnections,
  type ConnectionSortKey,
} from "@/features/observability/connection-export"
import {
  aggregateConnections,
  matchesConnection,
  summarizeConnections,
  type ConnectionGroupStat,
} from "@/features/observability/connection-stats"
import { downloadTextFile } from "@/features/observability/log-export"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent } from "@/lib/api/types"

function formatDuration(start: string) {
  const startedAt = new Date(start).getTime()
  if (!Number.isFinite(startedAt)) return "—"
  return `${Math.floor(Math.max(0, Date.now() - startedAt) / 1000)}s`
}

function GroupStatsTable({
  groups,
  field,
  busy,
  onCloseGroup,
  emptyTitle,
  emptyDescription,
}: {
  groups: ConnectionGroupStat[]
  field: "outbound" | "rule"
  busy: boolean
  onCloseGroup: (field: "outbound" | "rule", key: string) => void
  emptyTitle: string
  emptyDescription: string
}) {
  const { t } = useTranslation()
  if (groups.length === 0) {
    return <Empty><EmptyHeader><EmptyTitle>{emptyTitle}</EmptyTitle><EmptyDescription>{emptyDescription}</EmptyDescription></EmptyHeader></Empty>
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("observability.group")}</TableHead>
          <TableHead>{t("observability.count")}</TableHead>
          <TableHead>{t("dashboard.upload")}</TableHead>
          <TableHead>{t("dashboard.download")}</TableHead>
          <TableHead>{t("common.actions")}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {groups.map((group) => (
          <TableRow key={group.key}>
            <TableCell className="max-w-[14rem] truncate" title={group.key}>{group.key}</TableCell>
            <TableCell>{group.count}</TableCell>
            <TableCell>{formatBytes(group.upload)}</TableCell>
            <TableCell>{formatBytes(group.download)}</TableCell>
            <TableCell>
              <ConfirmAction
                trigger={
                  <Button size="sm" variant="destructive" disabled={busy || group.key === "—"}>
                    {t("observability.closeGroup")}
                  </Button>
                }
                title={t("observability.closeGroupTitle", { group: group.key })}
                description={t("observability.closeGroupDescription", { count: group.count, group: group.key })}
                confirmLabel={t("observability.confirmClose")}
                confirmVariant="destructive"
                onConfirm={() => onCloseGroup(field, group.key)}
              />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function ConnectionsPage() {
  const { t } = useTranslation()
  const token = useAuth().session!.token
  const stream = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const [closingId, setClosingId] = useState<string | "all" | null>(null)
  const [query, setQuery] = useState("")
  const [sort, setSort] = useState<ConnectionSortKey>("traffic")
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
              </CardTitle>
              <CardDescription>{t("observability.connectionsDescription")}</CardDescription>
            </div>
            <div className="flex flex-wrap gap-2 text-sm text-muted-foreground">
              <span>{t("dashboard.upload")}: {formatBytes(summary.upload)}</span>
              <span>{t("dashboard.download")}: {formatBytes(summary.download)}</span>
              <span>{t("observability.outboundCount", { count: summary.outbounds })}</span>
            </div>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <label className="sr-only" htmlFor="connections-search">{t("observability.searchConnections")}</label>
            <Input
              id="connections-search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("observability.searchConnectionsPlaceholder")}
              className="sm:max-w-xs"
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
                {sorted.length === 0 ? (
                  <Empty><EmptyHeader><EmptyTitle>{t("observability.noMatch")}</EmptyTitle><EmptyDescription>{t("observability.noMatchDescription")}</EmptyDescription></EmptyHeader></Empty>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("observability.target")}</TableHead>
                        <TableHead>{t("observability.outbound")}</TableHead>
                        <TableHead>{t("observability.rule")}</TableHead>
                        <TableHead>{t("dashboard.upload")}</TableHead>
                        <TableHead>{t("dashboard.download")}</TableHead>
                        <TableHead>{t("observability.duration")}</TableHead>
                        <TableHead>{t("common.actions")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sorted.map((connection) => {
                        const id = String(connection.id)
                        return (
                          <TableRow key={connection.id}>
                            <TableCell className="max-w-[12rem] truncate" title={connection.target}>{connection.target}</TableCell>
                            <TableCell>{connection.outbound}</TableCell>
                            <TableCell className="max-w-[10rem] truncate" title={connection.rule || undefined}>{connection.rule || "—"}</TableCell>
                            <TableCell>{formatBytes(connection.upload)}</TableCell>
                            <TableCell>{formatBytes(connection.download)}</TableCell>
                            <TableCell>{formatDuration(connection.start)}</TableCell>
                            <TableCell>
                              <Button size="sm" variant="destructive" disabled={busy} onClick={() => void run(api.stats.closeConnection(id), t("observability.close"), id)}>
                                {t("observability.close")}
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                )}
              </TabsContent>
              <TabsContent value="outbound" className="mt-3">
                <GroupStatsTable
                  groups={byOutbound}
                  field="outbound"
                  busy={busy}
                  onCloseGroup={closeGroup}
                  emptyTitle={t("observability.noMatch")}
                  emptyDescription={t("observability.noMatchDescription")}
                />
              </TabsContent>
              <TabsContent value="rule" className="mt-3">
                <GroupStatsTable
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

