import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  delayBatchToastTone,
  formatDelayBatchMessage,
  measureGroupDelays,
  pickPrimaryGroup,
  summarizeDelays,
  type DelayMap,
} from "@/features/dashboard/proxy-delay"
import { api } from "@/lib/api/endpoints"

export function useProxySelector() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [delays, setDelays] = useState<DelayMap>({})
  const query = useQuery({ queryKey: ["nodes", "groups"], queryFn: api.nodes.groups, refetchInterval: 5000 })
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
    mutationFn: async () => (!group ? {} as DelayMap : measureGroupDelays(group.tag, members)),
    onSuccess: (next) => {
      setDelays(next)
      const summary = summarizeDelays(next)
      const message = formatDelayBatchMessage(summary, t)
      const tone = delayBatchToastTone(summary)
      if (tone === "error") toast.error(message)
      else if (tone === "warning") toast.warning(message)
      else toast.success(message)
    },
    onError: (error: Error) => toast.error(error.message),
  })
  return {
    query, group, members, delays,
    select: (tag: string) => selectMutation.mutate(tag),
    selecting: selectMutation.isPending,
    probe: () => delayMutation.mutate(),
    probing: delayMutation.isPending,
  }
}
