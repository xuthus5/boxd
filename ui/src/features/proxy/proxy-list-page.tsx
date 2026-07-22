import { useQuery } from "@tanstack/react-query"
import { PlusIcon, WandSparklesIcon } from "lucide-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { ProxyEditorDialog } from "@/features/proxy/proxy-editor-dialog"
import { InboundCard } from "@/features/proxy/inbound-card"
import { OutboundCard } from "@/features/proxy/outbound-card"
import { RuntimeGroupCard } from "@/features/nodes/runtime-groups-card"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, OutboundGroup, Subscription } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>
interface Editing { index: number; item: JsonObject }

function objects(value: JsonValue | undefined) {
  return Array.isArray(value) ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item))) : []
}

function InboundCards({ items, onEdit, onDelete }: { items: JsonObject[]; onEdit: (index: number) => void; onDelete: (index: number) => void }) {
  return <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{items.map((item, index) => (
    <InboundCard key={`${String(item.tag)}-${index}`} item={item} onEdit={() => onEdit(index)} onDelete={() => onDelete(index)} />
  ))}</div>
}

function configGroup(item: JsonObject): OutboundGroup | null {
  const type = String(item.type ?? "")
  const tag = String(item.tag ?? "")
  const all = Array.isArray(item.outbounds) ? item.outbounds.filter((member): member is string => typeof member === "string") : []
  if (!tag || !["selector", "urltest"].includes(type) || !all.length) return null
  return { type, tag, all, now: typeof item.default === "string" ? item.default : all[0] }
}

function subscriptionTags(subscriptions: Subscription[]) {
  return new Set(subscriptions.flatMap((subscription) => subscription.outbounds?.map((outbound) => outbound.tag) ?? []))
}

function OutboundCards({ items, onEdit, onDelete }: { items: JsonObject[]; onEdit: (index: number) => void; onDelete: (index: number) => void }) {
  const { t } = useTranslation()
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const runtime = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups })
  if (subscriptions.isLoading || runtime.isLoading) return <Skeleton className="h-64 w-full" />
  const error = subscriptions.error ?? runtime.error
  if (error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{error.message}</AlertDescription></Alert>
  const subscriptionList = Array.isArray(subscriptions.data) ? subscriptions.data : []
  const memberTags = subscriptionTags(subscriptionList)
  const subscriptionNames = new Set(subscriptionList.map((subscription) => subscription.name))
  const runtimeList = Array.isArray(runtime.data?.groups) ? runtime.data.groups : []
  const runtimeGroups = new Map(runtimeList.map((group) => [group.tag, group]))
  const indexedItems = items.map((item, index) => ({ item, index }))
  const independent = indexedItems.filter(({ item }) => !memberTags.has(String(item.tag ?? "")) && !subscriptionNames.has(String(item.tag ?? "")))
  const groups = subscriptionList.flatMap((subscription) => {
    if (!subscription.outbounds?.length) return []
    const configured = indexedItems.find(({ item }) => item.tag === subscription.name)
    const configuredGroup = configured ? configGroup(configured.item) : null
    const group = runtimeGroups.get(subscription.name) ?? configuredGroup
    if (!group) return []
    return [{ group, configType: configuredGroup?.type ?? (typeof configured?.item.type === "string" ? configured.item.type : undefined) }]
  })
  return <div className="flex flex-col gap-4">
    {groups.length ? <section className="flex flex-col gap-3"><div><h2 className="text-lg font-medium">{t("proxy.outbound.group")}</h2><p className="text-sm text-muted-foreground">{t("proxy.description")}</p></div><div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{groups.map(({ group, configType }) => <RuntimeGroupCard key={group.tag} group={group} configType={configType} />)}</div></section> : null}
    {independent.length ? <section className="flex flex-col gap-3"><div><h2 className="text-lg font-medium">{t("proxy.outbound.protocol")}</h2><p className="text-sm text-muted-foreground">{t("proxy.description")}</p></div><div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{independent.map(({ item, index }) => <OutboundCard key={`${String(item.tag)}-${index}`} item={item} onEdit={() => onEdit(index)} onDelete={() => onDelete(index)} />)}</div></section> : null}
  </div>
}

export function ProxyListPage({ configKey, title, addLabel }: { configKey: "inbounds" | "outbounds"; title: string; addLabel: string }) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const [editing, setEditing] = useState<Editing | null>(null)
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  const items = objects(query.data?.[configKey])
  const persist = (nextItems: JsonObject[]) => save.mutate({ ...query.data!, [configKey]: nextItems }, {
    onSuccess: (response) => response.status === "rolled_back" ? toast.error(t("proxy.rolledBack")) : toast.success(t("proxy.saved")),
    onError: (error) => toast.error(error.message),
  })
  const saveItem = (item: JsonObject) => {
    const next = [...items]
    if (editing!.index < 0) next.push(item); else next[editing!.index] = item
    persist(next); setEditing(null)
  }
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><h1 className="text-2xl font-semibold">{title}</h1><div className="grid grid-cols-2 gap-2 sm:flex">
        {configKey === "outbounds" ? <Button variant="outline" onClick={() => api.config.installOutbounds().then(() => query.refetch()).then(() => toast.success(t("proxy.defaultsInstalled"))).catch((error: Error) => toast.error(error.message))}><WandSparklesIcon data-icon="inline-start" />{t("proxy.installDefaults")}</Button> : null}
        {configKey === "inbounds" ? <Button variant="outline" onClick={() => api.config.installInbounds().then(() => query.refetch()).then(() => toast.success(t("proxy.inboundDefaultsInstalled"))).catch((error: Error) => toast.error(error.message))}><WandSparklesIcon data-icon="inline-start" />{t("proxy.installInboundDefaults")}</Button> : null}
        <Button onClick={() => setEditing({ index: -1, item: {} })}><PlusIcon data-icon="inline-start" />{addLabel}</Button>
      </div></div>
      {(configKey === "inbounds" || configKey === "outbounds") && items.length > 0 ? configKey === "inbounds" ? <InboundCards items={items} onEdit={(index) => setEditing({ index, item: items[index] })} onDelete={(index) => persist(items.filter((_, itemIndex) => itemIndex !== index))} /> : <OutboundCards items={items} onEdit={(index) => setEditing({ index, item: items[index] })} onDelete={(index) => persist(items.filter((_, itemIndex) => itemIndex !== index))} /> : <Card><CardHeader><CardTitle>{title}{t("proxy.listSuffix")}</CardTitle><CardDescription>{t("proxy.description")}</CardDescription></CardHeader><CardContent>
        <Empty><EmptyHeader><EmptyTitle>{t("proxy.empty")}</EmptyTitle><EmptyDescription>{t("proxy.emptyDescription")}</EmptyDescription></EmptyHeader><EmptyContent><Button onClick={() => setEditing({ index: -1, item: {} })}>{addLabel}</Button></EmptyContent></Empty>
      </CardContent></Card>}
      {editing ? <ProxyEditorDialog key={`${editing.index}-${String(editing.item.tag)}`} title={editing.index < 0 ? addLabel : `${t("proxy.editPrefix")} ${String(editing.item.tag ?? "")}`} kind={configKey} item={editing.item} onClose={() => setEditing(null)} onSave={saveItem} /> : null}
    </div>
  )
}
