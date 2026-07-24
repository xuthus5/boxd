import { useQuery } from "@tanstack/react-query"
import { useId, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { NodeSection } from "@/features/nodes/node-section"
import { NodeStabilitySummaryBar } from "@/features/nodes/node-stability-summary"
import {
  filterAndSortNodes,
  nodeFiltersActive,
  parseNodeSearchParams,
  summarizeNodeStability,
  toNodeSearchParams,
  type NodeListFilters,
  type NodeSortKey,
  type NodeStabilityFilter,
} from "@/features/nodes/nodes-filter"
import { RuntimeGroupsCard } from "@/features/nodes/runtime-groups-card"
import { api } from "@/lib/api/endpoints"
import type { LatencyPoint, Outbound, TestResult } from "@/lib/api/types"
import { CardQueryError } from "@/features/common/card-query-error"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

function groupSubscriptions(nodes: Outbound[]) {
  const groups = new Map<string, Outbound[]>()
  for (const node of nodes) {
    if (node.source !== "subscription") continue
    const name = node.source_name || ""
    groups.set(name, [...(groups.get(name) ?? []), node])
  }
  return groups
}

function SubscriptionSections({
  groups,
  results,
  history,
  onBatchComplete,
}: {
  groups: Map<string, Outbound[]>
  results?: Record<string, Record<string, TestResult>>
  history?: Record<string, Record<string, LatencyPoint[]>>
  onBatchComplete?: () => void
}) {
  const { t } = useTranslation()
  const titleId = useId()
  return (
    <section aria-labelledby={titleId} className="flex flex-col gap-3">
      <div>
        <h2 id={titleId} className="text-lg font-medium">{t("nodes.subscriptionNodes")}</h2>
        <p className="text-sm text-muted-foreground">{t("nodes.subscriptionNodesDescription")}</p>
      </div>
      {groups.size
        ? <div className="grid gap-4">
          {Array.from(groups, ([name, nodes]) => (
            <NodeSection
              key={name || "subscription"}
              title={name || t("nodes.subscription")}
              description={t("nodes.nodeCount", { count: nodes.length })}
              nodes={nodes}
              results={results}
              history={history}
              onBatchComplete={onBatchComplete}
            />
          ))}
        </div>
        : <Empty><EmptyHeader><EmptyTitle>{t("nodes.empty")}</EmptyTitle><EmptyDescription>{t("nodes.emptyDescription")}</EmptyDescription></EmptyHeader></Empty>}
    </section>
  )
}

export function NodesPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseNodeSearchParams(searchParams), [searchParams])
  const query = filters.query ?? ""
  const stability = (filters.stability ?? "") as NodeStabilityFilter
  const sort = filters.sort ?? "name"
  const nodesQuery = useQuery({ queryKey: ["nodes"], queryFn: api.nodes.list })
  const resultsQuery = useQuery({ queryKey: ["nodes", "results"], queryFn: api.nodes.results })
  const historyQuery = useQuery({
    queryKey: ["nodes", "history"],
    queryFn: async () => {
      const payload = await api.nodes.testHistory()
      return (payload.history ?? {}) as Record<string, Record<string, LatencyPoint[]>>
    },
  })
  const history = historyQuery.data
  const filtered = useMemo(
    () => filterAndSortNodes(nodesQuery.data ?? [], { query, stability: stability || undefined, sort }, history),
    [history, nodesQuery.data, query, sort, stability],
  )
  const stabilitySummary = useMemo(
    () => summarizeNodeStability(nodesQuery.data ?? [], history, query),
    [history, nodesQuery.data, query],
  )
  const imported = filtered.filter((node) => node.source === "import")
  const subscriptions = groupSubscriptions(filtered)
  const facetsActive = nodeFiltersActive({ query, stability: stability || undefined })
  const listError = nodesQuery.error
  const auxError = resultsQuery.error || historyQuery.error
  const auxScope = resultsQuery.error ? "nodes-results" : historyQuery.error ? "nodes-history" : "nodes-aux"
  const stabilityOptions: { label: string; value: string }[] = [
    { label: t("nodes.filterAll"), value: "__all__" },
    { label: t("nodes.filterStable"), value: "stable" },
    { label: t("nodes.filterFair"), value: "fair" },
    { label: t("nodes.filterUnstable"), value: "unstable" },
    { label: t("nodes.filterFailed"), value: "failed" },
    { label: t("nodes.filterUnknown"), value: "unknown" },
  ]
  const sortOptions: { label: string; value: NodeSortKey }[] = [
    { label: t("nodes.sortByName"), value: "name" },
    { label: t("nodes.sortByStability"), value: "stability" },
    { label: t("nodes.sortByLatency"), value: "latency" },
  ]

  const writeFilters = (next: NodeListFilters) => {
    setSearchParams(toNodeSearchParams(next), { replace: true })
  }

  const onBatchComplete = () => {
    writeFilters({
      query: filters.query,
      stability: filters.stability,
      sort: "stability",
    })
  }

  if (nodesQuery.isLoading) return <Skeleton className="h-64 w-full" />
  if (listError) {
    return (
      <PageLoadErrorAlert
        error={listError}
        scope="nodes"
        onRetry={() => { void nodesQuery.refetch() }}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <h1 className="text-2xl font-semibold">{t("nodes.title")}</h1>
        <div className="grid w-full grid-cols-2 gap-2 sm:flex sm:max-w-3xl sm:flex-row sm:items-center sm:justify-end">
          <label className="sr-only" htmlFor="nodes-search">{t("nodes.search")}</label>
          <Input
            id="nodes-search"
            value={query}
            onChange={(event) => writeFilters({
              query: event.target.value,
              stability: filters.stability,
              sort: filters.sort,
            })}
            placeholder={t("nodes.searchPlaceholder")}
            className="col-span-2 h-8 sm:max-w-xs"
            aria-label={t("nodes.search")}
          />
          <Select
            items={stabilityOptions}
            value={stability || "__all__"}
            onValueChange={(value) => writeFilters({
              query: filters.query,
              stability: String(value) === "__all__" ? undefined : String(value) as NodeStabilityFilter,
              sort: filters.sort,
            })}
          >
            <SelectTrigger aria-label={t("nodes.filterStability")} className="h-8 w-full sm:w-36">
              <SelectValue placeholder={t("nodes.filterStability")} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {stabilityOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Select
            items={sortOptions}
            value={sort}
            onValueChange={(value) => writeFilters({
              query: filters.query,
              stability: filters.stability,
              sort: String(value) as NodeSortKey,
            })}
          >
            <SelectTrigger aria-label={t("nodes.sortNodes")} className="h-8 w-full sm:w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {sortOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {facetsActive ? (
            <Button
              variant="ghost"
              size="sm"
              className="col-span-2 h-8 sm:col-span-1"
              onClick={() => writeFilters({ sort: filters.sort })}
            >
              {t("nodes.clearFilters")}
            </Button>
          ) : null}
        </div>
      </div>
      <NodeStabilitySummaryBar summary={stabilitySummary} filters={filters} onChange={writeFilters} />
      {auxError ? (
        <CardQueryError
          error={auxError}
          scope={auxScope}
          onRetry={() => {
            if (resultsQuery.error) void resultsQuery.refetch()
            if (historyQuery.error) void historyQuery.refetch()
          }}
        />
      ) : null}
      <RuntimeGroupsCard />
      {filtered.length === 0 && facetsActive
        ? <Empty>
          <EmptyHeader>
            <EmptyTitle>{t("nodes.noMatch")}</EmptyTitle>
            <EmptyDescription>{t("nodes.noMatchDescription")}</EmptyDescription>
          </EmptyHeader>
          <EmptyContent>
            <Button variant="outline" onClick={() => writeFilters({ sort: filters.sort })}>
              {t("nodes.clearFilters")}
            </Button>
          </EmptyContent>
        </Empty>
        : <>
          <NodeSection title={t("nodes.allNodes")} description={t("nodes.allNodesDescription")} nodes={filtered} results={resultsQuery.data} history={history} onBatchComplete={onBatchComplete} />
          <SubscriptionSections groups={subscriptions} results={resultsQuery.data} history={history} onBatchComplete={onBatchComplete} />
          <NodeSection title={t("nodes.importedNodes")} description={t("nodes.importedNodesDescription")} nodes={imported} results={resultsQuery.data} history={history} onBatchComplete={onBatchComplete} />
        </>}
    </div>
  )
}
