import { SubscriptionItem } from "@/features/subscriptions/subscription-item"
import { PlusIcon, RefreshCcwIcon } from "lucide-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

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
import {
  reportSubscriptionRefreshBatch,
  reportSubscriptionRequestError,
} from "@/features/subscriptions/subscription-error-actions"
import {
  classifySubscriptionRequestError,
  extractSubscriptionRefreshFailures,
  extractSubscriptionSyncError,
  formatSubscriptionRefreshBatchMessage,
  summarizeSubscriptionRefreshFailures,
} from "@/features/subscriptions/subscription-error"
import { api } from "@/lib/api/endpoints"
import type { Subscription } from "@/lib/api/types"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

export function SubscriptionsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const listTitleId = useId()
  const [editing, setEditing] = useState<Subscription | "new" | null>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseSubscriptionSearchParams(searchParams), [searchParams])
  const search = filters.query ?? ""
  const status = (filters.status ?? "all") as SubscriptionFilter
  const [batchAction, setBatchAction] = useState<"refresh-all" | "retry-failed" | null>(null)
  const batchActionInFlight = useRef(false)
  const writeFilters = (next: SubscriptionListFilters) => {
    setSearchParams(toSubscriptionSearchParams(next), { replace: true })
  }
  const query = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const defaults = useQuery({ queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults })
  const refresh = () => Promise.all([
    client.invalidateQueries({ queryKey: ["subscriptions"] }),
    client.invalidateQueries({ queryKey: ["nodes"] }),
  ])
  const action = (
    request: Promise<unknown>,
    message: string,
    options: { scope?: string; id?: string; name?: string; fallback?: string } = {},
  ) => request
    .then(refresh)
    .then(() => toast.success(message))
    .catch((error: Error) => reportSubscriptionRequestError(error, t, options))

  const refreshAll = async () => {
    if (batchActionInFlight.current) return
    batchActionInFlight.current = true
    setBatchAction("refresh-all")
    try {
      const response = await api.subscriptions.refreshAll()
      if (response.status === "partial") {
        const failures = extractSubscriptionRefreshFailures(response.data)
        const syncError = extractSubscriptionSyncError(response.data)
        if (syncError) {
          failures.unshift({
            name: t("subscriptions.configSync"),
            code: "sync_failed",
            message: syncError,
          })
        }
        const metaCount = Number((response.meta as { failed_count?: number } | null)?.failed_count ?? 0)
        const summary = summarizeSubscriptionRefreshFailures(
          failures.length
            ? failures
            : Array.from({ length: Math.max(metaCount, 1) }, () => ({
              message: response.error?.message || t("subscriptions.partialFailure"),
            })),
        )
        if (summary.failed === 0) summary.failed = Math.max(metaCount, summary.failedSamples.length, 1)
        const detail = formatSubscriptionRefreshBatchMessage(summary, t)
        reportSubscriptionRefreshBatch(summary, detail || t("subscriptions.partialFailure"), t)
        await refresh()
        return
      }
      await refresh()
      toast.success(t("subscriptions.refreshedAll"))
    } catch (error) {
      reportSubscriptionRequestError(error, t, {
        scope: "refresh-all",
        fallback: t("subscriptions.refreshFailed"),
      })
    } finally {
      batchActionInFlight.current = false
      setBatchAction(null)
    }
  }

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
    if (!failedIds.length || batchActionInFlight.current) return
    writeFilters({ query: filters.query, status: "error" })
    batchActionInFlight.current = true
    setBatchAction("retry-failed")
    let ok = 0
    const failures: Array<{ id?: string; name?: string; code?: string; message?: string }> = []
    try {
      for (const id of failedIds) {
        const item = items.find((entry) => entry.id === id)
        try {
          await api.subscriptions.refresh(id)
          ok += 1
        } catch (error) {
          const message = error instanceof Error ? error.message : t("subscriptions.refreshFailed")
          failures.push({
            id,
            name: item?.name,
            code: classifySubscriptionRequestError(error),
            message,
          })
        }
      }
      await refresh()
      if (failures.length === 0) {
        toast.success(t("subscriptions.retryFailedDone", { count: ok }))
        return
      }
      const summary = summarizeSubscriptionRefreshFailures(failures)
      const message = t("subscriptions.retryFailedPartial", { ok, failed: failures.length })
      reportSubscriptionRefreshBatch(summary, message, t)
    } finally {
      batchActionInFlight.current = false
      setBatchAction(null)
    }
  }

  if (query.isLoading || defaults.isLoading) return <Skeleton className="h-64 w-full" />
  if (loadError) {
    return (
      <PageLoadErrorAlert
        error={loadError}
        scope="subscriptions"
        onRetry={() => {
          if (query.error) void query.refetch()
          if (defaults.error) void defaults.refetch()
        }}
      />
    )
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
              disabled={batchAction !== null}
              onClick={() => void retryFailed()}
            >
              <RefreshCcwIcon data-icon="inline-start" />
              {t("subscriptions.retryFailed", { count: failedIds.length })}
            </Button>
          ) : null}
          <Button variant="outline" size="sm" className="h-8" disabled={batchAction !== null} aria-busy={batchAction === "refresh-all"} onClick={() => void refreshAll()}>
            <RefreshCcwIcon className={batchAction === "refresh-all" ? "animate-spin" : undefined} data-icon="inline-start" />
            {t(batchAction === "refresh-all" ? "subscriptions.refreshingAll" : "subscriptions.refreshAll")}
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
                  onRefresh={() => action(api.subscriptions.refresh(item.id), item.error ? t("subscriptions.retryDone") : t("subscriptions.refreshDone"), { scope: "refresh", id: item.id, name: item.name, fallback: t("subscriptions.refreshFailed") })}
                  onDelete={() => action(api.subscriptions.delete(item.id).then(() => api.nodes.sync()), t("subscriptions.deleted"), { scope: "delete", id: item.id, name: item.name, fallback: t("subscriptions.deleteFailed") })}
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
