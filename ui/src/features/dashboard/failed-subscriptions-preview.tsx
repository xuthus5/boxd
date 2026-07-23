import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import {
  resolveSubscriptionErrorCode,
  subscriptionErrorHintKey,
} from "@/features/subscriptions/subscription-error"
import { buildSubscriptionsHref } from "@/features/subscriptions/subscription-list"
import type { Subscription } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export function FailedSubscriptionsPreview({
  items,
  total,
}: {
  items: readonly Subscription[]
  total: number
}) {
  const { t } = useTranslation()
  if (total <= 0 || items.length === 0) return null
  const remaining = Math.max(0, total - items.length)
  return (
    <div className="flex w-full flex-col gap-2" data-slot="failed-subscriptions-preview">
      <ul className="flex flex-col gap-1.5">
        {items.map((item) => {
          const code = resolveSubscriptionErrorCode(item)
          const href = buildSubscriptionsHref({ status: "error", query: item.name })
          return (
            <li
              key={item.id}
              className="flex min-w-0 flex-col gap-1 rounded-md border border-destructive/30 bg-destructive/5 px-2.5 py-1.5 sm:flex-row sm:items-center sm:justify-between"
            >
              <div className="min-w-0">
                <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                  <Link
                    to={href}
                    className="truncate text-sm font-medium underline-offset-4 hover:underline"
                    title={item.name}
                  >
                    {item.name}
                  </Link>
                  {code ? (
                    <Badge variant="outline" className="font-mono text-[10px]">{code}</Badge>
                  ) : null}
                </div>
                <p className="line-clamp-1 text-xs text-destructive" title={item.error}>
                  {item.error}
                </p>
                <p className="line-clamp-1 text-[11px] text-muted-foreground">
                  {t(subscriptionErrorHintKey(code))}
                </p>
              </div>
              <Link
                to={href}
                className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8 shrink-0")}
                aria-label={`${t("dashboard.openFailedSubscription")}: ${item.name}`}
              >
                {t("dashboard.openFailedSubscription")}
              </Link>
            </li>
          )
        })}
      </ul>
      {remaining > 0 ? (
        <p className="text-xs text-muted-foreground">
          {t("dashboard.failedSubscriptionsMore", { count: remaining })}
        </p>
      ) : null}
    </div>
  )
}
