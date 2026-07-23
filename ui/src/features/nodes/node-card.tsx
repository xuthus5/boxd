import { ActivityIcon, GaugeIcon, ScrollTextIcon } from "lucide-react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useId } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { LatencyHistoryDialog } from "@/features/nodes/latency-history-dialog"
import { formatLatency } from "@/features/nodes/node-format"
import { LatencyHealthBar } from "@/features/nodes/latency-health-bar"
import { LatencySparkline } from "@/features/nodes/latency-sparkline"
import { pickNodeHistorySeries } from "@/features/nodes/nodes-filter"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { latencyBadgeVariant, latencyTone, latencyToneClass } from "@/features/nodes/latency-style"
import { cn } from "@/lib/utils"
import { nodeTestInput, nodeTestTypes, type NodeTestType } from "@/features/nodes/node-test-inputs"
import { api } from "@/lib/api/endpoints"
import type { LatencyPoint, Outbound, TestResult } from "@/lib/api/types"

type TestChoice = NodeTestType | "all"

function ResultBadge({ result }: { result?: TestResult }) {
  const { t } = useTranslation()
  if (!result) return <Badge variant="outline">—</Badge>
  if (!result.success) return <Badge variant="destructive">{result.error || t("nodes.testFailed")}</Badge>
  const tone = latencyTone(result.latency_ms, true)
  const label = result.latency_ms === undefined ? t("common.normal") : formatLatency(result.latency_ms)
  return <Badge variant={latencyBadgeVariant(tone)} className={cn(latencyToneClass(tone))}>{label}</Badge>
}

function TestResults({ results }: { results?: Record<string, TestResult> }) {
  return (
    <dl className="flex flex-wrap gap-x-3 gap-y-1">
      {nodeTestTypes.map((type) => (
        <div key={type} className="flex items-center gap-1.5">
          <dt className="text-xs text-muted-foreground">{type.toUpperCase()}</dt>
          <dd><ResultBadge result={results?.[type]} /></dd>
        </div>
      ))}
    </dl>
  )
}

function TestControls({ node }: { node: Outbound }) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const available = Boolean(node.server && node.port)
  const mutation = useMutation({
    mutationFn: async (choice: TestChoice) => {
      if (choice === "all") return api.nodes.testBatch(nodeTestTypes.map((type) => nodeTestInput(node, type)!), 3)
      return api.nodes.test(nodeTestInput(node, choice)!)
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ["nodes", "results"] })
      void client.invalidateQueries({ queryKey: ["nodes", "history"] })
    },
    onError: (error: Error) => toast.error(error.message),
  })
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" size="xs" disabled={!available || mutation.isPending} />}>
        <GaugeIcon data-icon="inline-start" />{t("nodes.test")}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={() => mutation.mutate("all")}>{t("nodes.testAll")}</DropdownMenuItem>
          {nodeTestTypes.map((type) => (
            <DropdownMenuItem key={type} onClick={() => mutation.mutate(type)}>{type.toUpperCase()}</DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function NodeCard({
  node,
  results,
  history,
}: {
  node: Outbound
  results?: Record<string, TestResult>
  history?: Record<string, LatencyPoint[]>
}) {
  const { t } = useTranslation()
  const titleId = useId()
  const subscription = node.source === "subscription"
  const source = subscription ? node.source_name || t("nodes.subscription") : t("nodes.imported")
  const series = pickNodeHistorySeries(history)
  const endpoint = `${node.server ?? "—"}:${node.port ?? "—"}`
  return (
    <article aria-labelledby={titleId}>
      <Card size="sm" className="h-full">
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle className="min-w-0">
            <h3 id={titleId} className="truncate" title={node.tag}>{node.tag}</h3>
          </CardTitle>
          <CardDescription className="truncate" title={endpoint}>{endpoint}</CardDescription>
          <CardAction>
            <div className="flex items-center gap-1.5">
              <Badge variant="outline" className="max-w-20 truncate">{node.type}</Badge>
              <TestControls node={node} />
            </div>
          </CardAction>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant="secondary" className="max-w-full truncate">{source}</Badge>
          </div>
          <div className="flex flex-wrap gap-1.5">
            <Link
              to={buildConnectionsHref({ outbound: node.tag })}
              aria-label={`${t("nodes.viewConnections")}: ${node.tag}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <ActivityIcon data-icon="inline-start" />
              {t("nodes.viewConnections")}
            </Link>
            <Link
              to={buildLogsHref({ query: node.tag })}
              aria-label={`${t("nodes.viewLogs")}: ${node.tag}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <ScrollTextIcon data-icon="inline-start" />
              {t("nodes.viewLogs")}
            </Link>
          </div>
          <TestResults results={results} />
          <div className="space-y-1">
            <div className="flex items-center justify-between gap-2">
              <LatencyHistoryDialog tag={node.tag} history={history} />
              <span className="text-[11px] text-muted-foreground">
                {series.length
                  ? t("nodes.historySamples", { count: series.length })
                  : t("nodes.latencyHistoryEmpty")}
              </span>
            </div>
            <LatencyHealthBar points={series} />
            <LatencySparkline points={series} aria-label={t("nodes.latencyHistoryFor", { tag: node.tag })} />
          </div>
        </CardContent>
      </Card>
    </article>
  )
}
