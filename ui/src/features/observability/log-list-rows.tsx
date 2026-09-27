import { CopyIcon, EllipsisIcon, GlobeIcon, NetworkIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { TableCell, TableRow } from "@/components/ui/table"
import { logConnectionsHref, logDNSHref } from "@/features/observability/connection-facets"
import { reportExportError } from "@/features/observability/export-error-actions"
import { formatLogLine, formatLogMessage, formatLogTimestamp } from "@/features/observability/log-export"
import type { LogEvent } from "@/lib/api/types"
import { copyText } from "@/lib/clipboard"

function copyLogPayload(payload: string, okKey: string, failKey: string, t: (key: string) => string) {
  if (!payload) return
  void copyText(payload).then(
    () => toast.success(t(okKey)),
    (error: unknown) => reportExportError(error, t, {
      scope: "logs",
      kind: "copy-line",
      fallback: t(failKey),
    }),
  )
}

// 日志行操作统一收纳进菜单，避免操作列占用过多宽度。
export function LogActionsMenu({ item, deepLinks = true }: { item: LogEvent; deepLinks?: boolean }) {
  const { t } = useTranslation()
  const message = formatLogMessage(item)
  const line = formatLogLine(item)
  const subject = message || item.level || "log"
  const connectionsHref = deepLinks ? logConnectionsHref(item.message) : ""
  const dnsHref = deepLinks ? logDNSHref(item.message) : ""
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="outline" size="icon-sm" aria-label={t("observability.moreLogActions", { subject })} />}
      >
        <EllipsisIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem
            disabled={!message}
            aria-label={`${t("observability.copyLogMessage")}: ${subject}`}
            onClick={() => copyLogPayload(message, "observability.logMessageCopied", "observability.logCopyFailed", t)}
          >
            <CopyIcon />{t("observability.copyLogMessage")}
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={!line.trim()}
            aria-label={`${t("observability.copyLogLine")}: ${subject}`}
            onClick={() => copyLogPayload(line, "observability.logLineCopied", "observability.logCopyFailed", t)}
          >
            <CopyIcon />{t("observability.copyLogLine")}
          </DropdownMenuItem>
          {connectionsHref || dnsHref ? <DropdownMenuSeparator /> : null}
          {connectionsHref ? (
            <DropdownMenuItem
              aria-label={`${t("observability.viewConnections")}: ${item.message}`}
              render={<Link to={connectionsHref} />}
            >
              <NetworkIcon />{t("observability.viewConnections")}
            </DropdownMenuItem>
          ) : null}
          {dnsHref ? (
            <DropdownMenuItem
              aria-label={`${t("observability.viewDNS")}: ${item.message}`}
              render={<Link to={dnsHref} />}
            >
              <GlobeIcon />{t("observability.viewDNS")}
            </DropdownMenuItem>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function LogMobileCard({ item }: { item: LogEvent }) {
  return (
    <Card size="sm" className="h-full overflow-hidden">
      <CardHeader className="min-w-0 gap-1">
        <CardTitle className="flex flex-wrap items-center gap-2 text-sm font-medium">
          <Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge>
          <time className="text-muted-foreground" dateTime={item.timestamp || undefined}>
            {formatLogTimestamp(item.timestamp)}
          </time>
        </CardTitle>
        <CardDescription className="line-clamp-3 whitespace-normal break-words text-foreground">
          {item.message}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <LogActionsMenu item={item} />
      </CardContent>
    </Card>
  )
}

export function LogDesktopRow({ item }: { item: LogEvent }) {
  return (
    <TableRow>
      <TableCell className="whitespace-nowrap text-muted-foreground">
        <time dateTime={item.timestamp || undefined}>{formatLogTimestamp(item.timestamp)}</time>
      </TableCell>
      <TableCell>
        <Badge variant={item.level === "error" ? "destructive" : "secondary"}>{item.level}</Badge>
      </TableCell>
      <TableCell className="min-w-64 whitespace-normal break-words">
        <span className="line-clamp-2">{item.message}</span>
      </TableCell>
      <TableCell>
        <LogActionsMenu item={item} />
      </TableCell>
    </TableRow>
  )
}
