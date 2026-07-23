import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { buttonVariants } from "@/components/ui/button"
import {
  hasHealthOpsAlerts,
  type HealthOpsSignals,
} from "@/features/dashboard/health-ops-signals"
import { buildNodesHref } from "@/features/nodes/nodes-filter"
import { buildSubscriptionsHref } from "@/features/subscriptions/subscription-list"
import { cn } from "@/lib/utils"

export function HealthOpsAlertChips({ signals }: { signals: HealthOpsSignals }) {
  const { t } = useTranslation()
  if (!hasHealthOpsAlerts(signals)) return null
  return (
    <div className="flex flex-wrap gap-2" data-slot="health-ops-alerts">
      {signals.failedSubscriptions > 0 ? (
        <Link
          to={buildSubscriptionsHref({ status: "error" })}
          className={cn(
            buttonVariants({ variant: "outline", size: "sm" }),
            "h-7 gap-1.5 border-destructive/40 text-destructive",
          )}
        >
          {t("dashboard.healthFailedSubs", { count: signals.failedSubscriptions })}
        </Link>
      ) : null}
      {signals.unstableNodes > 0 ? (
        <Link
          to={buildNodesHref({ stability: "unstable" })}
          className={cn(
            buttonVariants({ variant: "outline", size: "sm" }),
            "h-7 gap-1.5 border-orange-500/40 text-orange-700 dark:text-orange-400",
          )}
        >
          {t("dashboard.healthUnstableNodes", { count: signals.unstableNodes })}
        </Link>
      ) : null}
      {signals.failedNodes > 0 ? (
        <Link
          to={buildNodesHref({ stability: "failed" })}
          className={cn(
            buttonVariants({ variant: "outline", size: "sm" }),
            "h-7 gap-1.5 border-destructive/40 text-destructive",
          )}
        >
          {t("dashboard.healthFailedNodes", { count: signals.failedNodes })}
        </Link>
      ) : null}
    </div>
  )
}

export function HealthOpsAlertActions({ signals }: { signals: HealthOpsSignals }) {
  const { t } = useTranslation()
  const failedSubsHref = buildSubscriptionsHref({ status: "error" })
  const unstableHref = buildNodesHref({ stability: "unstable" })
  const failedNodesHref = buildNodesHref({ stability: "failed" })
  return (
    <>
      {signals.failedSubscriptions > 0 ? (
        <Link to={failedSubsHref} className={cn(buttonVariants({ variant: "outline", size: "sm" }))}>
          {t("dashboard.openFailedSubscriptions")}
        </Link>
      ) : null}
      {signals.problemNodes > 0 ? (
        <Link
          to={signals.failedNodes > 0 && signals.unstableNodes === 0 ? failedNodesHref : unstableHref}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          {t("dashboard.healthOpenProblemNodes")}
        </Link>
      ) : null}
    </>
  )
}
