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
  delayBatchFailureClipboardText,
  delayErrorHintKey,
  delayFailureFromError,
  delayRequestErrorClipboardText,
  formatDelayRequestErrorToast,
} from "@/features/dashboard/proxy-delay"
import {
  classifyNodeRequestError,
  formatNodeRequestErrorToast,
  nodeRequestErrorClipboardText,
  nodeRequestErrorHintKey,
} from "@/features/nodes/node-request-error"
import { copyText } from "@/lib/clipboard"
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
    onError: (error: Error) => {
      const code = classifyNodeRequestError(error)
      const payload = nodeRequestErrorClipboardText(error, { scope: "select", group: group?.tag })
      toast.error(formatNodeRequestErrorToast(error, t("nodes.selectFailed")), {
        description: t(nodeRequestErrorHintKey(code)),
        action: payload ? {
          label: t("nodes.copyRequestError"),
          onClick: () => {
            void copyText(payload).then(
              () => toast.success(t("nodes.requestErrorCopied")),
              () => toast.error(t("nodes.requestErrorCopyFailed")),
            )
          },
        } : undefined,
      })
    },
  })
  const delayMutation = useMutation({
    mutationFn: async () => (!group ? {} as DelayMap : measureGroupDelays(group.tag, members)),
    onSuccess: (next) => {
      setDelays(next)
      const summary = summarizeDelays(next)
      const message = formatDelayBatchMessage(summary, t)
      const tone = delayBatchToastTone(summary)
      const clipboard = delayBatchFailureClipboardText(summary)
      const options = summary.failed > 0 ? {
        description: t(delayErrorHintKey(summary.failedSamples[0]?.code)),
        action: clipboard ? {
          label: t("dashboard.copyProxyDelayError"),
          onClick: () => {
            void copyText(clipboard).then(
              () => toast.success(t("dashboard.proxyDelayErrorCopied")),
              () => toast.error(t("dashboard.proxyDelayErrorCopyFailed")),
            )
          },
        } : undefined,
      } : undefined
      if (tone === "error") toast.error(message, options)
      else if (tone === "warning") toast.warning(message, options)
      else toast.success(message)
    },
    onError: (error: Error) => {
      const payload = delayRequestErrorClipboardText(error, "proxy-delay")
      toast.error(formatDelayRequestErrorToast(error, t("dashboard.proxyDelayFailed")), {
        description: t(delayErrorHintKey(delayFailureFromError(error).code)),
        action: payload ? {
          label: t("dashboard.copyProxyDelayError"),
          onClick: () => {
            void copyText(payload).then(
              () => toast.success(t("dashboard.proxyDelayErrorCopied")),
              () => toast.error(t("dashboard.proxyDelayErrorCopyFailed")),
            )
          },
        } : undefined,
      })
    },
  })
  return {
    query, group, members, delays,
    select: (tag: string) => selectMutation.mutate(tag),
    selecting: selectMutation.isPending,
    probe: () => delayMutation.mutate(),
    probing: delayMutation.isPending,
  }
}
