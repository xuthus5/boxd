import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  formatDelayValue,
  sortDelayEntries,
  type DelayMap,
} from "@/features/dashboard/proxy-delay"
import { useProxySelector } from "@/features/dashboard/use-proxy-selector"
import { formatLatency } from "@/features/nodes/node-format"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import type { OutboundGroup } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function DelayBadge({ delay, failed }: { delay?: number; failed?: boolean }) {
  const { t } = useTranslation()
  if (failed) return <Badge variant="destructive">{t("dashboard.proxyDelayFailed")}</Badge>
  if (delay === undefined) return <Badge variant="outline">—</Badge>
  return <Badge variant="secondary">{formatLatency(delay)}</Badge>
}

function ProxyStatusCard({ title, description }: { title: string; description: string }) {
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
    </Card>
  )
}

function DelayList({ delays }: { delays: DelayMap }) {
  const { t } = useTranslation()
  const entries = useMemo(() => sortDelayEntries(delays), [delays])
  if (!entries.length) return null
  const failedLabel = t("dashboard.proxyDelayFailed")
  return (
    <ul className="flex max-h-36 flex-col gap-1 overflow-auto text-xs text-muted-foreground">
      {entries.map(([tag, delay]) => (
        <li key={tag} className="flex items-center justify-between gap-2">
          <span className="min-w-0 truncate" title={tag}>{tag}</span>
          <span className={cn("shrink-0 tabular-nums", delay === "error" && "text-destructive")}>
            {formatDelayValue(delay, failedLabel)}
          </span>
        </li>
      ))}
    </ul>
  )
}

function CurrentOutboundLinks({ tag }: { tag: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap gap-1.5">
      <Link
        to={buildConnectionsHref({ outbound: tag })}
        aria-label={`${t("nodes.viewConnections")}: ${tag}`}
        className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
      >
        {t("nodes.viewConnections")}
      </Link>
      <Link
        to={buildLogsHref({ query: tag })}
        aria-label={`${t("nodes.viewLogs")}: ${tag}`}
        className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
      >
        {t("nodes.viewLogs")}
      </Link>
      <Link
        to={buildNodesHref({ query: tag })}
        aria-label={`${t("observability.viewNode")}: ${tag}`}
        className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
      >
        {t("observability.viewNode")}
      </Link>
    </div>
  )
}

function MemberSelect({
  group,
  members,
  delays,
  onSelect,
  selecting,
}: {
  group: OutboundGroup
  members: readonly string[]
  delays: DelayMap
  onSelect: (tag: string) => void
  selecting: boolean
}) {
  const { t } = useTranslation()
  const items = members.map((tag) => ({ label: tag, value: tag }))
  const failedLabel = t("dashboard.proxyDelayFailed")
  return (
    <Select items={items} value={group.now} onValueChange={(value) => onSelect(String(value))}>
      <SelectTrigger aria-label={t("dashboard.proxySelector")} className="h-8 w-full" disabled={selecting}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          {items.map((item) => {
            const delay = delays[item.value]
            const suffix = delay === undefined ? "" : ` (${formatDelayValue(delay, failedLabel)})`
            return <SelectItem key={item.value} value={item.value}>{item.label}{suffix}</SelectItem>
          })}
        </SelectGroup>
      </SelectContent>
    </Select>
  )
}

export function ProxySelectorCard() {
  const { t } = useTranslation()
  const state = useProxySelector()
  if (state.query.isLoading) return <Skeleton className="h-36 w-full" />
  if (state.query.error) {
    return <ProxyStatusCard title={t("dashboard.proxySelector")} description={state.query.error.message} />
  }
  if (!state.group) {
    return <ProxyStatusCard title={t("dashboard.proxySelector")} description={t("dashboard.proxySelectorEmpty")} />
  }
  const currentDelay = state.delays[state.group.now]
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="truncate">{t("dashboard.proxySelector")}</span>
          <Badge variant="outline">{state.group.tag}</Badge>
          <DelayBadge
            delay={typeof currentDelay === "number" ? currentDelay : undefined}
            failed={currentDelay === "error"}
          />
        </CardTitle>
        <CardDescription>{t("dashboard.proxySelectorDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <MemberSelect
          group={state.group}
          members={state.members}
          delays={state.delays}
          onSelect={state.select}
          selecting={state.selecting}
        />
        <Button
          variant="outline"
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={state.probing || state.members.length === 0}
          onClick={state.probe}
        >
          {state.probing ? t("dashboard.proxyDelayTesting") : t("dashboard.proxyDelayTest")}
        </Button>
        <DelayList delays={state.delays} />
        {state.group.now ? <CurrentOutboundLinks tag={state.group.now} /> : null}
      </CardContent>
    </Card>
  )
}
