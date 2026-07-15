import { PlusIcon, RefreshCcwIcon, Trash2Icon } from "lucide-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { ConfirmAction } from "@/components/confirm-action"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { ImportedNodesCard } from "@/features/subscriptions/imported-nodes-card"
import { SubscriptionDialog } from "@/features/subscriptions/subscription-dialog"
import { api } from "@/lib/api/endpoints"
import type { Subscription } from "@/lib/api/types"

interface ItemProps { item: Subscription; onEdit: () => void; onRefresh: () => void; onDelete: () => void }

function SubscriptionItem({ item, onEdit, onRefresh, onDelete }: ItemProps) {
  const { t } = useTranslation()
  const hasURLTestOverrides = Boolean(item.urltest && Object.values(item.urltest).some((value) => value !== undefined))
  const urlTestStatus = item.urltest?.enabled === false ? t("subscriptions.urlTestOff") : hasURLTestOverrides ? t("subscriptions.urlTestCustom") : t("subscriptions.urlTestInherited")
  return <article aria-label={item.name}><Card size="sm"><CardHeader><CardTitle>{item.name}</CardTitle><CardDescription>{item.url}</CardDescription><CardAction><Badge variant="outline">{t("subscriptions.nodeCount", { count: item.outbounds?.length ?? 0 })}</Badge></CardAction></CardHeader>
    <CardContent className="flex flex-col gap-2"><span className="text-sm text-muted-foreground">{t("subscriptions.lastUpdated")}: {new Date(item.last_updated).toLocaleString()}</span><div className="flex flex-wrap gap-2"><Badge variant={item.error ? "destructive" : "secondary"}>{item.error || t("common.normal")}</Badge><Badge variant="outline">{urlTestStatus}</Badge></div></CardContent>
    <CardFooter className="grid grid-cols-2 gap-2 sm:grid-cols-3"><Button size="sm" variant="outline" onClick={onEdit}>{t("common.edit")}</Button><Button size="sm" variant="outline" onClick={onRefresh}>{t("subscriptions.refresh")}</Button>
      <ConfirmAction trigger={<Button size="sm" variant="destructive" className="col-span-2 sm:col-span-1"><Trash2Icon data-icon="inline-start" />{t("common.delete")}</Button>} title={t("common.deleteTitle")} description={t("common.deleteDescription")} confirmLabel={t("common.confirmDelete")} confirmVariant="destructive" onConfirm={onDelete} />
    </CardFooter>
  </Card></article>
}

export function SubscriptionsPage() {
  const { t } = useTranslation()
  const listTitleId = useId()
  const client = useQueryClient()
  const query = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const defaults = useQuery({ queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults })
  const [editing, setEditing] = useState<Subscription | "new" | null>(null)
  const refresh = () => Promise.all([client.invalidateQueries({ queryKey: ["subscriptions"] }), client.invalidateQueries({ queryKey: ["nodes"] })])
  const action = (request: Promise<unknown>, message: string) => request.then(() => api.nodes.sync()).then(refresh).then(() => toast.success(message)).catch((error: Error) => toast.error(error.message))
  const refreshAll = () => api.subscriptions.refreshAll().then((response) => {
    if (response.status === "partial") throw new Error(response.error?.message || t("subscriptions.partialFailure"))
    return api.nodes.sync()
  }).then(refresh).then(() => toast.success(t("subscriptions.refreshedAll"))).catch((error: Error) => toast.error(error.message))
  if (query.isLoading || defaults.isLoading) return <Skeleton className="h-64 w-full" />
  const loadError = query.error || defaults.error
  if (loadError) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{loadError.message}</AlertDescription></Alert>
  return <div className="flex flex-col gap-4"><div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><h1 className="text-2xl font-semibold">{t("subscriptions.title")}</h1><div className="grid grid-cols-2 gap-2"><Button variant="outline" onClick={refreshAll}><RefreshCcwIcon data-icon="inline-start" />{t("subscriptions.refreshAll")}</Button><Button onClick={() => setEditing("new")}><PlusIcon data-icon="inline-start" />{t("subscriptions.add")}</Button></div></div>
    <section aria-labelledby={listTitleId} className="flex flex-col gap-3"><div><h2 id={listTitleId} className="text-lg font-medium">{t("subscriptions.list")}</h2><p className="text-sm text-muted-foreground">{t("subscriptions.description")}</p></div>{query.data?.length ? <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">{query.data.map((item) => <SubscriptionItem key={item.id} item={item} onEdit={() => setEditing(item)} onRefresh={() => action(api.subscriptions.refresh(item.id), t("subscriptions.refresh"))} onDelete={() => action(api.subscriptions.delete(item.id), t("common.delete"))} />)}</div>
      : <Empty><EmptyHeader><EmptyTitle>{t("common.empty")}</EmptyTitle><EmptyDescription>{t("subscriptions.description")}</EmptyDescription></EmptyHeader></Empty>}</section>
    <ImportedNodesCard />
    {editing ? <SubscriptionDialog defaults={defaults.data!} item={editing === "new" ? undefined : editing} onClose={() => setEditing(null)} onSaved={() => { setEditing(null); void refresh() }} /> : null}
  </div>
}
