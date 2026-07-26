import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"

import { Skeleton } from "@/components/ui/skeleton"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { InboundCard } from "@/features/proxy/inbound-card"
import { OutboundCard } from "@/features/proxy/outbound-card"
import { RuntimeGroupCard } from "@/features/nodes/runtime-groups-card"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, OutboundGroup, Subscription } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>
interface IndexedItem { item: JsonObject; index: number }

export function InboundCards({ items, onEdit, onDelete, onPatch, busy }: {
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

function OutboundSections({ groups, independent, onEdit, onDelete }: {
  groups: { group: OutboundGroup; configType?: string }[]
  independent: IndexedItem[]
  onEdit: (index: number) => void
  onDelete: (index: number) => void
}) {
  const { t } = useTranslation()
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

export function OutboundCards({ items, onEdit, onDelete }: {
  items: IndexedItem[]
  onEdit: (index: number) => void
  onDelete: (index: number) => void
}) {
  const subscriptions = useQuery({ queryKey: ["subscriptions"], queryFn: api.subscriptions.list })
  const runtime = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups })
  if (subscriptions.isLoading || runtime.isLoading) return <Skeleton className="h-64 w-full" />
  const error = subscriptions.error ?? runtime.error
  if (error) {
    return <PageLoadErrorAlert error={error} scope="proxy-outbounds" onRetry={() => {
      if (subscriptions.error) void subscriptions.refetch()
      if (runtime.error) void runtime.refetch()
    }} />
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
  return <OutboundSections groups={groups} independent={independent} onEdit={onEdit} onDelete={onDelete} />
}
