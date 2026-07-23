import { useQuery } from "@tanstack/react-query"
import { CheckCircle2Icon, CircleIcon } from "lucide-react"
import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useConfigQuery } from "@/features/config/config-hooks"
import { buildSetupSteps, setupProgress, type SetupStepId } from "@/features/dashboard/setup-checklist"
import {
  buildSubscriptionsHref,
  failedSubscriptionIds,
} from "@/features/subscriptions/subscription-list"
import { api } from "@/lib/api/endpoints"
import type { ServiceStatus, Subscription } from "@/lib/api/types"
import { cn } from "@/lib/utils"

const labels: Record<SetupStepId, string> = {
  kernel: "setupKernel",
  inbounds: "setupInbounds",
  outbounds: "setupOutbounds",
  subscriptions: "setupSubscriptions",
  route: "setupRoute",
  clashApi: "setupClashApi",
}

export function SetupChecklistCard({ status }: { status?: ServiceStatus }) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const items = useMemo(
    () => (Array.isArray(subscriptions.data) ? subscriptions.data as Subscription[] : []),
    [subscriptions.data],
  )
  const failedCount = useMemo(() => failedSubscriptionIds(items).length, [items])
  const failedHref = buildSubscriptionsHref({ status: "error" })

  const steps = useMemo(
    () => buildSetupSteps({
      status,
      config: config.data,
      subscriptions: Array.isArray(subscriptions.data) ? subscriptions.data : undefined,
    }),
    [status, config.data, subscriptions.data],
  )
  const progress = setupProgress(steps)
  const showChecklist = !progress.complete
  const showFailed = failedCount > 0

  if (config.isLoading || subscriptions.isLoading) return <Skeleton className="h-48 w-full" />
  if (!showChecklist && !showFailed) return null

  return (
    <Card className={showChecklist || showFailed ? "lg:col-span-3" : undefined}>
      <CardHeader className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <CardTitle>{t("dashboard.setupTitle")}</CardTitle>
          <CardDescription>
            {progress.complete ? t("dashboard.setupComplete") : t("dashboard.setupDescription")}
          </CardDescription>
        </div>
        <Badge variant="secondary">{t("dashboard.setupProgress", { done: progress.done, total: progress.total })}</Badge>
      </CardHeader>
      {showChecklist ? (
        <CardContent>
          <ul className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
            {steps.map((step) => (
              <li key={step.id} className="flex items-center justify-between gap-2 rounded-lg border px-3 py-2">
                <div className="flex min-w-0 items-center gap-2">
                  {step.done
                    ? <CheckCircle2Icon className="size-4 shrink-0 text-primary" aria-hidden />
                    : <CircleIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />}
                  <span className="truncate text-sm">{t(`dashboard.${labels[step.id]}`)}</span>
                </div>
                {step.done
                  ? <Badge variant="outline">{t("dashboard.setupDone")}</Badge>
                  : (
                    <Link
                      to={step.href}
                      className={cn(buttonVariants({ variant: "outline", size: "xs" }), "shrink-0")}
                    >
                      {t("dashboard.setupAction")}
                    </Link>
                  )}
              </li>
            ))}
          </ul>
        </CardContent>
      ) : null}
      {showFailed ? (
        <CardFooter className="flex flex-col items-start gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-destructive">
            {t("dashboard.failedSubscriptions", { count: failedCount })}
          </p>
          <Link
            to={failedHref}
            className={cn(buttonVariants({ variant: "outline", size: "sm" }), "shrink-0")}
          >
            {t("dashboard.openFailedSubscriptions")}
          </Link>
        </CardFooter>
      ) : null}
    </Card>
  )
}
