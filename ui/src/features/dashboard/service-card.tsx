import { PauseIcon, PlayIcon, RotateCcwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import type { ServiceStatus } from "@/lib/api/types"
import { cn } from "@/lib/utils"

interface ServiceCardProps {
  status: ServiceStatus
  pending?: string
  onAction: (action: "start" | "stop" | "restart") => void
}

function ActionButton({
  action,
  pending,
  disabled,
  onAction,
}: Omit<ServiceCardProps, "status"> & {
  action: "start" | "stop" | "restart"
  disabled?: boolean
}) {
  const { t } = useTranslation()
  const labels = {
    start: t("dashboard.start"),
    stop: t("dashboard.stop"),
    restart: t("dashboard.restart"),
  }
  const icons = { start: PlayIcon, stop: PauseIcon, restart: RotateCcwIcon }
  const Icon = icons[action]
  const button = (
    <Button
      variant={action === "stop" ? "destructive" : "outline"}
      size="sm"
      className="h-8 w-full"
      disabled={Boolean(pending) || disabled}
      onClick={action === "start" ? () => onAction(action) : undefined}
    >
      {pending === action ? <Spinner aria-hidden="true" data-icon="inline-start" /> : <Icon data-icon="inline-start" />}
      {labels[action]}
    </Button>
  )
  if (action === "start") return button
  return (
    <ConfirmAction
      trigger={button}
      title={t("dashboard.confirmActionTitle")}
      description={t("dashboard.confirmActionDescription")}
      confirmLabel={t("dashboard.confirmAction")}
      confirmVariant={action === "stop" ? "destructive" : "default"}
      onConfirm={() => onAction(action)}
    />
  )
}

function formatTimestamp(value?: string) {
  if (!value) return ""
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function DiagnosticRow({ label, value, title }: { label: string; value: string; title?: string }) {
  return (
    <div className="min-w-0 rounded-md border bg-muted/30 px-2.5 py-1.5">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="truncate text-sm font-medium tabular-nums" title={title ?? value}>
        {value}
      </p>
    </div>
  )
}

export function ServiceCard({ status, pending, onAction }: ServiceCardProps) {
  const { t } = useTranslation()
  const startedAt = formatTimestamp(status.started_at)
  const lastErrorAt = formatTimestamp(status.last_error_at)
  const configPath = status.config_path?.trim() || "—"
  const lastError = status.last_error?.trim() || ""
  const errorLogsHref = buildLogsHref({ preset: "errors" })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.service")}</CardTitle>
        <CardDescription>{t("dashboard.serviceDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={status.running ? "default" : "secondary"}>
            {status.running ? t("dashboard.running") : t("dashboard.stopped")}
          </Badge>
          <span className="text-muted-foreground tabular-nums">{status.uptime || "—"}</span>
          {status.version ? (
            <Badge variant="outline" className="max-w-full truncate tabular-nums" title={status.version}>
              {status.version}
            </Badge>
          ) : null}
        </div>
        <div className="grid gap-2 sm:grid-cols-2">
          <DiagnosticRow label={t("dashboard.serviceStartedAt")} value={startedAt || "—"} />
          <DiagnosticRow label={t("dashboard.serviceConfigPath")} value={configPath} title={configPath} />
        </div>
        {lastError ? (
          <div className="rounded-lg border border-destructive/40 bg-destructive/5 px-3 py-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs font-medium text-destructive">{t("dashboard.serviceLastError")}</p>
              {lastErrorAt ? (
                <p className="text-xs text-muted-foreground tabular-nums">{lastErrorAt}</p>
              ) : null}
            </div>
            <p className="mt-1 break-words text-sm text-destructive" title={lastError}>
              {lastError}
            </p>
            <Link
              to={errorLogsHref}
              className={cn(buttonVariants({ variant: "link", size: "sm" }), "h-auto px-0")}
            >
              {t("dashboard.openErrorLogs")}
            </Link>
          </div>
        ) : null}
      </CardContent>
      <CardFooter className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <ActionButton action="start" pending={pending} disabled={status.running} onAction={onAction} />
        <ActionButton action="stop" pending={pending} disabled={!status.running} onAction={onAction} />
        <ActionButton action="restart" pending={pending} disabled={!status.running} onAction={onAction} />
      </CardFooter>
    </Card>
  )
}
