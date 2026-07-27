import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import {
  configApplyErrorClipboardText,
  configApplyErrorPath,
  configApplyErrorSectionHref,
  configApplySourceKey,
  configApplyStatusLabelKey,
} from "@/features/dashboard/config-apply-source"
import { resolveKernelErrorCode } from "@/features/dashboard/kernel-error"
import { copyText } from "@/lib/clipboard"
import type { ConfigApplyEvent } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export function HealthApplyFailurePreview({
  event,
  count,
}: {
  event?: ConfigApplyEvent
  count: number
}) {
  const { t } = useTranslation()
  if (!event || count <= 0) return null
  const code = resolveKernelErrorCode(event)
  const path = configApplyErrorPath(event)
  const href = configApplyErrorSectionHref(event)
  const sourceLabel = t(`dashboard.${configApplySourceKey(event.source)}`)
  const statusLabel = t(`dashboard.${configApplyStatusLabelKey(event.status)}`)
  const payload = configApplyErrorClipboardText(event)

  return (
    <div
      className="rounded-md border border-destructive/40 bg-destructive/5 px-2.5 py-1.5"
      data-slot="apply-failure-preview"
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex flex-col gap-1">
          <p className="text-xs font-medium text-destructive">
            {t("dashboard.healthApplyFailures", { count })}
          </p>
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge variant="outline" className="font-mono text-[10px]">{statusLabel}</Badge>
            {code && code !== "unknown" ? (
              <Badge variant="outline" className="font-mono text-[10px]">{code}</Badge>
            ) : null}
            <Badge variant="secondary" className="max-w-full truncate text-[10px]" title={sourceLabel}>
              {sourceLabel}
            </Badge>
            {path ? (
              <Link to={href} title={path} aria-label={`${t("config.jumpToPath")}: ${path}`}>
                <Badge variant="outline" className="max-w-full truncate font-mono text-[10px]">
                  {path}
                </Badge>
              </Link>
            ) : null}
          </div>
          {event.error ? (
            <p className="line-clamp-2 text-sm text-destructive" title={event.error}>
              {event.error}
            </p>
          ) : null}
        </div>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-6 shrink-0 px-1.5 text-destructive"
          aria-label={t("dashboard.copyApplyError")}
          onClick={() => {
            if (!payload) return
            void copyText(payload).then(
              () => toast.success(t("dashboard.applyErrorCopied")),
              () => toast.error(t("dashboard.applyErrorCopyFailed")),
            )
          }}
        >
          <CopyIcon className="size-3.5" aria-hidden="true" />
        </Button>
      </div>
      <div className="mt-1.5 flex flex-wrap gap-1.5">
        <Link
          to={href}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7")}
        >
          {t("dashboard.healthOpenApplyFailure")}
        </Link>
        <Link
          to="/"
          className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7")}
        >
          {t("config.openApplyTimeline")}
        </Link>
      </div>
    </div>
  )
}
