import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  filterSuppressedConnections,
  formatClosedScopeMessage,
  groupClosingKey,
  isBenignCloseMiss,
  isBulkClosing,
  liveConnectionIdSet,
  pruneSuppressedIds,
  suppressConnectionIds,
  type ClosingTarget,
} from "@/features/observability/connection-close"
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
        toast.error(error instanceof Error ? error.message : String(error))
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
      toast.error(error instanceof Error ? error.message : String(error))
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
      toast.error(error instanceof Error ? error.message : String(error))
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
      toast.error(error instanceof Error ? error.message : String(error))
    } finally {
      setClosingId(null)
    }
  }

  return { connections, closingId, bulkBusy, closeOne, closeAll, closeGroup, closeFiltered }
}
