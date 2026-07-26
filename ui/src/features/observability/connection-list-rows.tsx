import { CopyIcon, NetworkIcon, RouteIcon, ScrollTextIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TableCell, TableRow } from "@/components/ui/table"
import { formatBytes } from "@/features/dashboard/format"
import { formatConnectionRatePair, type ConnectionWithRates } from "@/features/observability/connection-rate"
import {
  CONNECTION_COLUMNS,
  connectionColumnVisible,
  type ConnectionColumnId,
} from "@/features/observability/connection-columns"
import { FacetLink, MetaChip } from "@/features/observability/connection-facet-links"
import { formatConnectionClipboardText } from "@/features/observability/connection-export"
import { reportExportError } from "@/features/observability/export-error-actions"
import {
  cellValue,
  formatDuration,
  nodeHref,
  ruleRouteHref,
  targetLogsHref,
  titleFor,
} from "@/features/observability/connection-list-helpers"
import { copyText } from "@/features/proxy/copy-tag-button"
import { cn } from "@/lib/utils"

function copyConnectionDiagnostics(connection: ConnectionWithRates, t: (key: string) => string) {
  const payload = formatConnectionClipboardText(connection)
  if (!payload) return
  void copyText(payload).then(
    () => toast.success(t("observability.connectionCopied")),
    (error: unknown) => reportExportError(error, t, {
      scope: "connections",
      kind: "copy-connection",
      fallback: t("observability.connectionCopyFailed"),
    }),
  )
}

export function ConnectionMobileCard({
  connection,
  columns,
  busy,
  onClose,
}: {
  connection: ConnectionWithRates
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
}) {
  const { t } = useTranslation()
  const id = String(connection.id)
  const duration = formatDuration(connection.start)
  const show = (column: ConnectionColumnId) => connectionColumnVisible(columns, column)
  return (
    <Card size="sm" className="h-full overflow-hidden">
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
                  (error: unknown) => reportExportError(error, t, {
                    scope: "connections",
                    kind: "copy-target",
                    fallback: t("observability.targetCopyFailed"),
                  }),
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
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <ScrollTextIcon data-icon="inline-start" />
              {t("observability.viewTargetLogs")}
            </Link>
          ) : null}
          {nodeHref(connection.outbound) ? (
            <Link
              to={nodeHref(connection.outbound)}
              aria-label={`${t("observability.viewNode")}: ${connection.outbound}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <NetworkIcon data-icon="inline-start" />
              {t("observability.viewNode")}
            </Link>
          ) : null}
          {ruleRouteHref(connection.rule) ? (
            <Link
              to={ruleRouteHref(connection.rule)}
              aria-label={`${t("observability.viewRule")}: ${connection.rule}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <RouteIcon data-icon="inline-start" />
              {t("observability.viewRule")}
            </Link>
          ) : null}
          <Button
            type="button"
            size="sm"
            className="h-8"
            variant="outline"
            aria-label={`${t("observability.copyConnection")}: ${connection.target || id}`}
            onClick={() => copyConnectionDiagnostics(connection, t)}
          >
            <CopyIcon data-icon="inline-start" />
            {t("observability.copyConnection")}
          </Button>
          <Button size="sm" className="h-8" variant="destructive" disabled={busy} onClick={() => onClose(id)}>
            {t("observability.close")}
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <div className="flex flex-wrap gap-1.5">
          {show("network") ? <MetaChip field="network" label={t("observability.network")} value={connection.network || "—"} /> : null}
          {show("protocol") ? <MetaChip field="protocol" label={t("observability.protocol")} value={connection.protocol || "—"} /> : null}
          {show("inbound") ? <MetaChip field="inbound" label={t("observability.inbound")} value={connection.inbound || "—"} /> : null}
          {show("source") ? <MetaChip label={t("observability.source")} value={connection.source || "—"} /> : null}
          {show("process") ? <MetaChip field="process" label={t("observability.process")} value={connection.process || "—"} /> : null}
        </div>
        <div className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
          {show("upload") ? <span>{t("dashboard.upload")}: {formatBytes(connection.upload)}</span> : null}
          {show("download") ? <span>{t("dashboard.download")}: {formatBytes(connection.download)}</span> : null}
          {show("rate") ? <span>{t("observability.rate")}: {formatConnectionRatePair(connection.uploadRate, connection.downloadRate)}</span> : null}
          {show("duration") ? <span>{t("observability.duration")}: {duration}</span> : null}
        </div>
      </CardContent>
    </Card>
  )
}

export function ConnectionDesktopRow({
  connection,
  columns,
  busy,
  onClose,
}: {
  connection: ConnectionWithRates
  columns: readonly ConnectionColumnId[]
  busy: boolean
  onClose: (id: string) => void
}) {
  const { t } = useTranslation()
  const id = String(connection.id)
  const duration = formatDuration(connection.start)
  const visible = CONNECTION_COLUMNS.filter((column) => connectionColumnVisible(columns, column.id))
  return (
    <TableRow>
      {visible.map((column) => {
        if (column.id === "actions") {
          return (
            <TableCell key={column.id}>
              <div className="flex flex-wrap items-center gap-1">
                {connection.target ? (
                  <Link
                    to={targetLogsHref(connection.target)}
                    aria-label={`${t("observability.viewTargetLogs")}: ${connection.target}`}
                    className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
                  >
                    <ScrollTextIcon data-icon="inline-start" />
                    {t("observability.viewTargetLogs")}
                  </Link>
                ) : null}
                {nodeHref(connection.outbound) ? (
                  <Link
                    to={nodeHref(connection.outbound)}
                    aria-label={`${t("observability.viewNode")}: ${connection.outbound}`}
                    className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
                  >
                    <NetworkIcon data-icon="inline-start" />
                    {t("observability.viewNode")}
                  </Link>
                ) : null}
                {ruleRouteHref(connection.rule) ? (
                  <Link
                    to={ruleRouteHref(connection.rule)}
                    aria-label={`${t("observability.viewRule")}: ${connection.rule}`}
                    className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
                  >
                    <RouteIcon data-icon="inline-start" />
                    {t("observability.viewRule")}
                  </Link>
                ) : null}
                <Button
                  type="button"
                  size="sm"
                  className="h-8"
                  variant="outline"
                  aria-label={`${t("observability.copyConnection")}: ${connection.target || id}`}
                  onClick={() => copyConnectionDiagnostics(connection, t)}
                >
                  <CopyIcon data-icon="inline-start" />
                  {t("observability.copyConnection")}
                </Button>
                <Button size="sm" className="h-8" variant="destructive" disabled={busy} onClick={() => onClose(id)}>
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
                      (error: unknown) => reportExportError(error, t, {
                        scope: "connections",
                        kind: "copy-target",
                        fallback: t("observability.targetCopyFailed"),
                      }),
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
          return (
            <TableCell key={column.id} className="max-w-[12rem] truncate">
              <FacetLink
                field={column.id}
                value={value === "—" ? undefined : value}
                label={t(column.labelKey)}
                className="block truncate"
              />
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
}
