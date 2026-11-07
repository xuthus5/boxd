import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { ClashModeCard } from "@/features/dashboard/clash-mode-card"
import { DashboardOverview } from "@/features/dashboard/dashboard-overview"
import { HealthSummaryCard } from "@/features/dashboard/health-summary-card"
import { SetupChecklistCard } from "@/features/dashboard/setup-checklist-card"
import { ProxySelectorCard } from "@/features/dashboard/proxy-selector-card"
import { ConfigApplyTimelineCard } from "@/features/dashboard/config-apply-timeline-card"
import { ConfigDiagnosticsCard } from "@/features/dashboard/config-diagnostics-card"
import { RuleSetHealthCard } from "@/features/dashboard/rule-set-health-card"
import { OpsShortcutsCard } from "@/features/dashboard/ops-shortcuts-card"
import { RuntimeActions } from "@/features/dashboard/runtime-actions"
import { RuntimeStatsCard } from "@/features/dashboard/runtime-stats-card"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { ServiceCard } from "@/features/dashboard/service-card"
import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { TRAFFIC_CHART_SOURCE_LIMIT } from "@/features/dashboard/traffic-chart-model"
import { useAuth } from "@/features/auth/auth-context"
import { reportDashboardRequestError } from "@/features/dashboard/dashboard-request-error-actions"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { useConnectionRates } from "@/features/observability/use-connection-rates"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent, LogEvent, ServiceStatus, TrafficEvent } from "@/lib/api/types"

const serviceActions = { start: api.service.start, stop: api.service.stop, restart: api.service.restart }

function DashboardHeader({ status }: { status?: ServiceStatus }) {
  const { t } = useTranslation()
  const running = Boolean(status?.running)
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="flex min-w-0 flex-col gap-1.5">
        <span className="flex items-center gap-2 text-[0.65rem] font-semibold tracking-[0.18em] text-muted-foreground uppercase">
          <span className="size-1.5 rounded-full bg-primary shadow-[0_0_10px_var(--primary)]" aria-hidden="true" />
          {t("dashboard.controlPlane")}
        </span>
        <h1 className="text-3xl font-semibold tracking-[-0.04em] sm:text-[2rem]">{t("pages.dashboard")}</h1>
        <p className="max-w-xl text-sm leading-relaxed text-muted-foreground">{t("dashboard.subtitle")}</p>
      </div>
      <Badge
        variant={running ? "default" : "destructive"}
        className="h-8 w-fit gap-2 rounded-full px-3.5 text-xs shadow-sm"
        aria-label={running ? t("dashboard.running") : t("dashboard.stopped")}
      >
        <span className="relative flex size-2">
          <span className={running ? "absolute inline-flex size-full animate-ping rounded-full bg-primary-foreground/50 motion-reduce:hidden" : "hidden"} />
          <span className="relative inline-flex size-2 rounded-full bg-current" />
        </span>
        {running ? t("dashboard.running") : t("dashboard.stopped")}
        {running && status?.uptime ? <span className="font-normal opacity-70">· {status.uptime}</span> : null}
      </Badge>
    </div>
  )
}

export function DashboardPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const token = useAuth().session?.token ?? ""
  const [pendingAction, setPendingAction] = useState("")
  const traffic = useStreamBuffer<TrafficEvent>(api.stats.paths.traffic, token, TRAFFIC_CHART_SOURCE_LIMIT)
  const connections = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const logs = useStreamBuffer<LogEvent>(api.stats.paths.logs, token, 20)
  const appLogs = useStreamBuffer<LogEvent>(api.stats.paths.appLogs, token, 50)
  const connectionSnapshot = useMemo(() => connections.items.at(-1), [connections.items])
  const snapshotConnections = useMemo(() => connectionSnapshot?.list ?? [], [connectionSnapshot])
  const liveConnections = useConnectionRates(snapshotConnections)
  const connectionCounts = useMemo(() => ({
    total: liveConnections.length,
    tcp: liveConnections.filter((connection) => connection.network?.toLowerCase() === "tcp").length,
    udp: liveConnections.filter((connection) => connection.network?.toLowerCase() === "udp").length,
  }), [liveConnections])
  const healthSnapshot = useMemo(() => connectionSnapshot
    ? { ...connectionSnapshot, list: liveConnections }
    : undefined, [connectionSnapshot, liveConnections])
  const [status, history, memory, version] = useQueries({ queries: [
    { queryKey: ["service"], queryFn: api.service.status, refetchInterval: 1000, staleTime: 900 },
    { queryKey: ["traffic-history"], queryFn: api.stats.history },
    { queryKey: ["memory"], queryFn: api.runtime.memory, refetchInterval: 10000 },
    { queryKey: ["version"], queryFn: api.runtime.version },
  ] })
  const readiness = useQuery({
    queryKey: ["health", "readiness"],
    queryFn: api.health.readiness,
    refetchInterval: 10000,
    staleTime: 5000,
    retry: false,
  })
  const serviceMutation = useMutation({
    mutationFn: async (action: keyof typeof serviceActions) => {
      setPendingAction(action)
      await serviceActions[action]()
    },
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["service"] }); toast.success(t("dashboard.actionComplete")) },
    onError: (error: Error, action) => reportDashboardRequestError(error, t, {
      scope: "service",
      action,
      fallback: t("dashboard.serviceActionFailed"),
    }),
    onSettled: () => setPendingAction(""),
  })
  const maintenance = useMutation({
    mutationFn: (action: () => Promise<void>) => action(),
    onSuccess: () => toast.success(t("dashboard.maintenanceComplete")),
    onError: (error: Error) => reportDashboardRequestError(error, t, {
      scope: "maintenance",
      fallback: t("dashboard.maintenanceFailed"),
    }),
  })
  const points = useMemo(
    () => [...(history.data?.points ?? []), ...traffic.items].slice(-TRAFFIC_CHART_SOURCE_LIMIT),
    [history.data?.points, traffic.items],
  )

  if ([status, history, memory, version].some((query) => query.isLoading)) {
    return (
      <div className="flex flex-col gap-5">
        <DashboardHeader />
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }, (_, index) => <Skeleton key={index} className="h-32 rounded-2xl" />)}
        </div>
        <Skeleton className="h-72 w-full rounded-2xl" />
      </div>
    )
  }
  const error = [status, history, memory, version].find((query) => query.error)?.error
  if (error) {
    return (
      <div className="flex flex-col gap-5">
        <DashboardHeader />
        <PageLoadErrorAlert
          error={error}
          scope="dashboard"
          titleKey="dashboard.loadFailed"
          onRetry={() => {
            for (const item of [status, history, memory, version]) {
              if (item.error) void item.refetch()
            }
          }}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-5">
      <DashboardHeader status={status.data} />
      <DashboardOverview
        points={points}
        activeConnections={connectionCounts.total}
        tcpConnections={connectionCounts.tcp}
        udpConnections={connectionCounts.udp}
        memory={memory.data!}
        running={Boolean(status.data?.running)}
      />
      <SetupChecklistCard status={status.data} />
      <div className="grid items-start gap-5 md:grid-cols-2 xl:grid-cols-12">
        <aside data-testid="dashboard-status-column" className="order-1 flex min-w-0 flex-col gap-5 md:order-2 xl:col-span-4">
          <ServiceCard
            status={status.data!}
            pending={pendingAction}
            onAction={(action) => serviceMutation.mutate(action)}
            readiness={{
              isLoading: readiness.isLoading,
              isFetching: readiness.isFetching,
              ready: readiness.data?.status === "ready" && !readiness.error,
              error: readiness.error,
              onRetry: () => { void readiness.refetch() },
            }}
          />
          <ProxySelectorCard />
          <HealthSummaryCard
            snapshot={healthSnapshot}
            status={status.data}
            streamError={connections.error || undefined}
            streamStatus={connections.status}
            onReconnect={connections.reconnect}
          />
          <ClashModeCard enabled={Boolean(status.data?.running)} />
          <RuntimeStatsCard memory={memory.data!} panelVersion={version.data!.version} kernelVersion={version.data!.kernel_version} />
          <RuntimeActions pending={maintenance.isPending} onGC={() => maintenance.mutate(api.runtime.gc)} onFlushDNS={() => maintenance.mutate(api.runtime.flushDNS)} onFlushFakeIP={() => maintenance.mutate(api.runtime.flushFakeIP)} />
        </aside>
        <section data-testid="dashboard-main-column" className="order-2 flex min-w-0 flex-col gap-5 md:order-1 xl:col-span-8">
          <TrafficChart
            points={points}
            streamError={traffic.error || undefined}
            streamStatus={traffic.status}
            streamPath={api.stats.paths.traffic}
            onReconnect={traffic.reconnect}
          />
          <ConfigDiagnosticsCard />
          <RuleSetHealthCard
            appLogs={appLogs.items}
            appLogError={appLogs.error}
            appLogStatus={appLogs.status}
            onReconnectAppLogs={appLogs.reconnect}
          />
          <ConfigApplyTimelineCard />
        </section>
      </div>
      <OpsShortcutsCard />
      <RecentLogs
        items={logs.items}
        streamError={logs.error || undefined}
        streamStatus={logs.status}
        streamPath={api.stats.paths.logs}
        onReconnect={logs.reconnect}
      />
    </div>
  )
}
