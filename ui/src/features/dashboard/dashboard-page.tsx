import { useMutation, useQueries, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { RuntimeActions } from "@/features/dashboard/runtime-actions"
import { RecentLogs } from "@/features/dashboard/recent-logs"
import { ServiceCard } from "@/features/dashboard/service-card"
import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { formatBytes } from "@/features/dashboard/format"
import { useAuth } from "@/features/auth/auth-context"
import { useStreamBuffer } from "@/features/observability/use-stream-buffer"
import { api } from "@/lib/api/endpoints"
import type { LogEvent, TrafficEvent } from "@/lib/api/types"

const serviceActions = { start: api.service.start, stop: api.service.stop, restart: api.service.restart }

export function DashboardPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const token = useAuth().session!.token
  const [pendingAction, setPendingAction] = useState("")
  const traffic = useStreamBuffer<TrafficEvent>(api.stats.paths.traffic, token, 60)
  const logs = useStreamBuffer<LogEvent>(api.stats.paths.logs, token, 5)
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
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => setPendingAction(""),
  })
  const maintenance = useMutation({ mutationFn: (action: () => Promise<void>) => action(), onSuccess: () => toast.success(t("dashboard.maintenanceComplete")), onError: (error: Error) => toast.error(error.message) })
  const points = useMemo(() => [...(history.data?.points ?? []), ...traffic.items].slice(-60), [history.data?.points, traffic.items])

  if ([status, history, memory, version].some((query) => query.isLoading)) return <div className="flex flex-col gap-4"><h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1><Skeleton className="h-64 w-full" /></div>
  const error = [status, history, memory, version].find((query) => query.error)?.error
  if (error) return <div className="flex flex-col gap-4"><h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1><Alert variant="destructive"><AlertTitle>{t("dashboard.loadFailed")}</AlertTitle><AlertDescription>{error.message}</AlertDescription></Alert></div>

  return (
    <div className="flex flex-col gap-4">
      <h1 className="text-2xl font-semibold">{t("pages.dashboard")}</h1>
      <div className="grid gap-4 lg:grid-cols-3">
        <ServiceCard status={status.data!} pending={pendingAction} onAction={(action) => serviceMutation.mutate(action)} />
        <Card><CardHeader><CardTitle>{t("dashboard.memory")}</CardTitle><CardDescription>{t("dashboard.memoryDescription")}</CardDescription></CardHeader><CardContent><p className="text-2xl font-semibold">{formatBytes(memory.data!.alloc)}</p></CardContent></Card>
        <Card><CardHeader><CardTitle>{t("dashboard.version")}</CardTitle><CardDescription>{t("dashboard.versionDescription")}</CardDescription></CardHeader><CardContent><p className="text-2xl font-semibold">{version.data!.kernel_version}</p></CardContent></Card>
        <TrafficChart points={points} />
        <RuntimeActions pending={maintenance.isPending} onGC={() => maintenance.mutate(api.runtime.gc)} onFlushDNS={() => maintenance.mutate(api.runtime.flushDNS)} onFlushFakeIP={() => maintenance.mutate(api.runtime.flushFakeIP)} />
        <RecentLogs items={logs.items} />
      </div>
    </div>
  )
}
