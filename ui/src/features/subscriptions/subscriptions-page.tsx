import { SubscriptionItem } from "@/features/subscriptions/subscription-item"
import { PlusIcon, RefreshCcwIcon } from "lucide-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ImportedNodesCard } from "@/features/subscriptions/imported-nodes-card"
import { SubscriptionDialog } from "@/features/subscriptions/subscription-dialog"
import {
  failedSubscriptionIds,
  filterSubscriptions,
  parseSubscriptionSearchParams,
  subscriptionFiltersActive,
  summarizeSubscriptionStatus,
  toSubscriptionSearchParams,
  type SubscriptionFilter,
  type SubscriptionListFilters,
} from "@/features/subscriptions/subscription-list"
import { SubscriptionStatusSummaryBar } from "@/features/subscriptions/subscription-status-summary"
import { api } from "@/lib/api/endpoints"
import type { Subscription } from "@/lib/api/types"

export function SubscriptionsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const listTitleId = useId()
  const [editing, setEditing] = useState<Subscription | "new" | null>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseSubscriptionSearchParams(searchParams), [searchParams])
  const search = filters.query ?? ""
  const status = (filters.status ?? "all") as SubscriptionFilter
  const [retrying, setRetrying] = useState(false)
  const writeFilters = (next: SubscriptionListFilters) => {
    setSearchParams(toSubscriptionSearchParams(next), { replace: true })
  }
  const query = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const defaults = useQuery({ queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults })
  const refresh = () => Promise.all([
    client.invalidateQueries({ queryKey: ["subscriptions"] }),
    client.invalidateQueries({ queryKey: ["nodes"] }),
  ])
  const action = (request: Promise<unknown>, message: string) => request
    .then(refresh)
    .then(() => toast.success(message))
    .catch((error: Error) => toast.error(error.message))
  const refreshAll = () => api.subscriptions.refreshAll().then((response) => {
    if (response.status === "partial") throw new Error(response.error?.message || t("subscriptions.partialFailure"))
    return response
  }).then(refresh).then(() => toast.success(t("subscriptions.refreshedAll"))).catch((error: Error) => toast.error(error.message))

  const items = useMemo(
    () => (Array.isArray(query.data) ? query.data : []),
    [query.data],
  )
  const failedIds = useMemo(() => failedSubscriptionIds(items), [items])
  const visible = useMemo(
    () => filterSubscriptions(items, { query: search, status }),
    [items, search, status],
  )
  const statusSummary = useMemo(
    () => summarizeSubscriptionStatus(items, search),
    [items, search],
  )
  const loadError = query.error || defaults.error

  const retryFailed = async () => {
    if (!failedIds.length) return
    writeFilters({ query: filters.query, status: "error" })
    setRetrying(true)
    let ok = 0
    let failed = 0
    try {
      for (const id of failedIds) {
        try {
          await api.subscriptions.refresh(id)
          ok += 1
        } catch {
          failed += 1
        }
      }
      await refresh()
      if (failed === 0) toast.success(t("subscriptions.retryFailedDone", { count: ok }))
      else toast.error(t("subscriptions.retryFailedPartial", { ok, failed }))
    } finally {
      setRetrying(false)
    }
  }

  if (query.isLoading || defaults.isLoading) return <Skeleton className="h-64 w-full" />
  if (loadError) {
    return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{loadError.message}</AlertDescription></Alert>
  }

  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("subscriptions.title")}</h1>
        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap">
          {failedIds.length ? (
            <Button
              variant="outline"
              size="sm"
              className="col-span-2 h-8 sm:col-span-1"
              disabled={retrying}
              onClick={() => void retryFailed()}
            >
              <RefreshCcwIcon data-icon="inline-start" />
              {t("subscriptions.retryFailed", { count: failedIds.length })}
            </Button>
          ) : null}
          <Button variant="outline" size="sm" className="h-8" onClick={refreshAll}>
            <RefreshCcwIcon data-icon="inline-start" />
            {t("subscriptions.refreshAll")}
          </Button>
          <Button size="sm" className="h-8" onClick={() => setEditing("new")}>
            <PlusIcon data-icon="inline-start" />
            {t("subscriptions.add")}
          </Button>
        </div>
      </div>
      <section aria-labelledby={listTitleId} className="flex flex-col gap-2.5 sm:gap-3">
        <div>
          <h2 id={listTitleId} className="text-base font-medium sm:text-lg">{t("subscriptions.list")}</h2>
          <p className="text-xs text-muted-foreground sm:text-sm">{t("subscriptions.description")}</p>
        </div>
        {items.length ? (
          <div className="flex flex-col gap-2">
            <label className="sr-only" htmlFor="subscriptions-search">{t("subscriptions.search")}</label>
            <Input
              id="subscriptions-search"
              value={search}
              onChange={(event) => writeFilters({ query: event.target.value, status: filters.status })}
              placeholder={t("subscriptions.searchPlaceholder")}
              className="h-8 sm:max-w-sm"
              aria-label={t("subscriptions.search")}
            />
            <div className="flex flex-wrap gap-1.5">
              {([
                ["all", t("subscriptions.filterAll")],
                ["error", t("subscriptions.filterError")],
                ["ok", t("subscriptions.filterOk")],
              ] as const).map(([value, label]) => (
                <Button
                  key={value}
                  size="sm"
                  className="h-7"
                  variant={status === value ? "default" : "outline"}
                  aria-pressed={status === value}
                  onClick={() => writeFilters({
                    query: filters.query,
                    status: value === "all" ? undefined : value,
                  })}
                >
                  {label}
                </Button>
              ))}
              {subscriptionFiltersActive(filters) ? (
                <Button
                  size="sm"
                  className="h-7"
                  variant="ghost"
                  onClick={() => writeFilters({})}
                >
                  {t("subscriptions.clearFilters")}
                </Button>
              ) : null}
            </div>
          </div>
        ) : null}
        {items.length ? (
          <SubscriptionStatusSummaryBar summary={statusSummary} filters={filters} onChange={writeFilters} />
        ) : null}
        {items.length
          ? visible.length
            ? <div className="grid gap-2 sm:grid-cols-2 sm:gap-3 xl:grid-cols-3">
              {visible.map((item) => (
                <SubscriptionItem
                  key={item.id}
                  item={item}
                  onEdit={() => setEditing(item)}
                  onRefresh={() => action(api.subscriptions.refresh(item.id), item.error ? t("subscriptions.retry") : t("subscriptions.refresh"))}
                  onDelete={() => action(api.subscriptions.delete(item.id), t("common.delete"))}
                />
              ))}
            </div>
            : <Empty>
              <EmptyHeader>
                <EmptyTitle>{t("subscriptions.noMatch")}</EmptyTitle>
                <EmptyDescription>{t("subscriptions.noMatchDescription")}</EmptyDescription>
              </EmptyHeader>
              <EmptyContent>
                <Button variant="outline" onClick={() => writeFilters({})}>
                  {t("subscriptions.clearFilters")}
                </Button>
              </EmptyContent>
            </Empty>
          : <Empty><EmptyHeader><EmptyTitle>{t("common.empty")}</EmptyTitle><EmptyDescription>{t("subscriptions.description")}</EmptyDescription></EmptyHeader></Empty>}
      </section>
      <ImportedNodesCard />
      {editing
        ? <SubscriptionDialog
          defaults={defaults.data!}
          item={editing === "new" ? undefined : editing}
          onClose={() => setEditing(null)}
          onSaved={() => { setEditing(null); void refresh() }}
        />
        : null}
    </div>
  )
}
