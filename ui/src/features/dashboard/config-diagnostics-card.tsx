import { AlertTriangleIcon, CheckCircle2Icon, ExternalLinkIcon, RefreshCwIcon, XCircleIcon } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { CardQueryError } from "@/features/common/card-query-error"
import {
  configDiagnosticHref,
  configDiagnosticIssueHintKey,
  configDiagnosticIssueLabelKey,
  configDiagnosticStatusKey,
  configDiagnosticIssues,
  enabledConfigFeatures,
} from "@/features/dashboard/config-diagnostics"
import { api } from "@/lib/api/endpoints"
import type { ConfigDiagnostic, ConfigDiagnostics } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function statusVariant(status: ConfigDiagnostics["status"]): "default" | "outline" | "destructive" {
  if (status === "error") return "destructive"
  if (status === "warning") return "outline"
  return "default"
}

function StatusIcon({ status }: { status: ConfigDiagnostics["status"] }) {
  if (status === "error") return <XCircleIcon className="size-4 text-destructive" aria-hidden="true" />
  if (status === "warning") return <AlertTriangleIcon className="size-4 text-amber-600 dark:text-amber-400" aria-hidden="true" />
  return <CheckCircle2Icon className="size-4 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
}

function CountGrid({ diagnostics }: { diagnostics: ConfigDiagnostics }) {
  const { t } = useTranslation()
  const items = [
    [t("configDiagnostics.counts.inbounds"), diagnostics.counts.inbounds],
    [t("configDiagnostics.counts.outbounds"), diagnostics.counts.outbounds],
    [t("configDiagnostics.counts.endpoints"), diagnostics.counts.endpoints],
    [t("configDiagnostics.counts.routeRules"), diagnostics.counts.route_rules],
    [t("configDiagnostics.counts.ruleSets"), diagnostics.counts.rule_sets],
    [t("configDiagnostics.counts.dnsServers"), diagnostics.counts.dns_servers],
    [t("configDiagnostics.counts.dnsRules"), diagnostics.counts.dns_rules],
  ] as const
  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
      {items.map(([label, value]) => (
        <div key={label} className="rounded-md border bg-muted/30 px-2.5 py-1.5">
          <p className="truncate text-xs text-muted-foreground">{label}</p>
          <p className="text-sm font-semibold tabular-nums">{value}</p>
        </div>
      ))}
    </div>
  )
}

function DiagnosticIssue({ issue }: { issue: ConfigDiagnostic }) {
  const { t } = useTranslation()
  const href = configDiagnosticHref(issue.path)
  const label = t(configDiagnosticIssueLabelKey(issue.code))
  const value = issue.value?.trim()
  return (
    <li className="rounded-md border bg-muted/20 px-2.5 py-2">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1.5">
          <Badge variant={issue.severity === "error" ? "destructive" : "outline"}>{label}</Badge>
          {value ? <code className="min-w-0 break-all text-xs text-muted-foreground">{value}</code> : null}
        </div>
        <Link
          to={href}
          className={cn(buttonVariants({ variant: "ghost", size: "xs" }), "h-6 shrink-0 px-1.5")}
        >
          <ExternalLinkIcon data-icon="inline-start" />
          {t("configDiagnostics.openEditor")}
        </Link>
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{t(configDiagnosticIssueHintKey(issue.code))}</p>
      {issue.detail ? <p className="mt-1 break-words text-xs" title={issue.detail}>{issue.detail}</p> : null}
      {issue.path ? <code className="mt-1 block break-all text-[11px] text-muted-foreground">{issue.path}</code> : null}
    </li>
  )
}

function FeatureBadges({ diagnostics }: { diagnostics: ConfigDiagnostics }) {
  const { t } = useTranslation()
  const features = enabledConfigFeatures(diagnostics.features)
  return (
    <div className="flex flex-wrap gap-1.5">
      {features.length > 0
        ? features.map((key) => <Badge key={key} variant="secondary">{t(key)}</Badge>)
        : <span className="text-xs text-muted-foreground">{t("configDiagnostics.noFeatures")}</span>}
    </div>
  )
}

function IssueSection({ issues }: { issues: ConfigDiagnostic[] }) {
  const { t } = useTranslation()
  if (issues.length === 0) return <p className="text-sm text-muted-foreground">{t("configDiagnostics.noIssues")}</p>
  return (
    <div className="grid gap-1.5">
      <p className="text-xs font-medium text-muted-foreground">{t("configDiagnostics.issuesTitle")}</p>
      <ul className="grid max-h-72 gap-1.5 overflow-y-auto pr-1">
        {issues.map((issue, index) => <DiagnosticIssue key={`${issue.code}-${issue.path}-${index}`} issue={issue} />)}
      </ul>
    </div>
  )
}

function LoadingCard() {
  const { t } = useTranslation()
  return (
    <Card size="sm" data-testid="config-diagnostics-card">
      <CardHeader className="gap-1.5">
        <CardTitle>{t("configDiagnostics.title")}</CardTitle>
        <CardDescription>{t("configDiagnostics.description")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-16 w-full" />
      </CardContent>
    </Card>
  )
}

function DiagnosticsContent({
  diagnostics,
  isFetching,
  onRefresh,
}: {
  diagnostics: ConfigDiagnostics
  isFetching: boolean
  onRefresh: () => void
}) {
  const { t } = useTranslation()
  const issues = configDiagnosticIssues(diagnostics)
  return (
    <Card size="sm" data-testid="config-diagnostics-card">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex items-center gap-2">
          <StatusIcon status={diagnostics.status} />
          <span className="truncate">{t("configDiagnostics.title")}</span>
        </CardTitle>
        <CardDescription>{t("configDiagnostics.description")}</CardDescription>
        <CardAction>
          <Button type="button" variant="ghost" size="xs" className="h-7 px-1.5" aria-label={t("configDiagnostics.refresh")} disabled={isFetching} onClick={onRefresh}>
            <RefreshCwIcon className={cn(isFetching && "animate-spin")} aria-hidden="true" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={statusVariant(diagnostics.status)}>{t(configDiagnosticStatusKey(diagnostics.status))}</Badge>
          <span className="text-xs text-muted-foreground">
            {t("configDiagnostics.issueSummary", { errors: diagnostics.summary.errors, warnings: diagnostics.summary.warnings })}
          </span>
        </div>
        <CountGrid diagnostics={diagnostics} />
        <div className="grid gap-1.5">
          <p className="text-xs font-medium text-muted-foreground">{t("configDiagnostics.featuresTitle")}</p>
          <FeatureBadges diagnostics={diagnostics} />
        </div>
        <IssueSection issues={issues} />
        <Link to="/advanced/raw" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8 w-full sm:w-fit")}>{t("configDiagnostics.openRaw")}</Link>
      </CardContent>
    </Card>
  )
}

export function ConfigDiagnosticsCard() {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ["config", "diagnostics"],
    queryFn: api.config.diagnostics,
    refetchInterval: 15000,
    staleTime: 5000,
  })
  if (query.isLoading) return <LoadingCard />
  if (query.error || !query.data) {
    return (
      <Card size="sm" data-testid="config-diagnostics-card">
        <CardHeader className="gap-1.5">
          <CardTitle>{t("configDiagnostics.title")}</CardTitle>
          <CardDescription>{t("configDiagnostics.description")}</CardDescription>
        </CardHeader>
        <CardContent><CardQueryError error={query.error} scope="config-diagnostics" path="/api/config/diagnostics" onRetry={() => { void query.refetch() }} /></CardContent>
      </Card>
    )
  }
  return <DiagnosticsContent diagnostics={query.data} isFetching={query.isFetching} onRefresh={() => { void query.refetch() }} />
}
