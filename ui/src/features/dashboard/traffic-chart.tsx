import { memo, useMemo, useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { formatBytes } from "@/features/dashboard/format"
import { HealthStreamErrorBlock } from "@/features/dashboard/health-stream-error-block"
import { calculateTrafficRates } from "@/features/dashboard/traffic-rate"
import type { TrafficHistoryPoint } from "@/lib/api/types"

function formatRate(value: number) {
  return `${formatBytes(value)}/s`
}

const TRAFFIC_WINDOW_MS = 60_000
const TRAFFIC_CHART_MARGIN = { top: 4, right: 8, bottom: 0, left: 0 }

type TrafficChartPoint = { timestamp: string }

interface TrafficLinesProps {
  data: TrafficChartPoint[]
  uploadKey: string
  downloadKey: string
  formatter: (value: number) => string
  config: ChartConfig
}

interface TrafficLineProps {
  dataKey: string
}

interface TrafficChartProps {
  points: TrafficHistoryPoint[]
  streamError?: string
  streamStatus?: string
  streamPath?: string
  onReconnect?: () => void
}

function timestampToMilliseconds(value: unknown) {
  if (typeof value !== "string") return 0
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : 0
}

function getTimestampValue(point: TrafficChartPoint) {
  return timestampToMilliseconds(point.timestamp)
}

function formatTimestamp(value: unknown) {
  const timestamp = typeof value === "number" ? value : timestampToMilliseconds(value)
  return timestamp > 0 ? new Date(timestamp).toLocaleTimeString() : ""
}

function getTimeDomain(data: readonly TrafficChartPoint[]): [number, number] {
  const latest = data.reduce((current, point) => Math.max(current, getTimestampValue(point)), 0)
  if (latest <= 0) return [0, TRAFFIC_WINDOW_MS]
  return [latest - TRAFFIC_WINDOW_MS, latest]
}

function TrafficLine({ dataKey }: TrafficLineProps) {
  return <Line
    id={`traffic-${dataKey}`}
    dataKey={dataKey}
    type="monotone"
    stroke={`var(--color-${dataKey})`}
    dot={false}
    isAnimationActive={false}
    animateNewValues={false}
  />
}

const TrafficLines = memo(function TrafficLines({ data, uploadKey, downloadKey, formatter, config }: TrafficLinesProps) {
  const tooltipContent = useMemo(() => (
    <ChartTooltipContent
      labelFormatter={formatTimestamp}
      formatter={(value) => formatter(Number(value))}
    />
  ), [formatter])
  const timeDomain = useMemo(() => getTimeDomain(data), [data])

  return <ChartContainer config={config} className="h-64 w-full"><LineChart accessibilityLayer data={data} margin={TRAFFIC_CHART_MARGIN}>
    <CartesianGrid vertical={false} />
    <YAxis width={72} tickLine={false} axisLine={false} tickFormatter={formatter} />
    <XAxis
      type="number"
      dataKey={getTimestampValue}
      domain={timeDomain}
      allowDataOverflow
      tickLine={false}
      axisLine={false}
      tickFormatter={formatTimestamp}
    />
    <ChartTooltip content={tooltipContent} />
    <TrafficLine dataKey={uploadKey} />
    <TrafficLine dataKey={downloadKey} />
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
  const rates = useMemo(() => calculateTrafficRates(points), [points])
  const totalConfig = useMemo(() => ({
    upload_bytes: { label: t("dashboard.upload"), color: "var(--chart-2)" },
    download_bytes: { label: t("dashboard.download"), color: "var(--chart-1)" },
  } satisfies ChartConfig), [t])
  const rateConfig = useMemo(() => ({
    upload_rate: { label: t("dashboard.upload"), color: "var(--chart-2)" },
    download_rate: { label: t("dashboard.download"), color: "var(--chart-1)" },
  } satisfies ChartConfig), [t])
  const latestTotal = points.at(-1)
  const latestRate = rates.at(-1)
  const description = mode === "rate"
    ? `${t("dashboard.upload")} ${formatRate(latestRate?.upload_rate ?? 0)} · ${t("dashboard.download")} ${formatRate(latestRate?.download_rate ?? 0)}`
    : `${t("dashboard.upload")} ${formatBytes(latestTotal?.upload_bytes ?? 0)} · ${t("dashboard.download")} ${formatBytes(latestTotal?.download_bytes ?? 0)}`
  return (
    <Card size="sm" className="lg:col-span-2">
      <Tabs value={mode} onValueChange={(value) => setMode(String(value) as "rate" | "total")}>
      <CardHeader className="gap-2 sm:gap-3">
        <CardTitle className="truncate">{t("dashboard.traffic")}</CardTitle>
        <CardDescription>{description}</CardDescription>
        <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
          <TabsTrigger value="rate">{t("dashboard.realtimeRate")}</TabsTrigger>
          <TabsTrigger value="total">{t("dashboard.cumulativeTraffic")}</TabsTrigger>
        </TabsList>
      </CardHeader>
      <TabsContent value="rate"><CardContent><TrafficLines data={rates} uploadKey="upload_rate" downloadKey="download_rate" formatter={formatRate} config={rateConfig} /></CardContent></TabsContent>
      <TabsContent value="total"><CardContent><TrafficLines data={points} uploadKey="upload_bytes" downloadKey="download_bytes" formatter={formatBytes} config={totalConfig} /></CardContent></TabsContent>
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
