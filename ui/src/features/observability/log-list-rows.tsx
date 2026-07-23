import { GlobeIcon, NetworkIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TableCell, TableRow } from "@/components/ui/table"
import { logConnectionsHref, logDNSHref } from "@/features/observability/connection-facets"
import type { LogEvent } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export function formatLogTimestamp(timestamp?: string) {
  if (!timestamp) return "—"
  const date = new Date(timestamp)
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString()
}

export function LogMobileCard({ item }: { item: LogEvent }) {
  const { t } = useTranslation()
  const connectionsHref = logConnectionsHref(item.message)
  const dnsHref = logDNSHref(item.message)
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
      {connectionsHref || dnsHref ? (
        <CardContent className="flex flex-wrap gap-1.5">
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
        </CardContent>
      ) : null}
    </Card>
  )
}

export function LogDesktopRow({ item }: { item: LogEvent }) {
  const { t } = useTranslation()
  const connectionsHref = logConnectionsHref(item.message)
  const dnsHref = logDNSHref(item.message)
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
        </div>
      </TableCell>
    </TableRow>
  )
}
