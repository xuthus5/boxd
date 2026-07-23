import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import {
  configApplySourceKey,
  shortConfigHash,
} from "@/features/dashboard/config-apply-source"
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

function EventRow({ event, now, locale }: { event: ConfigApplyEvent; now: number; locale: string }) {
  const { t } = useTranslation()
  const rolledBack = event.status === "rolled_back"
  const relative = formatRelativeTime(event.applied_at, now, locale) || event.applied_at
  return (
    <li className="min-w-0 rounded-md border bg-muted/30 px-2.5 py-1.5">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {t(`dashboard.${configApplySourceKey(event.source)}`)}
          </p>
          <p className="truncate text-xs text-muted-foreground" title={event.applied_at}>
            {relative}
            {" · "}
            {shortConfigHash(event.hash)}
            {" · "}
            {formatBytes(event.size)}
          </p>
        </div>
        <Badge variant={rolledBack ? "destructive" : "secondary"} className="shrink-0">
          {rolledBack ? t("dashboard.applyStatusRolledBack") : t("dashboard.applyStatusApplied")}
        </Badge>
      </div>
      {event.error ? (
        <p className="mt-1 line-clamp-2 text-xs text-destructive" title={event.error}>
          {event.error}
        </p>
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
