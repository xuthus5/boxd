import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { CardQueryError } from "@/features/common/card-query-error"
import { useConfigQuery } from "@/features/config/config-hooks"
import { ConfigApplyEventRow } from "@/features/dashboard/config-apply-event-row"
import { type ConfigRestoreHandler, useConfigRestore } from "@/features/dashboard/use-config-restore"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent, SingBoxConfig } from "@/lib/api/types"
import { cn } from "@/lib/utils"

function TimelineLoadErrorCard({ error, onRetry }: { error: Error; onRetry: () => void }) {
  const { t } = useTranslation()
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.applyTimelineTitle")}</CardTitle>
        <CardDescription>{t("dashboard.applyTimelineLoadFailed")}</CardDescription>
      </CardHeader>
      <CardContent>
        <CardQueryError
          error={error}
          scope="apply-timeline"
          fallback={t("dashboard.applyTimelineLoadFailed")}
          onRetry={onRetry}
        />
      </CardContent>
    </Card>
  )
}

function TimelineContent({
  events,
  now,
  locale,
  currentConfig,
  currentConfigLoading,
  restoringID,
  onRestore,
}: {
  events: ConfigApplyEvent[]
  now: number
  locale: string
  currentConfig?: SingBoxConfig
  currentConfigLoading: boolean
  restoringID: string | null
  onRestore: ConfigRestoreHandler
}) {
  const { t } = useTranslation()
  if (events.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyTitle>{t("dashboard.applyTimelineEmpty")}</EmptyTitle>
          <EmptyDescription>{t("dashboard.applyTimelineEmptyDescription")}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  }
  return (
    <ul className="flex flex-col gap-1.5">
      {events.slice(0, 8).map((event) => (
        <ConfigApplyEventRow
          key={event.id || `${event.applied_at}-${event.hash}`}
          event={event}
          now={now}
          locale={locale}
          currentConfig={currentConfig}
          currentConfigLoading={currentConfigLoading}
          onRestore={onRestore}
          restoring={restoringID !== null}
        />
      ))}
    </ul>
  )
}

export function ConfigApplyTimelineCard() {
  const { t, i18n } = useTranslation()
  const { restore, restoringID } = useConfigRestore()
  const currentConfig = useConfigQuery()
  const history = useQuery({
    queryKey: ["config", "apply-history"],
    queryFn: api.config.applyHistory,
    refetchInterval: 15000,
  })
  if (history.isLoading) return <Skeleton className="h-48 w-full" />
  if (history.error) {
    return <TimelineLoadErrorCard error={history.error} onRetry={() => { void history.refetch() }} />
  }
  const events = history.data?.events ?? []
  const locale = i18n.language?.startsWith("en") ? "en-US" : "zh-CN"
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.applyTimelineTitle")}</CardTitle>
        <CardDescription>{t("dashboard.applyTimelineDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <TimelineContent
          events={events}
          now={history.dataUpdatedAt}
          locale={locale}
          currentConfig={currentConfig.data}
          currentConfigLoading={currentConfig.isFetching}
          restoringID={restoringID}
          onRestore={restore}
        />
      </CardContent>
      <CardFooter className="flex-wrap gap-2">
        <Link to="/advanced/history" className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}>
          {t("configHistory.openApplyHistory")}
        </Link>
        <Link to="/advanced/raw" className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-8")}>
          {t("dashboard.openRawConfig")}
        </Link>
      </CardFooter>
    </Card>
  )
}
