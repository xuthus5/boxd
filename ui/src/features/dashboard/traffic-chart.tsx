import { ActivityIcon } from "lucide-react"
import { memo, useMemo, useState } from "react"
import { LineChart } from "recharts"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { formatBytes } from "@/features/dashboard/format"
import { HealthStreamErrorBlock } from "@/features/dashboard/health-stream-error-block"
import {
  getTrafficTimeDomain,
  getTrafficValueDomain,
  TRAFFIC_CHART_POINT_LIMIT,
  TRAFFIC_CHART_MARGIN,
  formatTrafficTimestamp,
  type TrafficChartPoint,
} from "@/features/dashboard/traffic-chart-model"
import { TrafficPlot } from "@/features/dashboard/traffic-chart-plot"
import { calculateTrafficRates } from "@/features/dashboard/traffic-rate"
import type { TrafficHistoryPoint } from "@/lib/api/types"

function formatRate(value: number) {
  return `${formatBytes(value)}/s`
}

interface TrafficLinesProps {
  data: TrafficChartPoint[]
  uploadKey: string
  downloadKey: string
  formatter: (value: number) => string
  config: ChartConfig
}

interface TrafficChartProps {
  points: TrafficHistoryPoint[]
  streamError?: string
  streamStatus?: string
  streamPath?: string
  onReconnect?: () => void
}

const TrafficLines = memo(function TrafficLines({ data, uploadKey, downloadKey, formatter, config }: TrafficLinesProps) {
  const tooltipContent = useMemo(() => (
    <ChartTooltipContent
      labelFormatter={formatTrafficTimestamp}
      formatter={(value) => formatter(Number(value))}
    />
  ), [formatter])
  const timeDomain = useMemo(() => getTrafficTimeDomain(data), [data])
  const valueDomain = useMemo(
    () => getTrafficValueDomain(data, [uploadKey, downloadKey]),
    [data, downloadKey, uploadKey],
  )

  return <ChartContainer config={config} className="h-72 w-full sm:h-80"><LineChart accessibilityLayer data={data} margin={TRAFFIC_CHART_MARGIN}>
    <TrafficPlot
      data={data}
      uploadKey={uploadKey}
      downloadKey={downloadKey}
      formatter={formatter}
      timeDomain={timeDomain}
      valueDomain={valueDomain}
    />
    <ChartTooltip includeHidden content={tooltipContent} />
  </LineChart></ChartContainer>
})

export const TrafficChart = memo(function TrafficChart({
  points,
  streamError,
  streamStatus,
  streamPath,
  onReconnect,
}: TrafficChartProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<"rate" | "total">("rate")
  const visiblePoints = useMemo(() => points.slice(-TRAFFIC_CHART_POINT_LIMIT), [points])
  const rates = useMemo(
    () => calculateTrafficRates(points).slice(-TRAFFIC_CHART_POINT_LIMIT),
    [points],
  )
  const totalConfig = useMemo(() => ({
    upload_bytes: { label: t("dashboard.upload"), color: "var(--chart-2)" },
    download_bytes: { label: t("dashboard.download"), color: "var(--chart-1)" },
  } satisfies ChartConfig), [t])
  const rateConfig = useMemo(() => ({
    upload_rate: { label: t("dashboard.upload"), color: "var(--chart-2)" },
    download_rate: { label: t("dashboard.download"), color: "var(--chart-1)" },
  } satisfies ChartConfig), [t])
  const latestTotal = visiblePoints.at(-1)
  const latestRate = rates.at(-1)
  const description = mode === "rate"
    ? `${t("dashboard.upload")} ${formatRate(latestRate?.upload_rate ?? 0)} · ${t("dashboard.download")} ${formatRate(latestRate?.download_rate ?? 0)}`
    : `${t("dashboard.upload")} ${formatBytes(latestTotal?.upload_bytes ?? 0)} · ${t("dashboard.download")} ${formatBytes(latestTotal?.download_bytes ?? 0)}`
  return (
    <Card size="sm" className="overflow-hidden">
      <Tabs value={mode} onValueChange={(value) => setMode(String(value) as "rate" | "total")}>
      <CardHeader className="gap-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
        <CardTitle className="flex items-center gap-2 truncate">
          <span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-chart-1/10 text-chart-1 ring-1 ring-chart-1/15"><ActivityIcon className="size-4" /></span>
          {t("dashboard.traffic")}
        </CardTitle>
        <CardDescription>{description}</CardDescription>
          </div>
          <span className="mt-1 flex shrink-0 items-center gap-1.5 rounded-full border border-success/20 bg-success/8 px-2.5 py-1 text-[0.65rem] font-medium text-success">
            <span className="size-1.5 animate-pulse rounded-full bg-success motion-reduce:animate-none" />
            {t("dashboard.realtimeRate")}
          </span>
        </div>
        <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
          <TabsTrigger value="rate">{t("dashboard.realtimeRate")}</TabsTrigger>
          <TabsTrigger value="total">{t("dashboard.cumulativeTraffic")}</TabsTrigger>
        </TabsList>
      </CardHeader>
      <TabsContent value="rate"><CardContent className="rounded-2xl bg-muted/20 p-2 sm:p-3"><TrafficLines data={rates} uploadKey="upload_rate" downloadKey="download_rate" formatter={formatRate} config={rateConfig} /></CardContent></TabsContent>
      <TabsContent value="total"><CardContent className="rounded-2xl bg-muted/20 p-2 sm:p-3"><TrafficLines data={visiblePoints} uploadKey="upload_bytes" downloadKey="download_bytes" formatter={formatBytes} config={totalConfig} /></CardContent></TabsContent>
      </Tabs>
      {streamError || streamStatus === "reconnecting" ? (
        <CardContent className="pt-0">
          <HealthStreamErrorBlock
            error={streamError}
            status={streamStatus}
            path={streamPath}
            onReconnect={onReconnect}
          />
        </CardContent>
      ) : null}
    </Card>
  )
})
