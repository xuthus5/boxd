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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAuth } from "@/features/auth/auth-context"
import { formatBytes } from "@/features/dashboard/format"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { Connection, ConnectionEvent } from "@/lib/api/types"

function formatDuration(start: string) {
  const startedAt = new Date(start).getTime()
  if (!Number.isFinite(startedAt)) return "—"
  const milliseconds = Math.max(0, Date.now() - startedAt)
  return `${Math.floor(milliseconds / 1000)}s`
}

function matchesConnection(connection: Connection, query: string) {
  if (!query) return true
  const haystack = [
    connection.target,
    connection.outbound,
    connection.rule ?? "",
    String(connection.id),
  ].join(" ").toLowerCase()
  return haystack.includes(query)
}

export function ConnectionsPage() {
  const { t } = useTranslation()
  const token = useAuth().session!.token
  const stream = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const [closingId, setClosingId] = useState<string | "all" | null>(null)
  const [query, setQuery] = useState("")
  const snapshot = stream.items.at(-1)
  const connections = snapshot?.list ?? []
  const normalized = query.trim().toLowerCase()
  const filtered = useMemo(
    () => connections.filter((connection) => matchesConnection(connection, normalized)),
    [connections, normalized],
  )
  const action = async (request: Promise<unknown>, message: string, id: string | "all" = "all") => {
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

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("observability.connections")}</h1>
        <ConfirmAction
          trigger={<Button className="w-full sm:w-auto" variant="destructive" disabled={closingId !== null}><Trash2Icon data-icon="inline-start" />{t("observability.closeAll")}</Button>}
          title={t("observability.closeAllTitle")}
          description={t("observability.closeAllDescription")}
          confirmLabel={t("observability.confirmClose")}
          confirmVariant="destructive"
          onConfirm={() => action(api.stats.closeAll(), t("observability.closeAll"))}
        />
      </div>
      {stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}
      <Card>
        <CardHeader className="gap-3">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <CardTitle>{t("observability.liveConnections")} <Badge variant="secondary">{snapshot?.active_connections ?? 0}</Badge></CardTitle>
              <CardDescription>{t("observability.connectionsDescription")}</CardDescription>
            </div>
            <div className="w-full sm:max-w-sm">
              <label className="sr-only" htmlFor="connections-search">{t("observability.searchConnections")}</label>
              <Input
                id="connections-search"
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t("observability.searchConnectionsPlaceholder")}
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {connections.length === 0
            ? <Empty><EmptyHeader><EmptyTitle>{t("observability.noConnections")}</EmptyTitle><EmptyDescription>{t("observability.noConnectionsDescription")}</EmptyDescription></EmptyHeader></Empty>
            : filtered.length === 0
              ? <Empty><EmptyHeader><EmptyTitle>{t("observability.noMatch")}</EmptyTitle><EmptyDescription>{t("observability.noMatchDescription")}</EmptyDescription></EmptyHeader></Empty>
              : <Table>
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
                  {filtered.map((connection) => {
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
                          <Button size="sm" variant="destructive" disabled={closingId !== null} onClick={() => { void action(api.stats.closeConnection(id), t("observability.close"), id) }}>
                            {t("observability.close")}
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>}
        </CardContent>
      </Card>
    </div>
  )
}
