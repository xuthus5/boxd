import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
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
    <SelectTrigger aria-label={group.tag}><SelectValue /></SelectTrigger>
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

export function RuntimeGroupsCard() {
  const { t } = useTranslation()
  const query = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups })
  const groups = query.data?.groups ?? []
  if (!groups.length) return null
  return <Card><CardHeader><CardTitle>{t("nodes.runtimeGroups")}</CardTitle><CardDescription>{t("nodes.runtimeGroupsDescription")}</CardDescription></CardHeader><CardContent>
    <Table><TableHeader><TableRow><TableHead>Tag</TableHead><TableHead>{t("common.type")}</TableHead><TableHead>{t("nodes.current")}</TableHead><TableHead>{t("common.actions")}</TableHead></TableRow></TableHeader>
      <TableBody>{groups.map((group) => <TableRow key={group.tag}><TableCell>{group.tag}</TableCell><TableCell>{group.type}</TableCell><TableCell>{group.now}</TableCell><TableCell>{group.type === "selector" ? <SelectorControl group={group} /> : <URLTestControl group={group} />}</TableCell></TableRow>)}</TableBody>
    </Table>
  </CardContent></Card>
}
