import { useQuery } from "@tanstack/react-query"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { useConfigQuery } from "@/features/config/config-hooks"
import { useConfigRestore } from "@/features/dashboard/use-config-restore"
import { filterConfigHistory, type ConfigHistoryFilter } from "@/features/advanced/config-history-filter"
import { ConfigHistoryView } from "@/features/advanced/config-history-view"
import { api } from "@/lib/api/endpoints"
import type { ConfigApplyEvent } from "@/lib/api/types"
import { Skeleton } from "@/components/ui/skeleton"

const emptyHistoryEvents: ConfigApplyEvent[] = []

function useConfigHistoryData() {
  const history = useQuery({
    queryKey: ["config", "apply-history"],
    queryFn: api.config.applyHistory,
    refetchInterval: 15000,
  })
  return { history, currentConfig: useConfigQuery(), events: history.data?.events ?? emptyHistoryEvents }
}

export function ConfigHistoryPage() {
  const { i18n } = useTranslation()
  const [query, setQuery] = useState("")
  const [filter, setFilter] = useState<ConfigHistoryFilter>("all")
  const { restore, restoringID } = useConfigRestore()
  const { history, currentConfig, events } = useConfigHistoryData()
  const filtered = useMemo(() => filterConfigHistory(events, query, filter), [events, filter, query])
  const hasFilters = Boolean(query.trim()) || filter !== "all"
  const locale = i18n.language?.startsWith("en") ? "en-US" : "zh-CN"
  const clearFilters = () => {
    setQuery("")
    setFilter("all")
  }

  if (history.isLoading) return <Skeleton className="h-64 w-full" />
  if (history.error) {
    return <PageLoadErrorAlert error={history.error} titleKey="configHistory.loadFailed" scope="config-history" path="/api/config/apply-history" onRetry={() => { void history.refetch() }} />
  }
  return (
    <ConfigHistoryView
      isFetching={history.isFetching}
      onRefresh={() => { void history.refetch() }}
      events={events}
      filtered={filtered}
      query={query}
      filter={filter}
      now={history.dataUpdatedAt}
      locale={locale}
      currentConfig={currentConfig.data}
      currentConfigLoading={currentConfig.isFetching}
      restoringID={restoringID}
      onRestore={restore}
      hasFilters={hasFilters}
      onQueryChange={setQuery}
      onFilterChange={setFilter}
      onClear={clearFilters}
    />
  )
}
