import { CartesianGrid, Line, LineChart, XAxis } from "recharts"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import { formatBytes } from "@/features/dashboard/format"
import type { TrafficHistoryPoint } from "@/lib/api/types"

export function TrafficChart({ points }: { points: TrafficHistoryPoint[] }) {
  const { t } = useTranslation()
  const chartConfig = {
    upload_bytes: { label: t("dashboard.upload"), color: "var(--chart-2)" },
    download_bytes: { label: t("dashboard.download"), color: "var(--chart-1)" },
  } satisfies ChartConfig
  const latest = points.at(-1)
  return (
    <Card className="lg:col-span-2">
      <CardHeader>
        <CardTitle>{t("dashboard.traffic")}</CardTitle>
        <CardDescription>{t("dashboard.upload")} {formatBytes(latest?.upload_bytes ?? 0)} · {t("dashboard.download")} {formatBytes(latest?.download_bytes ?? 0)}</CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="h-64 w-full">
          <LineChart accessibilityLayer data={points}>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="timestamp" tickLine={false} axisLine={false} tickFormatter={(value: string) => new Date(value).toLocaleTimeString()} />
            <ChartTooltip content={<ChartTooltipContent formatter={(value) => formatBytes(Number(value))} />} />
            <Line dataKey="upload_bytes" type="monotone" stroke="var(--color-upload_bytes)" dot={false} />
            <Line dataKey="download_bytes" type="monotone" stroke="var(--color-download_bytes)" dot={false} />
          </LineChart>
        </ChartContainer>
      </CardContent>
    </Card>
  )
}
