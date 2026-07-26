import { AlertTriangleIcon, CheckCircle2Icon, CircleHelpIcon, ExternalLinkIcon, RefreshCwIcon, XCircleIcon } from "lucide-react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { CardQueryError } from "@/features/common/card-query-error"
import { HealthStreamErrorBlock } from "@/features/dashboard/health-stream-error-block"
import { RuleSetHealthFooter, type RuleSetUpdateAction } from "@/features/dashboard/rule-set-health-footer"
import {
  buildRuleSetHealth,
  ruleSetHealthHref,
  type AutoUpdateFailure,
  type AutoUpdateRun,
  type RuleSetHealthItem,
  type RuleSetHealthTone,
} from "@/features/dashboard/rule-set-health"
import {
  classifyRuleSetRequestError,
  formatRuleSetRequestErrorToast,
  formatRuleSetUpdateMessage,
  ruleSetBatchFailureClipboardText,
  ruleSetErrorHintKey,
  ruleSetUpdateToastTone,
  summarizeRuleSetUpdate,
} from "@/features/policy/ruleset-update-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { api } from "@/lib/api/endpoints"
import type { LogEvent, RuleSetAutoUpdate } from "@/lib/api/types"
import { cn } from "@/lib/utils"

const defaultAutoUpdate: RuleSetAutoUpdate = { enabled: false, interval: "24h" }

function statusVariant(tone: RuleSetHealthTone): "default" | "outline" | "destructive" {
  if (tone === "error") return "destructive"
  if (tone === "warning") return "outline"
  return "default"
}

function StatusIcon({ tone }: { tone: RuleSetHealthTone }) {
  if (tone === "error") return <XCircleIcon className="size-4 text-destructive" aria-hidden="true" />
  if (tone === "warning") return <AlertTriangleIcon className="size-4 text-muted-foreground" aria-hidden="true" />
  if (tone === "empty") return <CircleHelpIcon className="size-4 text-muted-foreground" aria-hidden="true" />
  return <CheckCircle2Icon className="size-4 text-primary" aria-hidden="true" />
}

function formatTime(value?: string) {
  if (!value) return "—"
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function MetricGrid({ total, updatable, available, missing }: { total: number; updatable: number; available: number; missing: number }) {
  const { t } = useTranslation()
  const metrics = [
    [t("ruleSetHealth.totalCount", { count: total }), t("ruleSetHealth.total")],
    [t("ruleSetHealth.updatableCount", { count: updatable }), t("ruleSetHealth.updatable")],
    [t("ruleSetHealth.availableCount", { count: available }), t("ruleSetHealth.available")],
    [t("ruleSetHealth.missingCount", { count: missing }), t("ruleSetHealth.missing")],
  ] as const
  return <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
    {metrics.map(([value, label]) => (
      <div key={label} className="rounded-md border bg-muted/30 px-2.5 py-1.5">
        <p className="text-sm font-semibold tabular-nums">{value}</p>
        <p className="truncate text-xs text-muted-foreground">{label}</p>
      </div>
    ))}
  </div>
}

function AutoUpdateFailureList({ failures }: { failures?: AutoUpdateFailure[] }) {
  const { t } = useTranslation()
  if (!failures?.length) return null
  return <ul aria-label={t("ruleSetHealth.autoUpdateFailed", { count: failures.length })} className="grid gap-1 rounded-md border border-destructive/30 bg-background/50 p-2">
    {failures.map((failure) => (
      <li key={`${failure.tag}-${failure.code ?? "unknown"}`} className="flex min-w-0 flex-wrap items-center gap-1.5 text-xs">
        <span className="min-w-0 break-all font-medium">{failure.tag}</span>
        <Badge variant="outline">{failure.code ?? "unknown"}</Badge>
        <span className="min-w-0 text-muted-foreground">{t(ruleSetErrorHintKey(failure.code))}</span>
      </li>
    ))}
  </ul>
}

function AutoUpdatePanel({ enabled, interval, latest, latestArtifactUpdatedAt }: { enabled: boolean; interval: string; latest?: AutoUpdateRun; latestArtifactUpdatedAt?: string }) {
  const { t } = useTranslation()
  return (
    <Alert variant={latest?.failed ? "destructive" : "default"}>
      <AlertTitle>{t("ruleSetHealth.latestRun")}</AlertTitle>
      <AlertDescription className="flex flex-col gap-1">
        {latest ? (
          <>
            <span>{t("ruleSetHealth.latestRunSummary", {
              updated: latest.updated,
              failed: latest.failed,
              skipped: latest.skipped,
            })}</span>
            {latest.failed > 0 ? <span>{t("ruleSetHealth.autoUpdateFailed", { count: latest.failed })}</span> : null}
            <AutoUpdateFailureList failures={latest.failures} />
            {latest.error ? <span>{t("ruleSetHealth.autoUpdateRequestFailed")}</span> : null}
            {latest.timestamp ? <time dateTime={latest.timestamp}>{formatTime(latest.timestamp)}</time> : null}
          </>
        ) : <span>{t("ruleSetHealth.noRun")}</span>}
        <span className="text-xs text-muted-foreground">
          {t("ruleSetHealth.autoUpdate")}: {enabled ? t("ruleSetHealth.enabled") : t("ruleSetHealth.disabled")} · {t("ruleSetHealth.interval", { interval })}
        </span>
        {latestArtifactUpdatedAt ? <span className="text-xs text-muted-foreground">{t("ruleSetHealth.latestArtifact", { time: formatTime(latestArtifactUpdatedAt) })}</span> : null}
      </AlertDescription>
    </Alert>
  )
}

function IssueList({ items }: { items: RuleSetHealthItem[] }) {
  const { t } = useTranslation()
  if (items.length === 0) return <p className="text-sm text-muted-foreground">{t("ruleSetHealth.noIssues")}</p>
  return <ul className="grid gap-1.5">
    {items.map((entry) => (
      <li key={`${entry.tag}-${entry.index}`} className="flex flex-wrap items-center justify-between gap-2 rounded-md border bg-muted/20 px-2.5 py-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <Badge variant={entry.state === "missing" ? "destructive" : "outline"}>{t(`ruleSetHealth.${entry.state}`)}</Badge>
          <span className="min-w-0 break-all text-sm">{entry.tag}</span>
        </div>
        <Link to={ruleSetHealthHref(entry.index)} className={cn(buttonVariants({ variant: "ghost", size: "xs" }), "h-6 shrink-0 px-1.5")}>
          <ExternalLinkIcon data-icon="inline-start" />
          {t("ruleSetHealth.openItem", { tag: entry.tag })}
        </Link>
      </li>
    ))}
  </ul>
}

function LoadingCard() {
  const { t } = useTranslation()
  return <Card size="sm" data-testid="rule-set-health-card">
    <CardHeader className="gap-1.5">
      <CardTitle>{t("ruleSetHealth.title")}</CardTitle>
      <CardDescription>{t("ruleSetHealth.description")}</CardDescription>
    </CardHeader>
    <CardContent className="flex flex-col gap-2">
      <Skeleton className="h-6 w-36" />
      <Skeleton className="h-16 w-full" />
    </CardContent>
  </Card>
}

interface RuleSetHealthCardProps {
  appLogs?: LogEvent[]
  appLogError?: string
  appLogStatus?: string
  onReconnectAppLogs?: () => void
}

type Translate = (key: string, values?: Record<string, string | number>) => string

function useRuleSetHealthData(appLogs: LogEvent[]) {
  const statusQuery = useQuery({
    queryKey: ["rule-sets", "status"],
    queryFn: api.config.ruleSetsStatus,
    refetchInterval: 15000,
    staleTime: 5000,
  })
  const autoUpdateQuery = useQuery({
    queryKey: ["rule-sets", "auto-update"],
    queryFn: api.config.ruleSetsAutoUpdate,
    refetchInterval: 15000,
    staleTime: 5000,
  })
  const refresh = () => {
    void Promise.allSettled([statusQuery.refetch(), autoUpdateQuery.refetch()])
  }
  return {
    loading: statusQuery.isLoading || autoUpdateQuery.isLoading,
    error: statusQuery.error || autoUpdateQuery.error,
    errorPath: statusQuery.error ? "/api/config/rule-sets/status" : "/api/config/rule-sets/auto-update",
    refreshing: statusQuery.isFetching || autoUpdateQuery.isFetching,
    autoUpdate: autoUpdateQuery.data ?? defaultAutoUpdate,
    health: buildRuleSetHealth(statusQuery.data ?? [], autoUpdateQuery.data ?? defaultAutoUpdate, appLogs),
    refresh,
  }
}

function showRuleSetRequestError(error: Error, t: Translate) {
  const code = classifyRuleSetRequestError(error)
  toast.error(formatRuleSetRequestErrorToast(error, t, t("policy.route.ruleSetUpdateFailed", { failed: 1 })), {
    description: t(ruleSetErrorHintKey(code)),
  })
}

function showRuleSetUpdateToast(envelope: Awaited<ReturnType<typeof api.config.updateRuleSets>>, t: Translate) {
  const summary = summarizeRuleSetUpdate(envelope.data)
  const message = formatRuleSetUpdateMessage(summary, t)
  const payload = ruleSetBatchFailureClipboardText(summary)
  const options = payload ? {
    description: summary.failedSamples[0] ? t(ruleSetErrorHintKey(summary.failedSamples[0].code)) : undefined,
    action: {
      label: t("policy.route.copyRuleSetError"),
      onClick: () => {
        void copyText(payload).then(
          () => toast.success(t("policy.route.ruleSetErrorCopied")),
          () => toast.error(t("policy.route.ruleSetErrorCopyFailed")),
        )
      },
    },
  } : undefined
  const tone = ruleSetUpdateToastTone(summary)
  if (tone === "error") toast.error(message, options)
  else if (tone === "warning") toast.warning(message, options)
  else toast.success(message)
}

function useRuleSetUpdateMutation(t: Translate): RuleSetUpdateAction {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => api.config.updateRuleSets(),
    onSuccess: (envelope) => showRuleSetUpdateToast(envelope, t),
    onError: (error: Error) => showRuleSetRequestError(error, t),
    onSettled: () => Promise.all([
      queryClient.invalidateQueries({ queryKey: ["rule-sets", "status"] }),
      queryClient.invalidateQueries({ queryKey: ["rule-sets", "auto-update"] }),
    ]),
  })
  return { isPending: mutation.isPending, mutate: () => mutation.mutate() }
}

function QueryErrorCard({ error, path, onRetry }: { error: Error; path: string; onRetry: () => void }) {
  const { t } = useTranslation()
  return <Card size="sm" data-testid="rule-set-health-card">
    <CardHeader className="gap-1.5">
      <CardTitle>{t("ruleSetHealth.title")}</CardTitle>
      <CardDescription>{t("ruleSetHealth.description")}</CardDescription>
    </CardHeader>
    <CardContent><CardQueryError error={error} scope="rule-set-health" path={path} onRetry={onRetry} /></CardContent>
  </Card>
}

function RuleSetHealthContent({
  health,
  autoUpdate,
  refreshing,
  refresh,
  updateAction,
  appLogError,
  appLogStatus,
  onReconnectAppLogs,
}: {
  health: ReturnType<typeof buildRuleSetHealth>
  autoUpdate: RuleSetAutoUpdate
  refreshing: boolean
  refresh: () => void
  updateAction: RuleSetUpdateAction
  appLogError?: string
  appLogStatus?: string
  onReconnectAppLogs?: () => void
}) {
  const { t } = useTranslation()
  const issues = health.items.filter((entry) => entry.state === "missing" || entry.state === "stale")
  return <Card size="sm" data-testid="rule-set-health-card">
    <CardHeader className="gap-1.5">
      <CardTitle className="flex items-center gap-2">
        <StatusIcon tone={health.tone} />
        <span className="truncate">{t("ruleSetHealth.title")}</span>
      </CardTitle>
      <CardDescription>{t("ruleSetHealth.description")}</CardDescription>
      <CardAction>
        <Button type="button" variant="ghost" size="xs" className="h-7 px-1.5" aria-label={t("ruleSetHealth.refresh")} disabled={refreshing} onClick={refresh}>
          <RefreshCwIcon className={cn(refreshing && "animate-spin")} aria-hidden="true" />
        </Button>
      </CardAction>
    </CardHeader>
    <CardContent className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={statusVariant(health.tone)}>{t(`ruleSetHealth.${health.tone}`)}</Badge>
        <span className="text-xs text-muted-foreground">{t("ruleSetHealth.summary", { total: health.total, updatable: health.updatable })}</span>
      </div>
      <MetricGrid total={health.total} updatable={health.updatable} available={health.available} missing={health.missing} />
      <AutoUpdatePanel enabled={autoUpdate.enabled} interval={autoUpdate.interval} latest={health.latestAutoUpdate} latestArtifactUpdatedAt={health.latestUpdatedAt} />
      <IssueList items={issues} />
      {appLogError || appLogStatus === "reconnecting" ? <HealthStreamErrorBlock error={appLogError} status={appLogStatus} path={api.stats.paths.appLogs} href="/observability/logs?tab=application" actionLabel={t("ruleSetHealth.openLogs")} onReconnect={onReconnectAppLogs} /> : null}
    </CardContent>
    <RuleSetHealthFooter updatable={health.updatable} updateAction={updateAction} />
  </Card>
}

export function RuleSetHealthCard({
  appLogs = [],
  appLogError,
  appLogStatus,
  onReconnectAppLogs,
}: RuleSetHealthCardProps) {
  const { t } = useTranslation()
  const data = useRuleSetHealthData(appLogs)
  const updateAction = useRuleSetUpdateMutation(t)
  if (data.loading) return <LoadingCard />
  if (data.error) return <QueryErrorCard error={data.error} path={data.errorPath} onRetry={data.refresh} />
  return <RuleSetHealthContent
    health={data.health}
    autoUpdate={data.autoUpdate}
    refreshing={data.refreshing}
    refresh={data.refresh}
    updateAction={updateAction}
    appLogError={appLogError}
    appLogStatus={appLogStatus}
    onReconnectAppLogs={onReconnectAppLogs}
  />
}
