import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  CONNECTION_COLUMNS,
  connectionColumnVisible,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import { formatBytes } from "@/features/dashboard/format"
import type { Connection } from "@/lib/api/types"

function formatDuration(start: string) {
  const startedAt = new Date(start).getTime()
  if (!Number.isFinite(startedAt)) return "—"
  return `${Math.floor(Math.max(0, Date.now() - startedAt) / 1000)}s`
}

function cellValue(connection: Connection, id: ConnectionColumnId, duration: string): string {
  switch (id) {
    case "target":
      return connection.target || "—"
    case "source":
      return connection.source || "—"
    case "network":
      return connection.network || "—"
    case "inbound":
      return connection.inbound || "—"
    case "outbound":
      return connection.outbound || "—"
    case "rule":
      return connection.rule || "—"
    case "protocol":
      return connection.protocol || "—"
    case "process":
      return connection.process || "—"
    case "upload":
      return formatBytes(connection.upload)
    case "download":
      return formatBytes(connection.download)
    case "duration":
      return duration
    default:
      return "—"
  }
}

function titleFor(connection: Connection, id: ConnectionColumnId): string | undefined {
  if (id === "target") return connection.target || undefined
  if (id === "source") return connection.source || undefined
  if (id === "inbound") return connection.inbound || undefined
  if (id === "rule") return connection.rule || undefined
  if (id === "process") return connection.process || undefined
  return undefined
}

export function ConnectionListTable({
  connections,
  columns,
  busy,
  onClose,
}: {
  connections: Connection[]
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
}) {
  const { t } = useTranslation()
  if (connections.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{t("observability.noMatch")}</EmptyTitle>
          <EmptyDescription>{t("observability.noMatchDescription")}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }
  const visible = CONNECTION_COLUMNS.filter((column) => connectionColumnVisible(columns, column.id))
  return (
    <Table>
      <TableHeader>
        <TableRow>
          {visible.map((column) => (
            <TableHead key={column.id}>{t(column.labelKey)}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {connections.map((connection) => {
          const id = String(connection.id)
          const duration = formatDuration(connection.start)
          return (
            <TableRow key={connection.id}>
              {visible.map((column) => {
                if (column.id === "actions") {
                  return (
                    <TableCell key={column.id}>
                      <Button size="sm" variant="destructive" disabled={busy} onClick={() => onClose(id)}>
                        {t("observability.close")}
                      </Button>
                    </TableCell>
                  )
                }
                const value = cellValue(connection, column.id, duration)
                return (
                  <TableCell key={column.id} className="max-w-[12rem] truncate" title={titleFor(connection, column.id)}>
                    {value}
                  </TableCell>
                )
              })}
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}
