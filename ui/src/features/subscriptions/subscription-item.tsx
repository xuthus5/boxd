import { NetworkIcon, ScrollTextIcon, Trash2Icon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildLogsHref } from "@/features/observability/log-filter-presets"
import { formatRelativeTime } from "@/features/subscriptions/relative-time"
import { SubscriptionTrafficBadges } from "@/features/subscriptions/subscription-traffic"
import type { Subscription } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export interface SubscriptionItemProps {
  item: Subscription
  onEdit: () => void
  onRefresh: () => void
  onDelete: () => void
}

function urlTestStatus(item: Subscription, t: (key: string) => string) {
  const hasOverrides = Boolean(item.urltest && Object.values(item.urltest).some((value) => value !== undefined))
  if (item.urltest?.enabled === false) return t("subscriptions.urlTestOff")
  if (hasOverrides) return t("subscriptions.urlTestCustom")
  return t("subscriptions.urlTestInherited")
}

export function SubscriptionItem({ item, onEdit, onRefresh, onDelete }: SubscriptionItemProps) {
  const { t, i18n } = useTranslation()
  const [mountedAt] = useState(() => Date.now())
  return (
    <article aria-label={item.name}>
      <Card size="sm" className={item.error ? "border-destructive/40" : undefined}>
        <CardHeader className="min-w-0 gap-1.5">
          <CardTitle className="truncate" title={item.name}>{item.name}</CardTitle>
          <CardDescription className="line-clamp-2 break-all" title={item.url}>{item.url}</CardDescription>
          <CardAction>
            <Badge variant="outline">{t("subscriptions.nodeCount", { count: item.outbounds?.length ?? 0 })}</Badge>
          </CardAction>
        </CardHeader>
        <CardContent className="flex flex-col gap-1.5">
          <span className="text-xs text-muted-foreground sm:text-sm" title={item.last_updated ? new Date(item.last_updated).toLocaleString() : undefined}>
            {item.last_updated && !Number.isNaN(Date.parse(item.last_updated))
              ? t("subscriptions.updatedRelative", { time: formatRelativeTime(item.last_updated, mountedAt, i18n.language) })
              : t("subscriptions.neverUpdated")}
          </span>
          <div className="flex flex-wrap gap-1.5">
            <Badge variant={item.error ? "destructive" : "secondary"} title={item.error || undefined}>
              {item.error ? t("subscriptions.statusError") : t("common.normal")}
            </Badge>
            <Badge variant="outline">{urlTestStatus(item, t)}</Badge>
          </div>
          {item.error ? (
            <p className="line-clamp-2 text-xs text-destructive sm:text-sm" title={item.error}>{item.error}</p>
          ) : null}
          <SubscriptionTrafficBadges traffic={item.traffic} />
          <div className="flex flex-wrap gap-1.5">
            <Link
              to={buildNodesHref({ query: item.name })}
              aria-label={`${t("subscriptions.viewNodes")}: ${item.name}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <NetworkIcon data-icon="inline-start" />
              {t("subscriptions.viewNodes")}
            </Link>
            <Link
              to={buildLogsHref({ query: item.name })}
              aria-label={`${t("subscriptions.viewLogs")}: ${item.name}`}
              className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8")}
            >
              <ScrollTextIcon data-icon="inline-start" />
              {t("subscriptions.viewLogs")}
            </Link>
          </div>
        </CardContent>
        <CardFooter className="grid grid-cols-3 gap-1.5">
          <Button size="sm" className="h-8" variant="outline" onClick={onEdit}>{t("common.edit")}</Button>
          <Button size="sm" className="h-8" variant="outline" onClick={onRefresh}>
            {item.error ? t("subscriptions.retry") : t("subscriptions.refresh")}
          </Button>
          <ConfirmAction
            trigger={(
              <Button size="sm" className="h-8" variant="destructive">
                <Trash2Icon data-icon="inline-start" />
                {t("common.delete")}
              </Button>
            )}
            title={t("common.deleteTitle")}
            description={t("common.deleteDescription")}
            confirmLabel={t("common.confirmDelete")}
            confirmVariant="destructive"
            onConfirm={onDelete}
          />
        </CardFooter>
      </Card>
    </article>
  )
}
