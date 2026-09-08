import { useQuery } from "@tanstack/react-query"
import { lazy, Suspense, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { Button, buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { CardQueryError } from "@/features/common/card-query-error"
import { buildSetupSteps, setupProgress } from "@/features/dashboard/setup-checklist"
import { FailedSubscriptionsPreview } from "@/features/dashboard/failed-subscriptions-preview"
import { SetupSteps } from "@/features/dashboard/setup-steps"
import { SetupAccessGuide } from "@/features/setup/setup-access-guide"
import {
  buildSubscriptionsHref,
  failedSubscriptionIds,
  listFailedSubscriptions,
} from "@/features/subscriptions/subscription-list"
import { api } from "@/lib/api/endpoints"
import { setupAPI, type SetupStatus } from "@/lib/api/setup"
import type { ServiceStatus, Subscription } from "@/lib/api/types"
import { cn } from "@/lib/utils"

const SetupDialog = lazy(() => import("@/features/setup/setup-dialog").then((module) => ({ default: module.SetupDialog })))

function useSetupChecklist(status?: ServiceStatus) {
  const setup = useQuery({ queryKey: ["config", "setup"], queryFn: setupAPI.status, retry: false })
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const [editing, setEditing] = useState(false)
  const [showAccess, setShowAccess] = useState(false)
  const items = useMemo(
    () => (Array.isArray(subscriptions.data) ? subscriptions.data as Subscription[] : []),
    [subscriptions.data],
  )

  const steps = useMemo(
    () => buildSetupSteps({
      status,
      setup: setup.data,
      subscriptions: Array.isArray(subscriptions.data) ? subscriptions.data : undefined,
    }),
    [status, setup.data, subscriptions.data],
  )
  const available = Boolean(setup.data?.capabilities && Array.isArray(setup.data.modules) && Array.isArray(setup.data.listeners))
  return { setup, subscriptions, items, steps, available, editing, setEditing, showAccess, setShowAccess }
}

export function SetupChecklistCard({ status }: { status?: ServiceStatus }) {
  const { t } = useTranslation()
  const state = useSetupChecklist(status)
  const { setup, subscriptions, steps, items, available, editing, setEditing, showAccess, setShowAccess } = state
  const progress = setupProgress(steps)
  if (setup.isLoading || subscriptions.isLoading) return <Skeleton className="h-48 w-full" />

  return (
    <Card size="sm" className="lg:col-span-3">
      <CardHeader className="flex flex-col gap-1.5 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <CardTitle>{t("setup.title")}</CardTitle>
          <CardDescription>{t("setup.description")}</CardDescription>
        </div>
        <Badge variant="secondary" className="shrink-0">{t("setup.progress", { done: progress.done, total: progress.total })}</Badge>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {setup.error ? <CardQueryError error={setup.error} scope="setup-config" onRetry={() => { void setup.refetch() }} /> : null}
        {subscriptions.error ? <CardQueryError error={subscriptions.error} scope="setup-subscriptions" onRetry={() => { void subscriptions.refetch() }} /> : null}
        <SetupSteps steps={steps} available={available} onModules={() => setEditing(true)} onAccess={() => setShowAccess((value) => !value)} />
        {showAccess && available ? <SetupAccessGuide status={setup.data!} /> : null}
        <p className="text-xs text-muted-foreground">{t("setup.verifyHint")}</p>
      </CardContent>
      <SetupFailures items={items} />
      <SetupDialogMount editing={editing && available} status={setup.data} onClose={() => setEditing(false)} />
    </Card>
  )
}

function SetupFailures({ items }: { items: Subscription[] }) {
  const { t } = useTranslation()
  const failedCount = failedSubscriptionIds(items).length
  if (!failedCount) return null
  const failedPreview = listFailedSubscriptions(items, 3)
  const failedHref = buildSubscriptionsHref({ status: "error" })
  return <CardFooter className="flex flex-col items-stretch gap-2">
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-destructive">{t("dashboard.failedSubscriptions", { count: failedCount })}</p>
      <Link to={failedHref} className={cn(buttonVariants({ variant: "outline", size: "sm" }), "h-8 shrink-0")}>
        {t("dashboard.openFailedSubscriptions")}
      </Link>
    </div>
    <FailedSubscriptionsPreview items={failedPreview} total={failedCount} />
  </CardFooter>
}

function SetupDialogMount({ editing, status, onClose }: { editing: boolean; status?: SetupStatus; onClose: () => void }) {
  const { t } = useTranslation()
  if (!editing || !status) return null
  return <Suspense fallback={<Button variant="outline" disabled>{t("common.loading")}</Button>}>
    <SetupDialog status={status} onClose={onClose} />
  </Suspense>
}
