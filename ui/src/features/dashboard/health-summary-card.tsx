import { ActivityIcon } from "lucide-react"
import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { formatBytes } from "@/features/dashboard/format"
import { buildHealthSummary, type HealthTone } from "@/features/dashboard/health-summary"
import type { ConnectionEvent, ServiceStatus } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function toneVariant(tone: HealthTone): "default" | "secondary" | "destructive" | "outline" {
  if (tone === "offline") return "destructive"
  if (tone === "warn") return "outline"
  if (tone === "ok") return "default"
  return "secondary"
}

function toneLabelKey(tone: HealthTone) {
  if (tone === "offline") return "dashboard.healthOffline"
  if (tone === "warn") return "dashboard.healthBusy"
  if (tone === "ok") return "dashboard.healthHealthy"
  return "dashboard.healthIdle"
}

export function HealthSummaryCard({
  snapshot,
  status,
  streamError,
}: {
  snapshot?: ConnectionEvent
  status?: ServiceStatus
  streamError?: string
}) {
  const { t } = useTranslation()
  const summary = useMemo(() => buildHealthSummary(snapshot, status), [snapshot, status])
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ActivityIcon className="size-4" aria-hidden="true" />
          {t("dashboard.healthTitle")}
        </CardTitle>
        <CardDescription>{t("dashboard.healthDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={toneVariant(summary.tone)}>{t(toneLabelKey(summary.tone))}</Badge>
          <span className="text-2xl font-semibold">{t("dashboard.healthActive", { count: summary.active })}</span>
        </div>
        <div className="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
          <p>{t("dashboard.upload")}: {formatBytes(summary.upload)}</p>
          <p>{t("dashboard.download")}: {formatBytes(summary.download)}</p>
          <p>{t("dashboard.healthOutbounds", { count: summary.outbounds })}</p>
          <p>{t("dashboard.healthNetworks", { tcp: summary.tcp, udp: summary.udp })}</p>
          <p className="truncate" title={summary.topOutbound}>{t("dashboard.healthTopOutbound")}: {summary.topOutbound}</p>
          <p className="truncate" title={summary.topRule}>{t("dashboard.healthTopRule")}: {summary.topRule}</p>
        </div>
        {streamError ? <p className="text-sm text-destructive">{streamError}</p> : null}
      </CardContent>
      <CardFooter>
        <Link to="/observability/connections" className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          {t("dashboard.healthOpenConnections")}
        </Link>
      </CardFooter>
    </Card>
  )
}
