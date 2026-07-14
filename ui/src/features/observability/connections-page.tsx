import { Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ConfirmAction } from "@/components/confirm-action"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useAuth } from "@/features/auth/auth-context"
import { formatBytes } from "@/features/dashboard/format"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent } from "@/lib/api/types"

function formatDuration(start: string) {
  const startedAt = new Date(start).getTime()
  if (!Number.isFinite(startedAt)) return "—"
  const milliseconds = Math.max(0, Date.now() - startedAt)
  return `${Math.floor(milliseconds / 1000)}s`
}

export function ConnectionsPage() {
  const { t } = useTranslation()
  const token = useAuth().session!.token
  const stream = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const snapshot = stream.items.at(-1)
  const connections = snapshot?.list ?? []
  const action = (request: Promise<unknown>, message: string) => request.then(() => toast.success(message)).catch((error: Error) => toast.error(error.message))
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between"><h1 className="text-2xl font-semibold">{t("observability.connections")}</h1><ConfirmAction trigger={<Button variant="destructive"><Trash2Icon data-icon="inline-start" />{t("observability.closeAll")}</Button>} title={t("observability.closeAllTitle")} description={t("observability.closeAllDescription")} confirmLabel={t("observability.confirmClose")} confirmVariant="destructive" onConfirm={() => action(api.stats.closeAll(), t("observability.closeAll"))} /></div>
      {stream.error ? <Alert variant="destructive"><AlertTitle>{t("observability.streamError")}</AlertTitle><AlertDescription>{stream.error}</AlertDescription></Alert> : null}
      <Card><CardHeader><CardTitle>{t("observability.liveConnections")} <Badge variant="secondary">{snapshot?.active_connections ?? 0}</Badge></CardTitle><CardDescription>{t("observability.connectionsDescription")}</CardDescription></CardHeader><CardContent>
        {connections.length === 0 ? <Empty><EmptyHeader><EmptyTitle>{t("observability.noConnections")}</EmptyTitle><EmptyDescription>{t("observability.noConnectionsDescription")}</EmptyDescription></EmptyHeader></Empty> : <Table><TableHeader><TableRow><TableHead>{t("observability.target")}</TableHead><TableHead>{t("observability.outbound")}</TableHead><TableHead>{t("dashboard.upload")}</TableHead><TableHead>{t("dashboard.download")}</TableHead><TableHead>{t("observability.duration")}</TableHead><TableHead>{t("common.actions")}</TableHead></TableRow></TableHeader><TableBody>{connections.map((connection) => <TableRow key={connection.id}><TableCell>{connection.target}</TableCell><TableCell>{connection.outbound}</TableCell><TableCell>{formatBytes(connection.upload)}</TableCell><TableCell>{formatBytes(connection.download)}</TableCell><TableCell>{formatDuration(connection.start)}</TableCell><TableCell><Button size="sm" variant="destructive" onClick={() => action(api.stats.closeConnection(String(connection.id)), t("observability.close"))}>{t("observability.close")}</Button></TableCell></TableRow>)}</TableBody></Table>}
      </CardContent></Card>
    </div>
  )
}
