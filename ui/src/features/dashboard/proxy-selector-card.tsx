import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api/endpoints"
import type { OutboundGroup } from "@/lib/api/types"

const preferredTags = ["proxy", "select", "GLOBAL"]

function pickPrimaryGroup(groups: OutboundGroup[]) {
  const selectors = groups.filter((group) => group.type === "selector" && group.all.length > 0)
  if (!selectors.length) return null
  for (const tag of preferredTags) {
    const found = selectors.find((group) => group.tag === tag)
    if (found) return found
  }
  return selectors[0]
}

export function ProxySelectorCard() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const query = useQuery({
    queryKey: ["nodes", "groups"],
    queryFn: api.nodes.groups,
    refetchInterval: 5000,
  })
  const group = pickPrimaryGroup(query.data?.groups ?? [])
  const mutation = useMutation({
    mutationFn: (tag: string) => {
      if (!group) throw new Error("missing group")
      return api.nodes.select(group.tag, tag)
    },
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["nodes", "groups"] })
      toast.success(t("dashboard.proxySelected"))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (query.isLoading) return <Skeleton className="h-36 w-full" />
  if (query.error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("dashboard.proxySelector")}</CardTitle>
          <CardDescription>{query.error.message}</CardDescription>
        </CardHeader>
      </Card>
    )
  }
  if (!group) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>{t("dashboard.proxySelector")}</CardTitle>
          <CardDescription>{t("dashboard.proxySelectorEmpty")}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const items = group.all.map((tag) => ({ label: tag, value: tag }))
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          {t("dashboard.proxySelector")}
          <Badge variant="outline">{group.tag}</Badge>
        </CardTitle>
        <CardDescription>{t("dashboard.proxySelectorDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <Select
          items={items}
          value={group.now}
          onValueChange={(value) => mutation.mutate(String(value))}
        >
          <SelectTrigger aria-label={t("dashboard.proxySelector")} className="w-full" disabled={mutation.isPending}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {items.map((item) => (
                <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </CardContent>
    </Card>
  )
}
