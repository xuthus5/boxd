import { useMutation, useQueries, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Skeleton } from "@/components/ui/skeleton"
import { ClashModeCard } from "@/features/dashboard/clash-mode-card"
import { HealthSummaryCard } from "@/features/dashboard/health-summary-card"
import { SetupChecklistCard } from "@/features/dashboard/setup-checklist-card"
import { ProxySelectorCard } from "@/features/dashboard/proxy-selector-card"
import { ConfigApplyTimelineCard } from "@/features/dashboard/config-apply-timeline-card"
import { OpsShortcutsCard } from "@/features/dashboard/ops-shortcuts-card"
import { RuntimeActions } from "@/features/dashboard/runtime-actions"
import { RuntimeStatsCard } from "@/features/dashboard/runtime-stats-card"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { ServiceCard } from "@/features/dashboard/service-card"
import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { useAuth } from "@/features/auth/auth-context"
import { reportDashboardRequestError } from "@/features/dashboard/dashboard-request-error-actions"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { ConnectionEvent, LogEvent, TrafficEvent } from "@/lib/api/types"

const serviceActions = { start: api.service.start, stop: api.service.stop, restart: api.service.restart }

export function DashboardPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const token = useAuth().session!.token
  const [pendingAction, setPendingAction] = useState("")
  const traffic = useStreamBuffer<TrafficEvent>(api.stats.paths.traffic, token, 60)
  const connections = useStreamBuffer<ConnectionEvent>(api.stats.paths.connections, token, 2)
  const logs = useStreamBuffer<LogEvent>(api.stats.paths.logs, token, 20)
  const [status, history, memory, version] = useQueries({ queries: [
    { queryKey: ["service"], queryFn: api.service.status, refetchInterval: 5000 },
    { queryKey: ["traffic-history"], queryFn: api.stats.history },
    { queryKey: ["memory"], queryFn: api.runtime.memory, refetchInterval: 10000 },
    { queryKey: ["version"], queryFn: api.runtime.version },
  ] })
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
  const points = useMemo(() => [...(history.data?.points ?? []), ...traffic.items].slice(-60), [history.data?.points, traffic.items])

  if ([status, history, memory, version].some((query) => query.isLoading)) return <div className="flex flex-col gap-4"><h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1><Skeleton className="h-64 w-full" /></div>
  const error = [status, history, memory, version].find((query) => query.error)?.error
  if (error) {
    return (
      <div className="flex flex-col gap-4">
        <h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1>
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
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1>
      <div className="grid gap-4 lg:grid-cols-3">
        <SetupChecklistCard status={status.data} />
        <HealthSummaryCard snapshot={connections.items.at(-1)} status={status.data} streamError={connections.error || undefined} streamStatus={connections.status} />
        <ServiceCard status={status.data!} pending={pendingAction} onAction={(action) => serviceMutation.mutate(action)} />
        <ProxySelectorCard />
        <ClashModeCard enabled={Boolean(status.data?.running)} />
        <RuntimeStatsCard memory={memory.data!} panelVersion={version.data!.version} kernelVersion={version.data!.kernel_version} />
        <OpsShortcutsCard />
        <ConfigApplyTimelineCard />
        <TrafficChart points={points} streamError={traffic.error || undefined} streamStatus={traffic.status} streamPath={api.stats.paths.traffic} />
        <RuntimeActions pending={maintenance.isPending} onGC={() => maintenance.mutate(api.runtime.gc)} onFlushDNS={() => maintenance.mutate(api.runtime.flushDNS)} onFlushFakeIP={() => maintenance.mutate(api.runtime.flushFakeIP)} />
        <RecentLogs items={logs.items} streamError={logs.error || undefined} streamStatus={logs.status} streamPath={api.stats.paths.logs} />
      </div>
    </div>
  )
}
