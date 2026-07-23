import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { formatLatency } from "@/features/nodes/node-format"
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

function DelayBadge({ delay, failed }: { delay?: number; failed?: boolean }) {
  const { t } = useTranslation()
  if (failed) return <Badge variant="destructive">{t("dashboard.proxyDelayFailed")}</Badge>
  if (delay === undefined) return <Badge variant="outline">—</Badge>
  return <Badge variant="secondary">{formatLatency(delay)}</Badge>
}

export function ProxySelectorCard() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [delays, setDelays] = useState<Record<string, number | "error">>({})
  const query = useQuery({
    queryKey: ["nodes", "groups"],
    queryFn: api.nodes.groups,
    refetchInterval: 5000,
  })
  const group = pickPrimaryGroup(query.data?.groups ?? [])
  const members = useMemo(() => group?.all ?? [], [group])

  const selectMutation = useMutation({
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

  const delayMutation = useMutation({
    mutationFn: async () => {
      if (!group) return {} as Record<string, number | "error">
      // Prefer group URLTest when the group also supports it; otherwise probe members individually.
      try {
        const urlTest = await api.nodes.urlTest(group.tag)
        const next: Record<string, number | "error"> = {}
        for (const tag of members) {
          const value = urlTest[tag]
          next[tag] = typeof value === "number" ? value : "error"
        }
        return next
      } catch {
        const entries = await Promise.all(members.map(async (tag) => {
          try {
            const result = await api.nodes.delay(tag) as { delay?: number }
            return [tag, typeof result.delay === "number" ? result.delay : "error"] as const
          } catch {
            return [tag, "error"] as const
          }
        }))
        return Object.fromEntries(entries)
      }
    },
    onSuccess: (next) => setDelays(next),
    onError: (error: Error) => toast.error(error.message),
  })

  if (query.isLoading) return <Skeleton className="h-36 w-full" />
  if (query.error) {
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("dashboard.proxySelector")}</CardTitle>
          <CardDescription>{query.error.message}</CardDescription>
        </CardHeader>
      </Card>
    )
  }
  if (!group) {
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("dashboard.proxySelector")}</CardTitle>
          <CardDescription>{t("dashboard.proxySelectorEmpty")}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const items = members.map((tag) => ({ label: tag, value: tag }))
  const currentDelay = delays[group.now]
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="truncate">{t("dashboard.proxySelector")}</span>
          <Badge variant="outline">{group.tag}</Badge>
          <DelayBadge
            delay={typeof currentDelay === "number" ? currentDelay : undefined}
            failed={currentDelay === "error"}
          />
        </CardTitle>
        <CardDescription>{t("dashboard.proxySelectorDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <Select
          items={items}
          value={group.now}
          onValueChange={(value) => selectMutation.mutate(String(value))}
        >
          <SelectTrigger aria-label={t("dashboard.proxySelector")} className="h-8 w-full" disabled={selectMutation.isPending}>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {items.map((item) => {
                const delay = delays[item.value]
                const suffix = delay === "error"
                  ? ` (${t("dashboard.proxyDelayFailed")})`
                  : typeof delay === "number"
                    ? ` (${formatLatency(delay)})`
                    : ""
                return <SelectItem key={item.value} value={item.value}>{item.label}{suffix}</SelectItem>
              })}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          className="h-8 w-full sm:w-auto"
          disabled={delayMutation.isPending || members.length === 0}
          onClick={() => delayMutation.mutate()}
        >
          {delayMutation.isPending ? t("dashboard.proxyDelayTesting") : t("dashboard.proxyDelayTest")}
        </Button>
      </CardContent>
    </Card>
  )
}
