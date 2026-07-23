import { useQuery } from "@tanstack/react-query"
import { PlusIcon, WandSparklesIcon } from "lucide-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { RuntimeGroupCard } from "@/features/nodes/runtime-groups-card"
import { InboundCard } from "@/features/proxy/inbound-card"
import { OutboundCard } from "@/features/proxy/outbound-card"
import { ProxyEditorDialog } from "@/features/proxy/proxy-editor-dialog"
import {
  filterProxyItems,
  parseProxySearchParams,
  proxyFiltersActive,
  summarizeProxyTypes,
  toProxySearchParams,
  type ProxyListFilters,
} from "@/features/proxy/proxy-filter"
import { ProxyTypeSummaryBar } from "@/features/proxy/proxy-type-summary"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, OutboundGroup, Subscription } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>
interface Editing { index: number; item: JsonObject }
interface IndexedItem { item: JsonObject; index: number }

function objects(value: JsonValue | undefined) {
  return Array.isArray(value)
    ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item)))
    : []
}

function InboundCards({ items, onEdit, onDelete, onPatch, busy }: {
  items: IndexedItem[]
  onEdit: (index: number) => void
  onDelete: (index: number) => void
  onPatch: (index: number, item: JsonObject) => void
  busy?: boolean
}) {
  return (
    <div className="grid gap-2 sm:gap-3 md:grid-cols-2 xl:grid-cols-3">
      {items.map(({ item, index }) => (
        <InboundCard
          key={`${String(item.tag)}-${index}`}
          item={item}
          busy={busy}
          onEdit={() => onEdit(index)}
          onDelete={() => onDelete(index)}
          onPatch={(next) => onPatch(index, next)}
        />
      ))}
    </div>
  )
}

function configGroup(item: JsonObject): OutboundGroup | null {
  const type = String(item.type ?? "")
  const tag = String(item.tag ?? "")
  const all = Array.isArray(item.outbounds)
    ? item.outbounds.filter((member): member is string => typeof member === "string")
    : []
  if (!tag || !["selector", "urltest"].includes(type) || !all.length) return null
  return { type, tag, all, now: typeof item.default === "string" ? item.default : all[0] }
}

function subscriptionTags(subscriptions: Subscription[]) {
  return new Set(subscriptions.flatMap((subscription) => subscription.outbounds?.map((outbound) => outbound.tag) ?? []))
}

function OutboundCards({ items, onEdit, onDelete }: {
  items: IndexedItem[]
  onEdit: (index: number) => void
  onDelete: (index: number) => void
}) {
  const { t } = useTranslation()
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const runtime = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups })
  if (subscriptions.isLoading || runtime.isLoading) return <Skeleton className="h-64 w-full" />
  const error = subscriptions.error ?? runtime.error
  if (error) {
    return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{error.message}</AlertDescription></Alert>
  }
  const subscriptionList = Array.isArray(subscriptions.data) ? subscriptions.data : []
  const memberTags = subscriptionTags(subscriptionList)
  const subscriptionNames = new Set(subscriptionList.map((subscription) => subscription.name))
  const runtimeGroups = new Map((Array.isArray(runtime.data?.groups) ? runtime.data.groups : []).map((group) => [group.tag, group]))
  const independent = items.filter(({ item }) => !memberTags.has(String(item.tag ?? "")) && !subscriptionNames.has(String(item.tag ?? "")))
  const groups = subscriptionList.flatMap((subscription) => {
    if (!subscription.outbounds?.length) return []
    const configured = items.find(({ item }) => item.tag === subscription.name)
    const configuredGroup = configured ? configGroup(configured.item) : null
    const group = runtimeGroups.get(subscription.name) ?? configuredGroup
    if (!group) return []
    const configType = configuredGroup?.type ?? (typeof configured?.item.type === "string" ? configured.item.type : undefined)
    return [{ group, configType }]
  })
  return (
    <div className="flex flex-col gap-4">
      {groups.length ? (
        <section className="flex flex-col gap-3">
          <div>
            <h2 className="text-lg font-medium">{t("proxy.outbound.group")}</h2>
            <p className="text-sm text-muted-foreground">{t("proxy.description")}</p>
          </div>
          <div className="grid gap-2 sm:gap-3 md:grid-cols-2 xl:grid-cols-3">
            {groups.map(({ group, configType }) => <RuntimeGroupCard key={group.tag} group={group} configType={configType} />)}
          </div>
        </section>
      ) : null}
      {independent.length ? (
        <section className="flex flex-col gap-3">
          <div>
            <h2 className="text-lg font-medium">{t("proxy.outbound.protocol")}</h2>
            <p className="text-sm text-muted-foreground">{t("proxy.description")}</p>
          </div>
          <div className="grid gap-2 sm:gap-3 md:grid-cols-2 xl:grid-cols-3">
            {independent.map(({ item, index }) => (
              <OutboundCard key={`${String(item.tag)}-${index}`} item={item} onEdit={() => onEdit(index)} onDelete={() => onDelete(index)} />
            ))}
          </div>
        </section>
      ) : null}
    </div>
  )
}

export function ProxyListPage({ configKey, title, addLabel }: {
  configKey: "inbounds" | "outbounds"
  title: string
  addLabel: string
}) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseProxySearchParams(searchParams), [searchParams])
  const search = filters.query ?? ""
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{query.error.message}</AlertDescription></Alert>
  }
  const items = objects(query.data?.[configKey])
  const typeFilter = filters.type
  const filteredIndexed = items
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => filterProxyItems([item], { query: search, type: typeFilter }).length > 0)
  const typeSummary = summarizeProxyTypes(items, search)
  const facetsActive = proxyFiltersActive({ query: search, type: typeFilter })
  const writeFilters = (next: ProxyListFilters) => {
    setSearchParams(toProxySearchParams(next), { replace: true })
  }
  const writeQuery = (queryValue: string) => {
    writeFilters({ query: queryValue, type: typeFilter })
  }
  const persist = (nextItems: JsonObject[]) => {
    clearSaveError()
    save.mutate({ ...query.data!, [configKey]: nextItems }, {
      onSuccess: (response) => {
        if (response.status === "rolled_back") {
          reportRollback(response, t("proxy.rolledBack"))
          return
        }
        toast.success(t("proxy.saved"))
      },
      onError: (error) => { reportError(error) },
    })
  }
  const saveItem = (item: JsonObject) => {
    const next = [...items]
    if (editing!.index < 0) next.push(item)
    else next[editing!.index] = item
    persist(next)
    setEditing(null)
  }
  const installDefaults = () => {
    const action = configKey === "outbounds" ? api.config.installOutbounds() : api.config.installInbounds()
    const success = configKey === "outbounds" ? t("proxy.defaultsInstalled") : t("proxy.inboundDefaultsInstalled")
    clearSaveError()
    action.then(() => query.refetch()).then(() => toast.success(success)).catch((error: Error) => {
      reportError(error)
    })
  }
  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{title}</h1>
        <div className="grid grid-cols-2 gap-2 sm:flex">
          <Button variant="outline" size="sm" className="h-8" onClick={installDefaults}>
            <WandSparklesIcon data-icon="inline-start" />
            {configKey === "outbounds" ? t("proxy.installDefaults") : t("proxy.installInboundDefaults")}
          </Button>
          <Button size="sm" className="h-8" onClick={() => setEditing({ index: -1, item: {} })}>
            <PlusIcon data-icon="inline-start" />{addLabel}
          </Button>
        </div>
      </div>
      <ConfigSaveErrorAlert error={saveError} onDismiss={clearSaveError} />
      {items.length > 0 ? (
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor={`proxy-search-${configKey}`}>{t("proxy.search")}</label>
            <Input id={`proxy-search-${configKey}`} value={search} onChange={(event) => writeQuery(event.target.value)} placeholder={t("proxy.searchPlaceholder")} className="h-8 sm:max-w-sm" aria-label={t("proxy.search")} />
            {facetsActive ? <p className="text-sm text-muted-foreground">{t("proxy.searchCount", { shown: filteredIndexed.length, total: items.length })}</p> : null}
          </div>
          <ProxyTypeSummaryBar summary={typeSummary} filters={filters} onChange={writeFilters} />
        </div>
      ) : null}
      {items.length > 0 ? (
        filteredIndexed.length > 0 ? (
          configKey === "inbounds" ? (
            <InboundCards
              items={filteredIndexed}
              busy={save.isPending}
              onEdit={(index) => setEditing({ index, item: items[index] })}
              onDelete={(index) => persist(items.filter((_, itemIndex) => itemIndex !== index))}
              onPatch={(index, item) => { const next = [...items]; next[index] = item; persist(next) }}
            />
          ) : (
            <OutboundCards
              items={filteredIndexed}
              onEdit={(index) => setEditing({ index, item: items[index] })}
              onDelete={(index) => persist(items.filter((_, itemIndex) => itemIndex !== index))}
            />
          )
        ) : (
          <Card>
            <CardHeader><CardTitle>{t("proxy.noMatch")}</CardTitle><CardDescription>{t("proxy.noMatchDescription")}</CardDescription></CardHeader>
            <CardContent><Button variant="outline" onClick={() => writeFilters({})}>{t("proxy.clearSearch")}</Button></CardContent>
          </Card>
        )
      ) : (
        <Card>
          <CardHeader><CardTitle>{title}{t("proxy.listSuffix")}</CardTitle><CardDescription>{t("proxy.description")}</CardDescription></CardHeader>
          <CardContent>
            <Empty>
              <EmptyHeader><EmptyTitle>{t("proxy.empty")}</EmptyTitle><EmptyDescription>{t("proxy.emptyDescription")}</EmptyDescription></EmptyHeader>
              <EmptyContent><Button onClick={() => setEditing({ index: -1, item: {} })}>{addLabel}</Button></EmptyContent>
            </Empty>
          </CardContent>
        </Card>
      )}
      {editing ? (
        <ProxyEditorDialog
          key={`${editing.index}-${String(editing.item.tag)}`}
          title={editing.index < 0 ? addLabel : `${t("proxy.editPrefix")} ${String(editing.item.tag ?? "")}`}
          kind={configKey}
          item={editing.item}
          onClose={() => setEditing(null)}
          onSave={saveItem}
        />
      ) : null}
    </div>
  )
}
