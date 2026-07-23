import { CopyIcon, GlobeIcon, NetworkIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TableCell, TableRow } from "@/components/ui/table"
import { logConnectionsHref, logDNSHref } from "@/features/observability/connection-facets"
import { copyText, formatLogLine, formatLogMessage } from "@/features/observability/log-export"
import type { LogEvent } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export function formatLogTimestamp(timestamp?: string) {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

function copyLogPayload(payload: string, okKey: string, failKey: string, t: (key: string) => string) {
  if (!payload) return
  void copyText(payload).then(
    () => toast.success(t(okKey)),
    () => toast.error(t(failKey)),
  )
}

export function LogCopyActions({ item }: { item: LogEvent }) {
  const { t } = useTranslation()
  const message = formatLogMessage(item)
  const line = formatLogLine(item)
  return (
    <>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="h-8"
        disabled={!message}
        onClick={() => copyLogPayload(message, "observability.logMessageCopied", "observability.logCopyFailed", t)}
        aria-label={`${t("observability.copyLogMessage")}: ${message || item.level || "log"}`}
      >
        <CopyIcon data-icon="inline-start" />
        {t("observability.copyLogMessage")}
      </Button>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="h-8"
        disabled={!line.trim()}
        onClick={() => copyLogPayload(line, "observability.logLineCopied", "observability.logCopyFailed", t)}
        aria-label={`${t("observability.copyLogLine")}: ${message || item.level || "log"}`}
      >
        <CopyIcon data-icon="inline-start" />
        {t("observability.copyLogLine")}
      </Button>
    </>
  )
}

function LogDeepLinks({ item }: { item: LogEvent }) {
  const { t } = useTranslation()
  const connectionsHref = logConnectionsHref(item.message)
  const dnsHref = logDNSHref(item.message)
  return (
    <>
      {connectionsHref ? (
        <Link
          to={connectionsHref}
          aria-label={`${t("observability.viewConnections")}: ${item.message}`}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
        >
          <NetworkIcon data-icon="inline-start" />
          {t("observability.viewConnections")}
        </Link>
      ) : null}
      {dnsHref ? (
        <Link
          to={dnsHref}
          aria-label={`${t("observability.viewDNS")}: ${item.message}`}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
        >
          <GlobeIcon data-icon="inline-start" />
          {t("observability.viewDNS")}
        </Link>
      ) : null}
    </>
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
      <CardContent className="flex flex-wrap gap-1.5">
        <LogCopyActions item={item} />
        <LogDeepLinks item={item} />
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
        <div className="flex flex-wrap gap-1">
          <LogCopyActions item={item} />
          <LogDeepLinks item={item} />
        </div>
      </TableCell>
    </TableRow>
  )
}
