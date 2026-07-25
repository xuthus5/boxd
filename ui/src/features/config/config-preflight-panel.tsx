import { CircleAlertIcon, CircleCheckIcon, TriangleAlertIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { ConfigPreflightIssue } from "@/features/config/config-preflight"
import { configPreflightMessageKey } from "@/features/config/config-preflight-message"
import { cn } from "@/lib/utils"

const MAX_VISIBLE_ISSUES = 12

function issueBadgeVariant(severity: ConfigPreflightIssue["severity"]): "destructive" | "outline" {
  return severity === "error" ? "destructive" : "outline"
}

function issueTextValues(issue: ConfigPreflightIssue) {
  return { reference: issue.reference ?? "" }
}

export interface ConfigPreflightPanelProps {
  issues: readonly ConfigPreflightIssue[]
  onSelectPath: (path: string) => void
}

export function ConfigPreflightPanel({ issues, onSelectPath }: ConfigPreflightPanelProps) {
  const { t } = useTranslation()
  const errors = issues.filter((item) => item.severity === "error")
  const warnings = issues.length - errors.length
  if (issues.length === 0) {
    return (
      <div
        className="flex items-start gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/5 px-3 py-2 text-sm"
        data-testid="config-preflight"
      >
        <CircleCheckIcon className="mt-0.5 size-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
        <div className="min-w-0">
          <p className="font-medium">{t("advanced.preflightTitle")}</p>
          <p className="text-xs text-muted-foreground">{t("advanced.preflightClean")}</p>
        </div>
      </div>
    )
  }
  const visibleIssues = issues.slice(0, MAX_VISIBLE_ISSUES)
  const hiddenCount = issues.length - visibleIssues.length
  return (
    <Alert
      variant={errors.length > 0 ? "destructive" : "default"}
      className={cn(errors.length === 0 && "border-amber-500/40 bg-amber-500/5")}
      data-testid="config-preflight"
    >
      {errors.length > 0 ? <CircleAlertIcon aria-hidden="true" /> : <TriangleAlertIcon aria-hidden="true" />}
      <AlertTitle className="flex flex-wrap items-center gap-1.5">
        <span>{t("advanced.preflightTitle")}</span>
        <Badge variant={errors.length > 0 ? "destructive" : "outline"}>{t("advanced.preflightErrorCount", { count: errors.length })}</Badge>
        {warnings > 0 ? <Badge variant="outline">{t("advanced.preflightWarningCount", { count: warnings })}</Badge> : null}
      </AlertTitle>
      <AlertDescription className="mt-1 flex flex-col gap-2">
        <p>{t("advanced.preflightDescription")}</p>
        <ul className="flex flex-col gap-1.5">
          {visibleIssues.map((item, index) => (
            <li key={`${item.path}:${item.code}:${index}`} className="min-w-0 rounded-md border bg-background/60 px-2 py-1.5">
              <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                <Badge variant={issueBadgeVariant(item.severity)}>
                  {item.severity === "error" ? t("advanced.preflightError") : t("advanced.preflightWarning")}
                </Badge>
                <Button
                  type="button"
                  variant="link"
                  size="xs"
                  className="h-auto min-w-0 max-w-full px-0 font-mono text-xs"
                  onClick={() => onSelectPath(item.path)}
                  title={item.path}
                >
                  {item.path}
                </Button>
              </div>
              <p className="mt-1 text-xs text-muted-foreground">
                {t(configPreflightMessageKey(item.code), issueTextValues(item))}
                {item.relatedPath ? (
                  <span className="ml-1">{t("advanced.preflightRelated", { path: item.relatedPath })}</span>
                ) : null}
              </p>
            </li>
          ))}
        </ul>
        {hiddenCount > 0 ? <p className="text-xs text-muted-foreground">{t("advanced.preflightMore", { count: hiddenCount })}</p> : null}
        <p className="text-xs text-muted-foreground">{t("advanced.preflightSaveHint")}</p>
      </AlertDescription>
    </Alert>
  )
}
