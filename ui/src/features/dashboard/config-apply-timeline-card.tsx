import { useQuery } from "@tanstack/react-query"
import { CopyIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import {
  configApplyErrorClipboardText,
  configApplyErrorPath,
  configApplyErrorSectionHref,
  configApplyErrorSectionOnlyHref,
  configApplySourceHref,
  configApplySourceKey,
  shortConfigHash,
} from "@/features/dashboard/config-apply-source"
import {
  kernelErrorHintKey,
  resolveKernelErrorCode,
} from "@/features/dashboard/kernel-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { formatRelativeTime } from "@/features/subscriptions/relative-time"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function formatBytes(size: number) {
  if (!Number.isFinite(size) || size < 0) return "—"
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 ** 2).toFixed(1)} MB`
}


function EventErrorBlock({
  event,
  sourceLabel,
  sourceHref,
  onCopy,
}: {
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
      <div className="flex flex-wrap items-center gap-1.5">
        {code && code !== "unknown" ? (
          <Badge variant="outline" className="font-mono text-[10px]">{code}</Badge>
        ) : null}
        {path ? (
          <Link
            to={sectionHref}
            className="max-w-full"
            aria-label={`${t("config.jumpToPath")}: ${path}`}
            title={path}
          >
            <Badge variant="outline" className="max-w-full truncate font-mono text-[10px]">
              {path}
            </Badge>
          </Link>
        ) : null}
      </div>
      <p className="line-clamp-2 text-xs text-destructive" title={event.error}>
        {event.error}
      </p>
      <p className="line-clamp-2 text-[11px] text-muted-foreground">
        {t(kernelErrorHintKey(code))}
      </p>
      <div className="flex flex-wrap gap-1.5">
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="h-7"
          onClick={onCopy}
          aria-label={`${t("dashboard.copyApplyError")}: ${sourceLabel}`}
        >
          <CopyIcon data-icon="inline-start" />
          {t("dashboard.copyApplyError")}
        </Button>
        <Link
          to={sectionHref}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-7")}
          aria-label={`${t("dashboard.openApplySource")}: ${sourceLabel}`}
        >
          {t("dashboard.openApplySource")}
        </Link>
        {path && sectionOnlyHref !== sectionHref ? (
          <Link
            to={sectionOnlyHref}
            className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7")}
            aria-label={`${t("config.openSection")}: ${path}`}
          >
            {t("config.openSection")}
          </Link>
        ) : null}
        {!path && sectionHref !== sourceHref ? (
          <Link
            to={sourceHref}
            className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7")}
          >
            {t("dashboard.openApplySourceAlt")}
          </Link>
        ) : null}
      </div>
    </div>
  )
}

function EventRow({ event, now, locale }: { event: ConfigApplyEvent; now: number; locale: string }) {
  const { t } = useTranslation()
  const rolledBack = event.status === "rolled_back"
  const relative = formatRelativeTime(event.applied_at, now, locale) || event.applied_at
  const sourceHref = configApplySourceHref(event.source)
  const sourceLabel = t(`dashboard.${configApplySourceKey(event.source)}`)

  const copyError = () => {
    const payload = configApplyErrorClipboardText(event)
    if (!payload) return
    void copyText(payload).then(
      () => toast.success(t("dashboard.applyErrorCopied")),
      () => toast.error(t("dashboard.applyErrorCopyFailed")),
    )
  }

  const copyHash = () => {
    const hash = event.hash?.trim()
    if (!hash) return
    void copyText(hash).then(
      () => toast.success(t("dashboard.applyHashCopied")),
      () => toast.error(t("dashboard.applyHashCopyFailed")),
    )
  }

  return (
    <li
      className={cn(
        "min-w-0 rounded-md border px-2.5 py-1.5",
        rolledBack ? "border-destructive/40 bg-destructive/5" : "bg-muted/30",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            <Link
              to={sourceHref}
              className="underline-offset-4 hover:underline"
              aria-label={sourceLabel}
            >
              {sourceLabel}
            </Link>
          </p>
          <p className="truncate text-xs text-muted-foreground" title={event.applied_at}>
            {relative}
            {" · "}
            <button
              type="button"
              className="font-mono underline-offset-2 hover:underline"
              onClick={copyHash}
              aria-label={t("dashboard.copyApplyHash")}
              title={event.hash}
            >
              {shortConfigHash(event.hash)}
            </button>
            {" · "}
            {formatBytes(event.size)}
          </p>
        </div>
        <Badge variant={rolledBack ? "destructive" : "secondary"} className="shrink-0">
          {rolledBack ? t("dashboard.applyStatusRolledBack") : t("dashboard.applyStatusApplied")}
        </Badge>
      </div>
      {event.error ? (
        <EventErrorBlock event={event} sourceLabel={sourceLabel} sourceHref={sourceHref} onCopy={copyError} />
      ) : null}
    </li>
  )
}

export function ConfigApplyTimelineCard() {
  const { t, i18n } = useTranslation()
  const query = useQuery({
    queryKey: ["config", "apply-history"],
    queryFn: api.config.applyHistory,
    refetchInterval: 15000,
  })
  if (query.isLoading) return <Skeleton className="h-48 w-full" />
  if (query.error) {
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("dashboard.applyTimelineTitle")}</CardTitle>
          <CardDescription>{t("dashboard.applyTimelineLoadFailed")}</CardDescription>
        </CardHeader>
      </Card>
    )
  }
  const events = query.data?.events ?? []
  const now = Date.now()
  const locale = i18n.language?.startsWith("en") ? "en-US" : "zh-CN"
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.applyTimelineTitle")}</CardTitle>
        <CardDescription>{t("dashboard.applyTimelineDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        {events.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("dashboard.applyTimelineEmpty")}</EmptyTitle>
              <EmptyDescription>{t("dashboard.applyTimelineEmptyDescription")}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {events.slice(0, 8).map((event) => (
              <EventRow key={event.id || `${event.applied_at}-${event.hash}`} event={event} now={now} locale={locale} />
            ))}
          </ul>
        )}
      </CardContent>
      <CardFooter>
        <Link
          to="/advanced/raw"
          className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
        >
          {t("dashboard.openRawConfig")}
        </Link>
      </CardFooter>
    </Card>
  )
}
