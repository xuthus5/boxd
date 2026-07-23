import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  closeErrorClipboardText,
  closeErrorHintKey,
  classifyCloseError,
  filterSuppressedConnections,
  formatCloseErrorToast,
  formatClosedScopeMessage,
  groupClosingKey,
  isBenignCloseMiss,
  isBulkClosing,
  liveConnectionIdSet,
  pruneSuppressedIds,
  suppressConnectionIds,
  type ClosingTarget,
} from "@/features/observability/connection-close"
import { copyText } from "@/features/proxy/copy-tag-button"
import { filterConnectionsByGroup } from "@/features/observability/connection-stats"
import { api } from "@/lib/api/endpoints"
import type { Connection } from "@/lib/api/types"

export function useConnectionCloseActions(liveConnections: Connection[]) {
  const { t } = useTranslation()
  const [closingId, setClosingId] = useState<ClosingTarget>(null)
  const [suppressedIds, setSuppressedIds] = useState<Set<string>>(() => new Set())
  const liveIds = useMemo(() => liveConnectionIdSet(liveConnections), [liveConnections])
  const activeSuppressed = useMemo(
    () => pruneSuppressedIds(suppressedIds, liveIds),
    [suppressedIds, liveIds],
  )
  const connections = useMemo(
    () => filterSuppressedConnections(liveConnections, activeSuppressed),
    [liveConnections, activeSuppressed],
  )
  const bulkBusy = isBulkClosing(closingId)

  const markSuppressed = (ids: Array<string | number>) => {
    if (!ids.length) return
    setSuppressedIds((current) => suppressConnectionIds(current, ids))
  }

  const restoreSuppressed = (ids: Array<string | number>) => {
    setSuppressedIds((current) => {
      const next = new Set(current)
      for (const id of ids) next.delete(String(id))
      return next
    })
  }

  const reportCloseError = (
    error: unknown,
    options: { scope: "one" | "all" | "filtered" | "group"; target?: string; group?: string; count?: number },
  ) => {
    const message = formatCloseErrorToast(error, t, options)
    const code = classifyCloseError(error)
    const payload = closeErrorClipboardText(error, options)
    toast.error(message, {
      description: t(closeErrorHintKey(code)),
      action: {
        label: t("observability.copyCloseError"),
        onClick: () => {
          void copyText(payload).then(
            () => toast.success(t("observability.closeErrorCopied")),
            () => toast.error(t("observability.closeErrorCopyFailed")),
          )
        },
      },
    })
  }

  const closeOne = async (id: string) => {
    const target = connections.find((item) => String(item.id) === id)?.target
    setClosingId(id)
    markSuppressed([id])
    try {
      await api.stats.closeConnection(id)
      toast.success(formatClosedScopeMessage("one", t, { target }))
    } catch (error) {
      if (isBenignCloseMiss(error)) toast.success(t("observability.closedAlready"))
      else {
        restoreSuppressed([id])
        reportCloseError(error, { scope: "one", target, count: 1 })
      }
    } finally {
      setClosingId(null)
    }
  }

  const closeAll = async () => {
    const ids = connections.map((item) => item.id)
    setClosingId("all")
    markSuppressed(ids)
    try {
      const result = await api.stats.closeAll()
      toast.success(formatClosedScopeMessage("all", t, { count: result.closed }))
    } catch (error) {
      restoreSuppressed(ids)
      reportCloseError(error, { scope: "all", count: ids.length })
    } finally {
      setClosingId(null)
    }
  }

  const closeGroup = async (
    field: "outbound" | "rule" | "process",
    key: string,
    scope: Connection[],
  ) => {
    if (key === "—") return
    const groupIds = filterConnectionsByGroup(scope, field, key).map((item) => item.id)
    setClosingId(groupClosingKey(field, key))
    markSuppressed(groupIds)
    try {
      const result = await api.stats.closeAll({ [field]: key })
      toast.success(formatClosedScopeMessage("group", t, { group: key, count: result.closed }))
    } catch (error) {
      restoreSuppressed(groupIds)
      reportCloseError(error, { scope: "group", group: key, count: groupIds.length })
    } finally {
      setClosingId(null)
    }
  }

  const closeFiltered = async (scope: Connection[]) => {
    const ids = scope.map((item) => item.id)
    setClosingId("filtered")
    markSuppressed(ids)
    try {
      const result = await api.stats.closeAll({ ids })
      toast.success(formatClosedScopeMessage("filtered", t, { count: result.closed }))
    } catch (error) {
      restoreSuppressed(ids)
      reportCloseError(error, { scope: "filtered", count: ids.length })
    } finally {
      setClosingId(null)
    }
  }

  return { connections, closingId, bulkBusy, closeOne, closeAll, closeGroup, closeFiltered }
}
