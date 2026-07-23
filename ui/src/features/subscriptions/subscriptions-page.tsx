import { PlusIcon, RefreshCcwIcon, Trash2Icon } from "lucide-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ImportedNodesCard } from "@/features/subscriptions/imported-nodes-card"
import { SubscriptionDialog } from "@/features/subscriptions/subscription-dialog"
import { formatRelativeTime } from "@/features/subscriptions/relative-time"
import {
  failedSubscriptionIds,
  filterSubscriptions,
  type SubscriptionFilter,
} from "@/features/subscriptions/subscription-list"
import { SubscriptionTrafficBadges } from "@/features/subscriptions/subscription-traffic"
import { api } from "@/lib/api/endpoints"
import type { Subscription } from "@/lib/api/types"

interface ItemProps { item: Subscription; onEdit: () => void; onRefresh: () => void; onDelete: () => void }

function urlTestStatus(item: Subscription, t: (key: string) => string) {
  const hasOverrides = Boolean(item.urltest && Object.values(item.urltest).some((value) => value !== undefined))
  if (item.urltest?.enabled === false) return t("subscriptions.urlTestOff")
  if (hasOverrides) return t("subscriptions.urlTestCustom")
  return t("subscriptions.urlTestInherited")
}

function SubscriptionItem({ item, onEdit, onRefresh, onDelete }: ItemProps) {
  const { t, i18n } = useTranslation()
  const [mountedAt] = useState(() => Date.now())
  return (
    <article aria-label={item.name}>
      <Card size="sm" className={item.error ? "border-destructive/40" : undefined}>
        <CardHeader>
          <CardTitle>{item.name}</CardTitle>
          <CardDescription className="break-all">{item.url}</CardDescription>
          <CardAction><Badge variant="outline">{t("subscriptions.nodeCount", { count: item.outbounds?.length ?? 0 })}</Badge></CardAction>
        </CardHeader>
        <CardContent className="flex flex-col gap-2">
          <span className="text-sm text-muted-foreground" title={item.last_updated ? new Date(item.last_updated).toLocaleString() : undefined}>
            {item.last_updated && !Number.isNaN(Date.parse(item.last_updated))
              ? t("subscriptions.updatedRelative", { time: formatRelativeTime(item.last_updated, mountedAt, i18n.language) })
              : t("subscriptions.neverUpdated")}
          </span>
          <div className="flex flex-wrap gap-2">
            <Badge variant={item.error ? "destructive" : "secondary"} title={item.error || undefined}>
              {item.error ? t("subscriptions.statusError") : t("common.normal")}
            </Badge>
            <Badge variant="outline">{urlTestStatus(item, t)}</Badge>
          </div>
          {item.error ? (
            <p className="line-clamp-3 text-sm text-destructive" title={item.error}>{item.error}</p>
          ) : null}
          <SubscriptionTrafficBadges traffic={item.traffic} />
        </CardContent>
        <CardFooter className="grid grid-cols-2 gap-2 sm:grid-cols-3">
          <Button size="sm" variant="outline" onClick={onEdit}>{t("common.edit")}</Button>
          <Button size="sm" variant="outline" onClick={onRefresh}>
            {item.error ? t("subscriptions.retry") : t("subscriptions.refresh")}
          </Button>
          <ConfirmAction
            trigger={<Button size="sm" variant="destructive" className="col-span-2 sm:col-span-1"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>}
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

export function SubscriptionsPage() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const listTitleId = useId()
  const [editing, setEditing] = useState<Subscription | "new" | null>(null)
  const [search, setSearch] = useState("")
  const [status, setStatus] = useState<SubscriptionFilter>("all")
  const [retrying, setRetrying] = useState(false)
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
  const loadError = query.error || defaults.error

  const retryFailed = async () => {
    if (!failedIds.length) return
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
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("subscriptions.title")}</h1>
        <div className="flex flex-wrap gap-2">
          {failedIds.length ? (
            <Button variant="outline" disabled={retrying} onClick={() => void retryFailed()}>
              <RefreshCcwIcon data-icon="inline-start" />
              {t("subscriptions.retryFailed", { count: failedIds.length })}
            </Button>
          ) : null}
          <Button variant="outline" onClick={refreshAll}><RefreshCcwIcon data-icon="inline-start" />{t("subscriptions.refreshAll")}</Button>
          <Button onClick={() => setEditing("new")}><PlusIcon data-icon="inline-start" />{t("subscriptions.add")}</Button>
        </div>
      </div>
      <section aria-labelledby={listTitleId} className="flex flex-col gap-3">
        <div>
          <h2 id={listTitleId} className="text-lg font-medium">{t("subscriptions.list")}</h2>
          <p className="text-sm text-muted-foreground">{t("subscriptions.description")}</p>
        </div>
        {items.length ? (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor="subscriptions-search">{t("subscriptions.search")}</label>
            <Input
              id="subscriptions-search"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("subscriptions.searchPlaceholder")}
              className="sm:max-w-sm"
            />
            <div className="flex flex-wrap gap-2">
              {([
                ["all", t("subscriptions.filterAll")],
                ["error", t("subscriptions.filterError")],
                ["ok", t("subscriptions.filterOk")],
              ] as const).map(([value, label]) => (
                <Button
                  key={value}
                  size="sm"
                  variant={status === value ? "default" : "outline"}
                  onClick={() => setStatus(value)}
                >
                  {label}
                </Button>
              ))}
            </div>
          </div>
        ) : null}
        {items.length
          ? visible.length
            ? <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
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
            : <Empty><EmptyHeader><EmptyTitle>{t("subscriptions.noMatch")}</EmptyTitle><EmptyDescription>{t("subscriptions.noMatchDescription")}</EmptyDescription></EmptyHeader></Empty>
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
