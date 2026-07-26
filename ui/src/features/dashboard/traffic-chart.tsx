import { memo, useEffect, useMemo, useRef, useState } from "react"
import {
  CartesianGrid, Curve, Line, LineChart, XAxis, YAxis, type CurveProps, useXAxisScale, useYAxisScale,
} from "recharts"
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
const TRAFFIC_UPDATE_DURATION_MS = 800
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

interface TrafficSeriesProps extends TrafficLineProps {
  data: TrafficChartPoint[]
}

interface TrafficCurvePoint {
  x: number | null
  y: number | null
  payload?: { timestamp?: unknown }
}

interface SmoothTrafficCurveProps extends Omit<CurveProps, "points"> {
  points?: readonly TrafficCurvePoint[]
  transitionDuration?: number
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

function curveTimestamp(point: TrafficCurvePoint) {
  const timestamp = point.payload?.timestamp
  return typeof timestamp === "string" ? timestamp : undefined
}

function curveStartPoints(previous: readonly TrafficCurvePoint[], target: readonly TrafficCurvePoint[]) {
  if (previous.length === 0 || target.length === 0) return undefined
  const previousByTimestamp = new Map<string, TrafficCurvePoint>()
  for (const point of previous) {
    const timestamp = curveTimestamp(point)
    if (timestamp) previousByTimestamp.set(timestamp, point)
  }
  const tail = previous.at(-1)
  let matches = 0
  const start = target.map((point, index) => {
    const timestamp = curveTimestamp(point)
    const matched = timestamp ? previousByTimestamp.get(timestamp) : previous[index]
    if (matched) matches += 1
    const origin = matched ?? tail
    return origin ? { ...point, x: origin.x, y: origin.y } : point
  })
  return matches > 0 ? start : undefined
}

function interpolateCoordinate(start: number | null, target: number | null, progress: number) {
  if (start === null || target === null) return target
  return start + ((target - start) * progress)
}

function interpolateCurvePoints(
  start: readonly TrafficCurvePoint[],
  target: readonly TrafficCurvePoint[],
  progress: number,
) {
  return target.map((point, index) => {
    const origin = start[index] ?? point
    return {
      ...point,
      x: interpolateCoordinate(origin.x, point.x, progress),
      y: interpolateCoordinate(origin.y, point.y, progress),
    }
  })
}

function shouldSkipCurveMotion(duration: number) {
  return duration <= 0 || window.matchMedia("(prefers-reduced-motion: reduce)").matches
}

export function SmoothTrafficCurve({
  points = [],
  transitionDuration = TRAFFIC_UPDATE_DURATION_MS,
  ...curveProps
}: SmoothTrafficCurveProps) {
  const [displayedPoints, setDisplayedPoints] = useState<readonly TrafficCurvePoint[]>(points)
  const displayedPointsRef = useRef(displayedPoints)

  useEffect(() => {
    if (displayedPointsRef.current === points) return
    const start = curveStartPoints(displayedPointsRef.current, points)
    if (!start || shouldSkipCurveMotion(transitionDuration)) {
      displayedPointsRef.current = points
      setDisplayedPoints(points)
      return
    }

    const startedAt = performance.now()
    let frame = 0
    const update = (timestamp: number) => {
      const progress = Math.min(1, Math.max(0, (timestamp - startedAt) / transitionDuration))
      const next = progress === 1 ? points : interpolateCurvePoints(start, points, progress)
      displayedPointsRef.current = next
      setDisplayedPoints(next)
      if (progress < 1) frame = window.requestAnimationFrame(update)
    }
    frame = window.requestAnimationFrame(update)
    return () => window.cancelAnimationFrame(frame)
  }, [points, transitionDuration])

  return <Curve {...curveProps} points={displayedPoints} />
}

function pointNumberValue(point: TrafficChartPoint, dataKey: string) {
  const value = (point as unknown as Record<string, unknown>)[dataKey]
  return typeof value === "number" && Number.isFinite(value) ? value : undefined
}

function scaledCurvePoints(
  data: TrafficChartPoint[],
  dataKey: string,
  xScale: ReturnType<typeof useXAxisScale>,
  yScale: ReturnType<typeof useYAxisScale>,
) {
  if (!xScale || !yScale) return []
  const points: TrafficCurvePoint[] = []
  for (const point of data) {
    const value = pointNumberValue(point, dataKey)
    if (value === undefined) continue
    const x = xScale(getTimestampValue(point))
    const y = yScale(value)
    if (x === undefined || y === undefined) continue
    points.push({ x, y, payload: point })
  }
  return points
}

function TrafficLine({ dataKey }: TrafficLineProps) {
  return <Line
    id={`traffic-${dataKey}`}
    dataKey={dataKey}
    type="monotone"
    stroke={`var(--color-${dataKey})`}
    dot={false}
    activeDot={false}
    opacity={0}
    isAnimationActive={false}
    animateNewValues={false}
  />
}

function TrafficSeries({ data, dataKey }: TrafficSeriesProps) {
  const xScale = useXAxisScale()
  const yScale = useYAxisScale()
  const points = useMemo(
    () => scaledCurvePoints(data, dataKey, xScale, yScale),
    [data, dataKey, xScale, yScale],
  )
  return <SmoothTrafficCurve
    className="traffic-chart-curve"
    data-series={dataKey}
    points={points}
    type="monotone"
    fill="none"
    stroke={`var(--color-${dataKey})`}
    strokeLinecap="round"
    strokeLinejoin="round"
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
    <TrafficLine dataKey={uploadKey} />
    <TrafficLine dataKey={downloadKey} />
    <TrafficSeries data={data} dataKey={uploadKey} />
    <TrafficSeries data={data} dataKey={downloadKey} />
    <ChartTooltip content={tooltipContent} />
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
