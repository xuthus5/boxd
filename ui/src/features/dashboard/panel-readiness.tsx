import { CheckCircle2Icon, CircleAlertIcon, LoaderCircleIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { CardQueryError } from "@/features/common/card-query-error"
import { cn } from "@/lib/utils"

export interface PanelReadinessState {
  isLoading: boolean
  isFetching: boolean
  ready: boolean
  error: unknown
  onRetry: () => void
}

function ReadinessIcon({ state }: { state: PanelReadinessState }) {
  if (state.isLoading) return <LoaderCircleIcon className="size-4 animate-spin" aria-hidden="true" />
  if (state.ready) return <CheckCircle2Icon className="size-4" aria-hidden="true" />
  return <CircleAlertIcon className="size-4" aria-hidden="true" />
}

function readinessVariant(state: PanelReadinessState): "default" | "destructive" | "outline" {
  if (state.isLoading || state.isFetching) return "outline"
  return state.ready ? "default" : "destructive"
}

function readinessLabel(state: PanelReadinessState, t: (key: string) => string): string {
  if (state.isLoading) return t("dashboard.panelReadinessChecking")
  return state.ready ? t("dashboard.panelReadinessReady") : t("dashboard.panelReadinessNotReady")
}

function readinessDescription(state: PanelReadinessState, t: (key: string) => string): string {
  if (state.isLoading) return t("dashboard.panelReadinessCheckingDescription")
  return state.ready
    ? t("dashboard.panelReadinessDescription")
    : t("dashboard.panelReadinessNotReadyDescription")
}

export function PanelReadiness({ state }: { state: PanelReadinessState }) {
  const { t } = useTranslation()
  const label = readinessLabel(state, t)
  return (
    <div
      className={cn(
        "rounded-md border px-2.5 py-1.5",
        state.ready || state.isLoading ? "border-border bg-muted/30" : "border-destructive/40 bg-destructive/5",
      )}
      data-panel-readiness={state.isLoading ? "loading" : state.ready ? "ready" : "not-ready"}
      aria-busy={state.isLoading || state.isFetching}
      aria-live="polite"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-start gap-2">
          <ReadinessIcon state={state} />
          <div className="min-w-0">
            <p className="text-xs text-muted-foreground">{t("dashboard.panelReadiness")}</p>
            <p className="text-sm font-medium">{readinessDescription(state, t)}</p>
          </div>
        </div>
        <Badge variant={readinessVariant(state)} className="shrink-0">{label}</Badge>
      </div>
      {state.error ? (
        <CardQueryError
          className="mt-2"
          error={state.error}
          scope="panel-readiness"
          path="/readyz"
          fallback={t("dashboard.panelReadinessLoadFailed")}
          onRetry={state.onRetry}
        />
      ) : null}
    </div>
  )
}
