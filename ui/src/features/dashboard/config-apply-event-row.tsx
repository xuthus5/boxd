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
  configApplyErrorSectionOnlyHref,
  configApplyEventFailed,
  configApplySourceHref,
  configApplySourceKey,
  configApplyStatusLabelKey,
  configApplyStatusVariant,
  shortConfigHash,
} from "@/features/dashboard/config-apply-source"
import { ConfigRestoreAction } from "@/features/dashboard/config-restore-action"
import type { ConfigRestoreHandler } from "@/features/dashboard/use-config-restore"
import { kernelErrorHintKey, resolveKernelErrorCode } from "@/features/dashboard/kernel-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { formatRelativeTime } from "@/features/subscriptions/relative-time"
import { cn } from "@/lib/utils"
import type { ConfigApplyEvent, SingBoxConfig } from "@/lib/api/types"

interface ConfigApplyEventRowProps {
  event: ConfigApplyEvent
  now: number
  locale: string
  currentConfig?: SingBoxConfig
  currentConfigLoading: boolean
  restoring: boolean
  onRestore: ConfigRestoreHandler
}

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size < 0) return "—"
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 ** 2).toFixed(1)} MB`
}

function EventErrorBadges({ code, path, href }: { code?: string; path?: string; href: string }) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {code && code !== "unknown" ? <Badge variant="outline" className="font-mono text-[10px]">{code}</Badge> : null}
      {path ? (
        <Link to={href} className="max-w-full" aria-label={`${t("config.jumpToPath")}: ${path}`} title={path}>
          <Badge variant="outline" className="max-w-full truncate font-mono text-[10px]">{path}</Badge>
        </Link>
      ) : null}
    </div>
  )
}

function EventErrorActions({
  path,
  sourceHref,
  sectionHref,
  sectionOnlyHref,
  sourceLabel,
  onCopy,
}: {
  path?: string
  sourceHref: string
  sectionHref: string
  sectionOnlyHref: string
  sourceLabel: string
  onCopy: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex flex-wrap gap-1.5">
      <Button type="button" size="sm" variant="outline" className="h-7" onClick={onCopy} aria-label={`${t("dashboard.copyApplyError")}: ${sourceLabel}`}>
        <CopyIcon data-icon="inline-start" />
        {t("dashboard.copyApplyError")}
      </Button>
      <Link to={sectionHref} className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7")} aria-label={`${t("dashboard.openApplySource")}: ${sourceLabel}`}>
        {t("dashboard.openApplySource")}
      </Link>
      {path && sectionOnlyHref !== sectionHref ? (
        <Link to={sectionOnlyHref} className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7")} aria-label={`${t("config.openSection")}: ${path}`}>
          {t("config.openSection")}
        </Link>
      ) : null}
      {!path && sectionHref !== sourceHref ? (
        <Link to={sourceHref} className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7")}>
          {t("dashboard.openApplySourceAlt")}
        </Link>
      ) : null}
    </div>
  )
}

function EventErrorBlock({ event, sourceLabel, sourceHref, onCopy }: {
  event: ConfigApplyEvent
  sourceLabel: string
  sourceHref: string
  onCopy: () => void
}) {
  const { t } = useTranslation()
  const code = resolveKernelErrorCode(event)
  const path = configApplyErrorPath(event)
  const sectionHref = configApplyErrorSectionHref(event)
  const sectionOnlyHref = configApplyErrorSectionOnlyHref(event)
  return (
    <div className="mt-1.5 flex flex-col gap-1.5">
      <EventErrorBadges code={code} path={path} href={sectionHref} />
      <p className="line-clamp-2 text-xs text-destructive" title={event.error}>{event.error}</p>
      <p className="line-clamp-2 text-[11px] text-muted-foreground">{t(kernelErrorHintKey(code))}</p>
      <EventErrorActions path={path} sourceHref={sourceHref} sectionHref={sectionHref} sectionOnlyHref={sectionOnlyHref} sourceLabel={sourceLabel} onCopy={onCopy} />
    </div>
  )
}

function EventSummary({ event, relative, sourceHref, sourceLabel, statusLabel, onCopyHash }: {
  event: ConfigApplyEvent
  relative: string
  sourceHref: string
  sourceLabel: string
  statusLabel: string
  onCopyHash: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex items-start justify-between gap-2">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium"><Link to={sourceHref} className="underline-offset-4 hover:underline" aria-label={sourceLabel}>{sourceLabel}</Link></p>
        <p className="truncate text-xs text-muted-foreground" title={event.applied_at}>
          {relative}{" · "}
          <button type="button" className="font-mono underline-offset-2 hover:underline" onClick={onCopyHash} aria-label={t("dashboard.copyApplyHash")} title={event.hash}>
            {shortConfigHash(event.hash)}
          </button>
          {" · "}{formatBytes(event.size)}
        </p>
      </div>
      <div className="flex shrink-0 flex-wrap justify-end gap-1">
        {event.current ? <Badge variant="outline">{t("configRestore.currentConfig")}</Badge> : null}
        <Badge variant={configApplyStatusVariant(event.status)}>{statusLabel}</Badge>
      </div>
    </div>
  )
}

export function ConfigApplyEventRow({
  event,
  now,
  locale,
  currentConfig,
  currentConfigLoading,
  restoring,
  onRestore,
}: ConfigApplyEventRowProps) {
  const { t } = useTranslation()
  const failed = configApplyEventFailed(event.status)
  const sourceHref = configApplySourceHref(event.source)
  const sourceLabel = event.source === "restore"
    ? t("configRestore.source")
    : t(`dashboard.${configApplySourceKey(event.source)}`)
  const statusLabel = t(`dashboard.${configApplyStatusLabelKey(event.status)}`)
  const relative = formatRelativeTime(event.applied_at, now, locale) || event.applied_at
  const copyError = () => copyEventText(configApplyErrorClipboardText(event), t("dashboard.applyErrorCopied"), t("dashboard.applyErrorCopyFailed"))
  const copyHash = () => copyEventText(event.hash?.trim() ?? "", t("dashboard.applyHashCopied"), t("dashboard.applyHashCopyFailed"))
  return (
    <li className={cn("min-w-0 rounded-md border px-2.5 py-1.5", failed ? "border-destructive/40 bg-destructive/5" : "bg-muted/30")}>
      <EventSummary event={event} relative={relative} sourceHref={sourceHref} sourceLabel={sourceLabel} statusLabel={statusLabel} onCopyHash={copyHash} />
      {event.error ? <EventErrorBlock event={event} sourceLabel={sourceLabel} sourceHref={sourceHref} onCopy={copyError} /> : null}
      {event.restorable && event.id && !event.current && !failed ? (
        <ConfigRestoreAction
          event={event}
          currentConfig={currentConfig}
          currentConfigLoading={currentConfigLoading}
          restoring={restoring}
          onRestore={onRestore}
        />
      ) : null}
    </li>
  )
}

function copyEventText(value: string, success: string, failure: string) {
  if (!value) return
  void copyText(value).then(
    () => toast.success(success),
    () => toast.error(failure),
  )
}
