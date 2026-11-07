import { ArrowDownIcon, ArrowUpIcon, CpuIcon, RadioTowerIcon } from "lucide-react"
import type { ComponentType } from "react"
import { useTranslation } from "react-i18next"

import { Card, CardContent } from "@/components/ui/card"
import { formatBytes } from "@/features/dashboard/format"
import { calculateTrafficRates } from "@/features/dashboard/traffic-rate"
import type { MemoryStats, TrafficHistoryPoint } from "@/lib/api/types"
import { cn } from "@/lib/utils"

type MetricTone = "download" | "upload" | "live" | "memory"

interface MetricCardProps {
  icon: ComponentType<{ className?: string }>
  label: string
  value: string
  detail: string
  tone: MetricTone
  testId: string
}

const toneClasses: Record<MetricTone, string> = {
  download: "bg-chart-1/12 text-chart-1 ring-chart-1/20",
  upload: "bg-chart-2/12 text-chart-2 ring-chart-2/20",
  live: "bg-success/12 text-success ring-success/20",
  memory: "bg-warning/15 text-warning ring-warning/25 dark:text-warning",
}

function MetricCard({ icon: Icon, label, value, detail, tone, testId }: MetricCardProps) {
  return (
    <Card size="sm" className="dashboard-kpi relative gap-0 py-0" data-testid={testId}>
      <CardContent className="relative z-10 flex flex-col gap-4 p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-medium tracking-[0.02em] text-muted-foreground">{label}</p>
            <p className="mt-2 truncate text-2xl font-semibold tracking-[-0.035em] tabular-nums sm:text-[1.7rem]">{value}</p>
          </div>
          <span className={cn("flex size-10 shrink-0 items-center justify-center rounded-2xl ring-1", toneClasses[tone])}>
            <Icon className="size-4" />
          </span>
        </div>
        <p className="truncate text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  )
}

export function DashboardOverview({
  points,
  activeConnections,
  tcpConnections,
  udpConnections,
  memory,
  running,
}: {
  points: TrafficHistoryPoint[]
  activeConnections: number
  tcpConnections: number
  udpConnections: number
  memory: MemoryStats
  running: boolean
}) {
  const { t } = useTranslation()
  const latest = points.at(-1)
  const rate = calculateTrafficRates(points).at(-1)
  return (
    <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label={t("dashboard.controlPlane")}>
      <MetricCard
        icon={ArrowUpIcon}
        label={t("dashboard.overviewUpload")}
        value={`${formatBytes(rate?.upload_rate ?? 0)}/s`}
        detail={t("dashboard.overviewCumulative", { value: formatBytes(latest?.upload_bytes ?? 0) })}
        tone="upload"
        testId="dashboard-metric-upload"
      />
      <MetricCard
        icon={ArrowDownIcon}
        label={t("dashboard.overviewDownload")}
        value={`${formatBytes(rate?.download_rate ?? 0)}/s`}
        detail={t("dashboard.overviewCumulative", { value: formatBytes(latest?.download_bytes ?? 0) })}
        tone="download"
        testId="dashboard-metric-download"
      />
      <MetricCard
        icon={RadioTowerIcon}
        label={t("dashboard.overviewConnections")}
        value={String(activeConnections)}
        detail={running ? t("dashboard.overviewTcpUdp", { tcp: tcpConnections, udp: udpConnections }) : t("dashboard.stopped")}
        tone="live"
        testId="dashboard-metric-connections"
      />
      <MetricCard
        icon={CpuIcon}
        label={t("dashboard.overviewMemory")}
        value={formatBytes(memory.alloc)}
        detail={t("dashboard.overviewGc", { count: memory.num_gc, goroutines: memory.num_goroutine ?? 0 })}
        tone="memory"
        testId="dashboard-metric-memory"
      />
    </section>
  )
}
