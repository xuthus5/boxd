import { useState } from "react"
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart"
import {
  Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { formatLatency } from "@/features/nodes/node-format"
import {
  latencyHistoryChartRows as chartRows,
  latencyHistoryTypes as availableTypes,
  summarizeLatencyHistory as summarize,
} from "@/features/nodes/latency-history-model"
import { latencyBadgeVariant, latencyTone, latencyToneClass } from "@/features/nodes/latency-style"
import type { LatencyPoint } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface LatencyHistoryDialogProps {
  tag: string
  history?: Record<string, LatencyPoint[]>
  triggerClassName?: string
}

export function LatencyHistoryDialog({ tag, history, triggerClassName }: LatencyHistoryDialogProps) {
  const { t } = useTranslation()
  const types = availableTypes(history)
  const [open, setOpen] = useState(false)
  const [type, setType] = useState(types[0] ?? "tcp")
  const config = {
    latency: { label: t("nodes.latency"), color: "var(--chart-1)" },
  } satisfies ChartConfig
  const disabled = types.length === 0

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className={cn("h-auto px-0 text-xs text-muted-foreground hover:text-foreground", triggerClassName)}
        disabled={disabled}
        onClick={() => {
          if (types[0]) setType(types[0])
          setOpen(true)
        }}
      >
        {t("nodes.latencyHistory")}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl" data-testid="latency-history-dialog">
          <DialogHeader>
            <DialogTitle>{t("nodes.latencyHistoryTitle", { tag })}</DialogTitle>
            <DialogDescription>{t("nodes.latencyHistoryDescription")}</DialogDescription>
          </DialogHeader>
          {types.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("nodes.latencyHistoryEmpty")}</p>
          ) : (
            <Tabs value={type} onValueChange={setType}>
              <TabsList className="max-w-full">
                {types.map((item) => (
                  <TabsTrigger key={item} value={item}>{item.toUpperCase()}</TabsTrigger>
                ))}
              </TabsList>
              {types.map((item) => {
                const series = history?.[item] ?? []
                const seriesStats = summarize(series)
                const seriesRows = chartRows(series)
                return (
                  <TabsContent key={item} value={item} className="flex flex-col gap-3">
                    <div className="flex flex-wrap gap-2">
                      <Badge variant="outline">{t("nodes.historySamples", { count: seriesStats.count })}</Badge>
                      <Badge variant="secondary">{t("nodes.historySuccess", { count: seriesStats.success })}</Badge>
                      {seriesStats.latest !== undefined ? (
                        <Badge
                          variant={latencyBadgeVariant(latencyTone(seriesStats.latest, true))}
                          className={cn(latencyToneClass(latencyTone(seriesStats.latest, true)))}
                        >
                          {t("nodes.historyLatest", { value: formatLatency(seriesStats.latest) })}
                        </Badge>
                      ) : null}
                      {seriesStats.avg !== undefined ? (
                        <Badge variant="outline">{t("nodes.historyAvg", { value: formatLatency(seriesStats.avg) })}</Badge>
                      ) : null}
                      {seriesStats.min !== undefined && seriesStats.max !== undefined ? (
                        <Badge variant="outline">
                          {t("nodes.historyRange", {
                            min: formatLatency(seriesStats.min),
                            max: formatLatency(seriesStats.max),
                          })}
                        </Badge>
                      ) : null}
                    </div>
                    {seriesRows.filter((row) => row.latency !== null).length < 2 ? (
                      <p className="text-sm text-muted-foreground">{t("nodes.latencyHistoryNeedMore")}</p>
                    ) : (
                      <ChartContainer config={config} className="h-56 w-full">
                        <LineChart accessibilityLayer data={seriesRows}>
                          <CartesianGrid vertical={false} />
                          <YAxis
                            width={56}
                            tickLine={false}
                            axisLine={false}
                            tickFormatter={(value: number) => `${Math.round(value)}`}
                          />
                          <XAxis
                            dataKey="timestamp"
                            tickLine={false}
                            axisLine={false}
                            minTickGap={24}
                            tickFormatter={(value: string) => {
                              const date = new Date(value)
                              return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleTimeString()
                            }}
                          />
                          <ChartTooltip
                            content={
                              <ChartTooltipContent
                                formatter={(value) => (value == null ? "—" : formatLatency(Number(value)))}
                                labelFormatter={(label) => {
                                  const date = new Date(String(label))
                                  return Number.isNaN(date.getTime()) ? String(label) : date.toLocaleString()
                                }}
                              />
                            }
                          />
                          <Line
                            dataKey="latency"
                            type="monotone"
                            stroke="var(--color-latency)"
                            dot={false}
                            connectNulls={false}
                          />
                        </LineChart>
                      </ChartContainer>
                    )}
                    <ul className="max-h-40 space-y-1 overflow-y-auto text-xs text-muted-foreground">
                      {[...series].reverse().slice(0, 12).map((point, index) => (
                        <li
                          key={`${point.timestamp}-${index}`}
                          className="flex items-center justify-between gap-2 border-b border-border/50 py-1"
                        >
                          <span>{point.timestamp ? new Date(point.timestamp).toLocaleString() : "—"}</span>
                          <span>
                            {point.success
                              ? (point.latency_ms === undefined ? t("common.normal") : formatLatency(point.latency_ms))
                              : (point.error || t("nodes.testFailed"))}
                          </span>
                        </li>
                      ))}
                    </ul>
                  </TabsContent>
                )
              })}
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}

