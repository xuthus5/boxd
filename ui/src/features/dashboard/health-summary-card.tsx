import { ActivityIcon } from "lucide-react"
import { useMemo, type ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { CardQueryError } from "@/features/common/card-query-error"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { formatBytes } from "@/features/dashboard/format"
import { FailedSubscriptionsPreview } from "@/features/dashboard/failed-subscriptions-preview"
import { HealthApplyFailurePreview } from "@/features/dashboard/health-apply-failure-preview"
import { HealthOpsAlertActions, HealthOpsAlertChips } from "@/features/dashboard/health-ops-alerts"
import { HealthStreamErrorBlock } from "@/features/dashboard/health-stream-error-block"
import { ProblemNodesPreview } from "@/features/dashboard/problem-nodes-preview"
import { buildHealthSummary, type HealthTone } from "@/features/dashboard/health-summary"
import { useHealthOpsSignals } from "@/features/dashboard/use-health-ops-signals"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { api } from "@/lib/api/endpoints"
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

function DeepLink({ to, children, className }: { to: string; children: ReactNode; className?: string }) {
  if (!to) return <span className={className}>{children}</span>
  return <Link to={to} className={cn("underline-offset-4 hover:underline", className)}>{children}</Link>
}

export function HealthSummaryCard({
  snapshot,
  status,
  streamError,
  streamStatus,
  onReconnect,
}: {
  snapshot?: ConnectionEvent
  status?: ServiceStatus
  streamError?: string
  streamStatus?: string
  onReconnect?: () => void
}) {
  const { t } = useTranslation()
  const summary = useMemo(() => buildHealthSummary(snapshot, status), [snapshot, status])
  const { signals: ops, queryError, queryScope, onRetry } = useHealthOpsSignals()
  const topOutboundHref = summary.topOutbound && summary.topOutbound !== "—"
    ? buildConnectionsHref({ outbound: summary.topOutbound })
    : ""
  const topRuleHref = summary.topRule && summary.topRule !== "—"
    ? buildConnectionsHref({ rule: summary.topRule })
    : ""
  const tcpHref = summary.tcp > 0 ? buildConnectionsHref({ network: "tcp" }) : ""
  const udpHref = summary.udp > 0 ? buildConnectionsHref({ network: "udp" }) : ""
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex min-w-0 items-center gap-2 truncate">
          <ActivityIcon className="size-4 shrink-0" aria-hidden="true" />
          <span className="truncate">{t("dashboard.healthTitle")}</span>
        </CardTitle>
        <CardDescription>{t("dashboard.healthDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={toneVariant(summary.tone)}>{t(toneLabelKey(summary.tone))}</Badge>
          <span className="text-2xl font-semibold">{t("dashboard.healthActive", { count: summary.active })}</span>
        </div>
        <div className="grid gap-2 text-sm text-muted-foreground sm:grid-cols-2">
          <p>{t("dashboard.upload")}: {formatBytes(summary.upload)}</p>
          <p>{t("dashboard.download")}: {formatBytes(summary.download)}</p>
          <p>{t("dashboard.healthOutbounds", { count: summary.outbounds })}</p>
          <p>
            {t("dashboard.healthNetworksPrefix")}{" "}
            <DeepLink to={tcpHref}>TCP {summary.tcp}</DeepLink>
            {" · "}
            <DeepLink to={udpHref}>UDP {summary.udp}</DeepLink>
          </p>
          <p className="truncate" title={summary.topOutbound}>
            {t("dashboard.healthTopOutbound")}:{" "}
            <DeepLink to={topOutboundHref}>{summary.topOutbound}</DeepLink>
          </p>
          <p className="truncate" title={summary.topRule}>
            {t("dashboard.healthTopRule")}:{" "}
            <DeepLink to={topRuleHref}>{summary.topRule}</DeepLink>
          </p>
        </div>
        {queryError ? (
          <CardQueryError
            error={queryError}
            scope={queryScope || "health-ops"}
            onRetry={onRetry}
          />
        ) : null}
        <HealthOpsAlertChips signals={ops} />
        <HealthApplyFailurePreview
          event={ops.latestApplyFailure}
          count={ops.applyFailures}
        />
        <FailedSubscriptionsPreview
          items={ops.failedSubscriptionItems}
          total={ops.failedSubscriptions}
        />
        <ProblemNodesPreview
          items={ops.problemNodeItems}
          total={ops.problemNodes}
        />
        <HealthStreamErrorBlock
          error={streamError}
          status={streamStatus}
          path={api.stats.paths.connections}
          href="/observability/connections"
          onReconnect={onReconnect}
        />
      </CardContent>
      <CardFooter className="flex flex-wrap gap-2">
        <Link to="/observability/connections" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}>
          {t("dashboard.healthOpenConnections")}
        </Link>
        <HealthOpsAlertActions signals={ops} />
      </CardFooter>
    </Card>
  )
}
