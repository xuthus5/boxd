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

const TrafficLines = memo(function TrafficLines({ data, uploadKey, downloadKey, formatter, config }: { data: object[]; uploadKey: string; downloadKey: string; formatter: (value: number) => string; config: ChartConfig }) {
  return <ChartContainer config={config} className="h-64 w-full"><LineChart accessibilityLayer data={data}>
    <CartesianGrid vertical={false} />
    <YAxis width={72} tickLine={false} axisLine={false} tickFormatter={(value: number) => formatter(value)} />
    <XAxis dataKey="timestamp" tickLine={false} axisLine={false} tickFormatter={(value: string) => new Date(value).toLocaleTimeString()} />
    <ChartTooltip content={<ChartTooltipContent formatter={(value) => formatter(Number(value))} />} />
    <Line dataKey={uploadKey} type="monotone" stroke={`var(--color-${uploadKey})`} dot={false} isAnimationActive={false} />
    <Line dataKey={downloadKey} type="monotone" stroke={`var(--color-${downloadKey})`} dot={false} isAnimationActive={false} />
  </LineChart></ChartContainer>
})

export const TrafficChart = memo(function TrafficChart({
  points,
  streamError,
  streamStatus,
  streamPath,
  onReconnect,
}: {
  points: TrafficHistoryPoint[]
  streamError?: string
  streamStatus?: string
  streamPath?: string
  onReconnect?: () => void
}) {
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
