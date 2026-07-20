import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useId, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { api } from "@/lib/api/endpoints"
import type { OutboundGroup } from "@/lib/api/types"

function SelectorControl({ group }: { group: OutboundGroup }) {
  const client = useQueryClient()
  const mutation = useMutation({
    mutationFn: (tag: string) => api.nodes.select(group.tag, tag),
    onSuccess: () => client.invalidateQueries({ queryKey: ["nodes", "groups"] }),
    onError: (error: Error) => toast.error(error.message),
  })
  const items = group.all.map((tag) => ({ label: tag, value: tag }))
  return <Select items={items} value={group.now} onValueChange={(value) => mutation.mutate(String(value))}>
    <SelectTrigger aria-label={group.tag} className="w-full"><SelectValue /></SelectTrigger>
    <SelectContent><SelectGroup>{items.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectGroup></SelectContent>
  </Select>
}

function URLTestControl({ group }: { group: OutboundGroup }) {
  const { t } = useTranslation()
  const [delays, setDelays] = useState<Record<string, number>>({})
  const mutation = useMutation({ mutationFn: () => api.nodes.urlTest(group.tag), onSuccess: setDelays, onError: (error: Error) => toast.error(error.message) })
  return <div className="flex flex-col gap-2"><Button variant="outline" size="sm" disabled={mutation.isPending} onClick={() => mutation.mutate()}>{t("nodes.runURLTest", { group: group.tag })}</Button>
    {Object.entries(delays).map(([tag, delay]) => <span key={tag} className="text-sm text-muted-foreground">{tag}: {delay} ms</span>)}
  </div>
}

export function RuntimeGroupCard({ group, configType }: { group: OutboundGroup; configType?: string }) {
  const { t } = useTranslation()
  const mismatched = Boolean(configType && configType !== group.type)
  return <Card size="sm"><CardHeader><CardTitle>{group.tag}</CardTitle><CardDescription>{t("nodes.current")}: {group.now}</CardDescription><CardAction className="flex flex-wrap items-center justify-end gap-1"><Badge variant="outline">{group.type}</Badge>{mismatched ? <Badge variant="destructive">{t("nodes.groupTypeMismatchBadge")}</Badge> : null}</CardAction></CardHeader>
    <CardContent className="flex flex-col gap-3">
      {mismatched ? <Alert><AlertTitle>{t("nodes.groupTypeMismatchTitle")}</AlertTitle><AlertDescription>{t("nodes.groupTypeMismatchDescription", { config: configType, runtime: group.type })}</AlertDescription></Alert> : null}
      {group.type === "selector" ? <SelectorControl group={group} /> : <URLTestControl group={group} />}
    </CardContent>
  </Card>
}

export function RuntimeGroupsCard() {
  const { t } = useTranslation()
  const titleId = useId()
  const query = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups })
  const groups = query.data?.groups ?? []
  if (!groups.length) return null
  return <section aria-labelledby={titleId} className="flex flex-col gap-3">
    <div><h2 id={titleId} className="text-lg font-medium">{t("nodes.runtimeGroups")}</h2><p className="text-sm text-muted-foreground">{t("nodes.runtimeGroupsDescription")}</p></div>
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">{groups.map((group) => <RuntimeGroupCard key={group.tag} group={group} />)}</div>
  </section>
}
