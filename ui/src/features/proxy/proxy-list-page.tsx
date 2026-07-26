import { PlusIcon, WandSparklesIcon } from "lucide-react"
import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"
import { ConfigSaveErrorAlert } from "@/features/config/config-save-error-alert"
import { useConfigPathReveal } from "@/features/config/use-config-path-reveal"
import { useConfigQuery, useSaveConfigMutation } from "@/features/config/config-hooks"
import { useConfigSaveError } from "@/features/config/use-config-save-error"
import { ProxyEditorDialog } from "@/features/proxy/proxy-editor-dialog"
import { InboundCards, OutboundCards } from "@/features/proxy/proxy-list-cards"
import { parseProxyItemPath } from "@/features/proxy/proxy-path"
import {
  filterProxyItems,
  parseProxySearchParams,
  proxyFiltersActive,
  summarizeProxyTypes,
  toProxySearchParams,
  type ProxyListFilters,
} from "@/features/proxy/proxy-filter"
import { ProxyTypeSummaryBar } from "@/features/proxy/proxy-type-summary"
import { api } from "@/lib/api/endpoints"
import type { JsonValue } from "@/lib/api/types"

type JsonObject = Record<string, JsonValue>
interface Editing { index: number; item: JsonObject }

function objects(value: JsonValue | undefined) {
  return Array.isArray(value)
    ? value.filter((item): item is JsonObject => Boolean(item && typeof item === "object" && !Array.isArray(item)))
    : []
}

export function ProxyListPage({ configKey, title, addLabel }: {
  configKey: "inbounds" | "outbounds"
  title: string
  addLabel: string
}) {
  const { t } = useTranslation()
  const query = useConfigQuery()
  const save = useSaveConfigMutation()
  const [editing, setEditing] = useState<Editing | null>(null)
  const [jumpPath, setJumpPath] = useState<string | null>(null)
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseProxySearchParams(searchParams), [searchParams])
  const search = filters.query ?? ""
  const { saveError, clearSaveError, reportError, reportRollback } = useConfigSaveError()
  const items = useMemo(() => objects(query.data?.[configKey]), [configKey, query.data])
  const openPath = useCallback((path: string) => {
    const target = parseProxyItemPath(path, configKey)
    if (!target || target.index < 0 || target.index >= items.length) {
      toast.message(t("config.pathNotFound", { path }))
      return false
    }
    setEditing({ index: target.index, item: items[target.index]! })
    setJumpPath(target.relativePath)
    return true
  }, [configKey, items, t])
  useConfigPathReveal((path) => openPath(path), {
    section: configKey,
    ready: !query.isLoading && !query.error,
  })
  if (query.isLoading) return <Skeleton className="h-64 w-full" />
  if (query.error) {
    return (
      <PageLoadErrorAlert
        error={query.error}
        scope="proxy"
        onRetry={() => { void query.refetch() }}
      />
    )
  }
  const typeFilter = filters.type
  const filteredIndexed = items
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => filterProxyItems([item], { query: search, type: typeFilter }).length > 0)
  const typeSummary = summarizeProxyTypes(items, search)
  const facetsActive = proxyFiltersActive({ query: search, type: typeFilter })
  const writeFilters = (next: ProxyListFilters) => {
    setSearchParams(toProxySearchParams(next), { replace: true })
  }
  const writeQuery = (queryValue: string) => {
    writeFilters({ query: queryValue, type: typeFilter })
  }
  const persist = async (nextItems: JsonObject[]) => {
    clearSaveError()
    try {
      const response = await save.mutateAsync({ ...query.data!, [configKey]: nextItems })
      if (response.status === "rolled_back") {
        reportRollback(response, t("proxy.rolledBack"))
        return false
      }
      toast.success(t("proxy.saved"))
      return true
    } catch (error) {
      reportError(error)
      return false
    }
  }
  const saveItem = async (item: JsonObject) => {
    const next = [...items]
    if (editing!.index < 0) next.push(item)
    else next[editing!.index] = item
    if (await persist(next)) {
      setEditing(null)
      setJumpPath(null)
    }
  }
  const installDefaults = () => {
    const action = configKey === "outbounds" ? api.config.installOutbounds() : api.config.installInbounds()
    const success = configKey === "outbounds" ? t("proxy.defaultsInstalled") : t("proxy.inboundDefaultsInstalled")
    clearSaveError()
    action.then(() => query.refetch()).then(() => toast.success(success)).catch((error: Error) => {
      reportError(error)
    })
  }
  return (
    <div className="flex flex-col gap-3 sm:gap-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold">{title}</h1>
        <div className="grid grid-cols-2 gap-2 sm:flex">
          <Button variant="outline" size="sm" className="h-8" onClick={installDefaults}>
            <WandSparklesIcon data-icon="inline-start" />
            {configKey === "outbounds" ? t("proxy.installDefaults") : t("proxy.installInboundDefaults")}
          </Button>
          <Button size="sm" className="h-8" onClick={() => setEditing({ index: -1, item: {} })}>
            <PlusIcon data-icon="inline-start" />{addLabel}
          </Button>
        </div>
      </div>
      <ConfigSaveErrorAlert error={saveError} onDismiss={clearSaveError} onJumpToPath={(path) => { void openPath(path) }} />
      {items.length > 0 ? (
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor={`proxy-search-${configKey}`}>{t("proxy.search")}</label>
            <Input id={`proxy-search-${configKey}`} value={search} onChange={(event) => writeQuery(event.target.value)} placeholder={t("proxy.searchPlaceholder")} className="h-8 sm:max-w-sm" aria-label={t("proxy.search")} />
            {facetsActive ? <p className="text-sm text-muted-foreground">{t("proxy.searchCount", { shown: filteredIndexed.length, total: items.length })}</p> : null}
          </div>
          <ProxyTypeSummaryBar summary={typeSummary} filters={filters} onChange={writeFilters} />
        </div>
      ) : null}
      {items.length > 0 ? (
        filteredIndexed.length > 0 ? (
          configKey === "inbounds" ? (
            <InboundCards
              items={filteredIndexed}
              busy={save.isPending}
              onEdit={(index) => setEditing({ index, item: items[index] })}
              onDelete={(index) => { void persist(items.filter((_, itemIndex) => itemIndex !== index)) }}
              onPatch={(index, item) => { const next = [...items]; next[index] = item; void persist(next) }}
            />
          ) : (
            <OutboundCards
              items={filteredIndexed}
              onEdit={(index) => setEditing({ index, item: items[index] })}
              onDelete={(index) => { void persist(items.filter((_, itemIndex) => itemIndex !== index)) }}
            />
          )
        ) : (
          <Card size="sm">
            <CardHeader className="gap-1.5">
              <CardTitle className="truncate">{t("proxy.noMatch")}</CardTitle>
              <CardDescription>{t("proxy.noMatchDescription")}</CardDescription>
            </CardHeader>
            <CardContent>
              <Button variant="outline" size="sm" className="h-8" onClick={() => writeFilters({})}>{t("proxy.clearSearch")}</Button>
            </CardContent>
          </Card>
        )
      ) : (
        <Card size="sm">
          <CardHeader className="gap-1.5">
            <CardTitle className="truncate">{title}{t("proxy.listSuffix")}</CardTitle>
            <CardDescription>{t("proxy.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <Empty>
              <EmptyHeader><EmptyTitle>{t("proxy.empty")}</EmptyTitle><EmptyDescription>{t("proxy.emptyDescription")}</EmptyDescription></EmptyHeader>
              <EmptyContent>
                <Button size="sm" className="h-8" onClick={() => setEditing({ index: -1, item: {} })}>{addLabel}</Button>
              </EmptyContent>
            </Empty>
          </CardContent>
        </Card>
      )}
      {editing ? (
        <ProxyEditorDialog
          key={`${editing.index}-${String(editing.item.tag)}`}
          title={editing.index < 0 ? addLabel : `${t("proxy.editPrefix")} ${String(editing.item.tag ?? "")}`}
          kind={configKey}
          item={editing.item}
          index={editing.index}
          onClose={() => { setEditing(null); setJumpPath(null) }}
          onSave={saveItem}
          jumpPath={jumpPath}
          onJumpPathHandled={() => setJumpPath(null)}
          reportError={reportError}
          clearSaveError={clearSaveError}
          saving={save.isPending}
        />
      ) : null}
    </div>
  )
}
