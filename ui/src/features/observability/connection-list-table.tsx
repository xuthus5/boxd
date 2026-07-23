import { CopyIcon, ScrollTextIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  CONNECTION_COLUMNS,
  connectionColumnVisible,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import { formatBytes } from "@/features/dashboard/format"
import { connectionTargetLogQuery } from "@/features/observability/connection-facets"
import { FacetLink, MetaChip } from "@/features/observability/connection-facet-links"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { copyText } from "@/features/proxy/copy-tag-button"
import { useIsMobile } from "@/hooks/use-mobile"
import type { Connection } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function targetLogsHref(target?: string) {
  const query = connectionTargetLogQuery(target)
  return query ? buildLogsHref({ query }) : ""
}

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

function ConnectionMobileCard({
  connection,
  columns,
  busy,
  onClose,
}: {
  connection: Connection
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
}) {
  const { t } = useTranslation()
  const id = String(connection.id)
  const duration = formatDuration(connection.start)
  const show = (column: ConnectionColumnId) => connectionColumnVisible(columns, column)
  return (
    <Card size="sm">
      <CardHeader className="min-w-0">
        <CardTitle className="flex min-w-0 items-center gap-1">
          <span className="truncate" title={connection.target || undefined}>{connection.target || "—"}</span>
          {connection.target ? (
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="shrink-0"
              aria-label={`${t("observability.copyTarget")}: ${connection.target}`}
              onClick={() => {
                void copyText(connection.target!).then(
                  () => toast.success(t("observability.targetCopied")),
                  () => toast.error(t("observability.targetCopyFailed")),
                )
              }}
            >
              <CopyIcon className="size-3.5" />
            </Button>
          ) : null}
        </CardTitle>
        <CardDescription className="truncate">
          {show("outbound") ? (
            <FacetLink field="outbound" value={connection.outbound} label={t("observability.outbound")} />
          ) : null}
          {show("outbound") && show("rule") ? " · " : null}
          {show("rule") ? (
            <FacetLink field="rule" value={connection.rule} label={t("observability.rule")} />
          ) : null}
        </CardDescription>
        <CardAction className="flex flex-wrap justify-end gap-1">
          {connection.target ? (
            <Link
              to={targetLogsHref(connection.target)}
              aria-label={`${t("observability.viewTargetLogs")}: ${connection.target}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
            >
              <ScrollTextIcon data-icon="inline-start" />
              {t("observability.viewTargetLogs")}
            </Link>
          ) : null}
          <Button size="sm" variant="destructive" disabled={busy} onClick={() => onClose(id)}>
            {t("observability.close")}
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <div className="flex flex-wrap gap-1.5">
          {show("network") ? <MetaChip field="network" label={t("observability.network")} value={connection.network || "—"} /> : null}
          {show("protocol") ? <MetaChip field="protocol" label={t("observability.protocol")} value={connection.protocol || "—"} /> : null}
          {show("inbound") ? <MetaChip label={t("observability.inbound")} value={connection.inbound || "—"} /> : null}
          {show("source") ? <MetaChip label={t("observability.source")} value={connection.source || "—"} /> : null}
          {show("process") ? <MetaChip field="process" label={t("observability.process")} value={connection.process || "—"} /> : null}
        </div>
        <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
          {show("upload") ? <span>{t("dashboard.upload")}: {formatBytes(connection.upload)}</span> : null}
          {show("download") ? <span>{t("dashboard.download")}: {formatBytes(connection.download)}</span> : null}
          {show("duration") ? <span>{t("observability.duration")}: {duration}</span> : null}
        </div>
      </CardContent>
    </Card>
  )
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
  const isMobile = useIsMobile()
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
  if (isMobile) {
    return (
      <div className="flex flex-col gap-2">
        {connections.map((connection) => (
          <ConnectionMobileCard
            key={connection.id}
            connection={connection}
            columns={columns}
            busy={busy}
            onClose={onClose}
          />
        ))}
      </div>
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
                      <div className="flex flex-wrap items-center gap-1">
                        {connection.target ? (
                          <Link
                            to={targetLogsHref(connection.target)}
                            aria-label={`${t("observability.viewTargetLogs")}: ${connection.target}`}
                            className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
                          >
                            <ScrollTextIcon data-icon="inline-start" />
                            {t("observability.viewTargetLogs")}
                          </Link>
                        ) : null}
                        <Button size="sm" variant="destructive" disabled={busy} onClick={() => onClose(id)}>
                          {t("observability.close")}
                        </Button>
                      </div>
                    </TableCell>
                  )
                }
                const value = cellValue(connection, column.id, duration)
                if (column.id === "target" && connection.target) {
                  return (
                    <TableCell key={column.id} className="max-w-[14rem]">
                      <div className="flex min-w-0 items-center gap-1">
                        <span className="truncate" title={connection.target}>{value}</span>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-sm"
                          className="shrink-0"
                          aria-label={`${t("observability.copyTarget")}: ${connection.target}`}
                          onClick={() => {
                            void copyText(connection.target!).then(
                              () => toast.success(t("observability.targetCopied")),
                              () => toast.error(t("observability.targetCopyFailed")),
                            )
                          }}
                        >
                          <CopyIcon className="size-3.5" />
                        </Button>
                      </div>
                    </TableCell>
                  )
                }
                if (column.id === "network" || column.id === "protocol" || column.id === "outbound" || column.id === "rule" || column.id === "process") {
                  const label = t(column.labelKey)
                  return (
                    <TableCell key={column.id} className="max-w-[12rem] truncate">
                      <FacetLink field={column.id} value={value === "—" ? undefined : value} label={label} className="block truncate" />
                    </TableCell>
                  )
                }
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
