import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  delayErrorHintKey,
  delayFailureClipboardText,
  formatDelayValue,
  isDelayFailure,
  sortDelayEntries,
  type DelayMap,
} from "@/features/dashboard/proxy-delay"
import { copyText } from "@/lib/clipboard"
import { toast } from "sonner"
import { CardQueryError } from "@/features/common/card-query-error"
import { useProxySelector } from "@/features/dashboard/use-proxy-selector"
import { formatLatency } from "@/features/nodes/node-format"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildConnectionsHref } from "@/features/observability/connection-facets"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import type { OutboundGroup } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function DelayBadge({ value }: { value?: DelayMap[string] }) {
  const { t } = useTranslation()
  if (isDelayFailure(value)) {
    const label = formatDelayValue(value, t("dashboard.proxyDelayFailed"))
    return (
      <Badge
        variant="destructive"
        className="max-w-[10rem] truncate"
        title={`${label}
${t(delayErrorHintKey(value.code))}`}
      >
        {label}
      </Badge>
    )
  }
  if (value === undefined) return <Badge variant="outline">—</Badge>
  return <Badge variant="secondary">{formatLatency(value)}</Badge>
}

function ProxyStatusCard({
  title,
  description,
  error,
  onRetry,
}: {
  title: string
  description?: string
  error?: unknown
  onRetry?: () => void
}) {
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{title}</CardTitle>
        {description && !error ? <CardDescription>{description}</CardDescription> : null}
      </CardHeader>
      {error ? (
        <CardContent>
          <CardQueryError error={error} scope="proxy-selector" onRetry={onRetry} />
        </CardContent>
      ) : null}
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
      {entries.map(([tag, delay]) => {
        const failed = isDelayFailure(delay)
        const label = formatDelayValue(delay, failedLabel)
        const title = failed
          ? `${label}
${t(delayErrorHintKey(delay.code))}`
          : tag
        return (
          <li key={tag} className="flex items-center justify-between gap-2">
            <span className="min-w-0 truncate" title={tag}>{tag}</span>
            {failed ? (
              <button
                type="button"
                className="max-w-[12rem] shrink-0 truncate text-left tabular-nums text-destructive underline-offset-2 hover:underline"
                title={title}
                aria-label={`${t("dashboard.copyProxyDelayError")}: ${tag} ${label}`}
                onClick={() => {
                  const payload = delayFailureClipboardText(tag, delay)
                  if (!payload) return
                  void copyText(payload).then(
                    () => toast.success(t("dashboard.proxyDelayErrorCopied")),
                    () => toast.error(t("dashboard.proxyDelayErrorCopyFailed")),
                  )
                }}
              >
                {label}
              </button>
            ) : (
              <span className="shrink-0 tabular-nums">{label}</span>
            )}
          </li>
        )
      })}
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
    return (
      <ProxyStatusCard
        title={t("dashboard.proxySelector")}
        error={state.query.error}
        onRetry={() => { void state.query.refetch() }}
      />
    )
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
          <DelayBadge value={currentDelay} />
        </CardTitle>
        <CardDescription>{t("dashboard.proxySelectorDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        {state.group.type === "selector" ? (
          <MemberSelect
            group={state.group}
            members={state.members}
            delays={state.delays}
            onSelect={state.select}
            selecting={state.selecting}
          />
        ) : (
          <div className="flex items-center justify-between gap-2 rounded-md border px-3 py-2 text-sm">
            <span className="text-muted-foreground">{t("dashboard.proxySelectorCurrent")}</span>
            <span className="min-w-0 truncate font-medium" title={state.group.now}>{state.group.now}</span>
          </div>
        )}
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
