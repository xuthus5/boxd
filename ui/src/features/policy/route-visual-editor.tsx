import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useMemo, useState } from "react"
import { ListPlusIcon, RouteIcon } from "lucide-react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { RouteGlobalCard } from "@/features/policy/route-global-card"
import { cloneJsonObject, moveItem, type JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { RouteRuleCard } from "@/features/policy/route-rule-card"
import { RouteActionSummaryBar } from "@/features/policy/route-action-summary"
import {
  matchesRouteAction,
  matchesRouteRule,
  parseRouteSearchParams,
  routeFiltersActive,
  summarizeRouteActions,
  toRouteSearchParams,
  type RouteSearchFilters,
} from "@/features/policy/route-rule-filter"
import { toggleRuleInvert } from "@/features/policy/rule-invert"
import { RouteRuleSetCard } from "@/features/policy/route-rule-set-card"
import {
  formatRuleSetRequestErrorToast,
  formatRuleSetUpdateMessage,
  ruleSetBatchFailureClipboardText,
  ruleSetErrorHintKey,
  ruleSetRequestErrorClipboardText,
  classifyRuleSetRequestError,
  ruleSetUpdateToastTone,
  summarizeRuleSetUpdate,
} from "@/features/policy/ruleset-update-error"
import { copyText } from "@/features/proxy/copy-tag-button"
import { RouteVisualDialogs, type RouteEditorSelection } from "@/features/policy/route-visual-dialogs"
import { routeRuleSets, routeRules, setRouteRuleSets, setRouteRules } from "@/features/policy/route-form-model"
import { usePolicyVisualPathJump } from "@/features/policy/use-policy-visual-path-jump"
import type { PolicyDialogSelection } from "@/features/policy/policy-path"
import { api } from "@/lib/api/endpoints"
import type { JsonValue, RouteRuleMetadata, RuleSetStatusItem, RuleSetUpdateResult } from "@/lib/api/types"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

interface RouteVisualEditorProps extends PolicyVisualEditorProps {
  outbounds?: JsonValue
  metadata?: RouteRuleMetadata[]
  metadataLoading?: boolean
  metadataError?: string
  onMetadataChange?: (metadata: RouteRuleMetadata[]) => void
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
}

const emptyMetadata = (): RouteRuleMetadata => ({ name: "", description: "" })

function alignedMetadata(rules: readonly JsonObject[], metadata: readonly RouteRuleMetadata[]) {
  return rules.map((_, index) => metadata[index] ?? emptyMetadata())
}

function EmptySection({ title, description, action, onAdd }: {
  title: string; description: string; action: string; onAdd: () => void
}) {
  return <Empty><EmptyHeader><EmptyMedia variant="icon"><RouteIcon /></EmptyMedia>
    <EmptyTitle>{title}</EmptyTitle><EmptyDescription>{description}</EmptyDescription></EmptyHeader>
    <EmptyContent><Button onClick={onAdd}><ListPlusIcon data-icon="inline-start" />{action}</Button></EmptyContent>
  </Empty>
}

function replaceOrAppend<T>(items: readonly T[], index: number | null, item: T) {
  if (index === null) return [...items, item]
  return items.map((current, currentIndex) => currentIndex === index ? item : current)
}

function insertMetadataCopy(items: readonly RouteRuleMetadata[], index: number) {
  return [...items.slice(0, index + 1), { ...items[index] }, ...items.slice(index + 1)]
}

function insertCopy(items: readonly JsonObject[], index: number) {
  return [...items.slice(0, index + 1), cloneJsonObject(items[index]), ...items.slice(index + 1)]
}

function RuleSection({ object, metadata, metadataLoading, metadataError, onChange, onMetadataChange, onRulesChange, onEdit, onInstall }: {
  object: JsonObject; metadata: RouteRuleMetadata[]; onChange: (object: JsonObject) => void
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
  metadataLoading?: boolean; metadataError?: string
  onMetadataChange: (metadata: RouteRuleMetadata[]) => void; onEdit: (index: number | null) => void
  onInstall?: () => void
}) {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseRouteSearchParams(searchParams), [searchParams])
  const search = filters.query ?? ""
  const actionFilter = filters.action
  const rules = routeRules(object)
  const normalized = search.trim().toLowerCase()
  const writeFilters = (next: RouteSearchFilters) => {
    setSearchParams(toRouteSearchParams(next), { replace: true })
  }
  const writeQuery = (value: string) => {
    writeFilters({ query: value, action: actionFilter })
  }
  const indexed = rules.map((item, index) => ({ item, index, meta: metadata[index] }))
  const visible = indexed.filter(({ item, meta }) => (
    matchesRouteRule(item, meta, normalized) && matchesRouteAction(item, actionFilter)
  ))
  const actionSummary = summarizeRouteActions(rules, search, metadata)
  const facetsActive = routeFiltersActive({ query: search, action: actionFilter })
  const update = (next: readonly JsonObject[]) => onChange(setRouteRules(object, next))
  const updateBoth = (nextRules: readonly JsonObject[], nextMetadata: RouteRuleMetadata[]) => {
    const nextObject = setRouteRules(object, nextRules)
    update(nextRules); onMetadataChange(nextMetadata); onRulesChange?.(nextObject, nextMetadata)
  }
  return <Card size="sm"><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.route.rulesTitle")}</CardTitle><CardDescription>{t("policy.route.rulesDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <div className="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto"><Button variant="outline" size="sm" className="h-8" onClick={onInstall}>{t("policy.installRoute")}</Button><Button size="sm" className="h-8" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.route.addRule")}</Button></div>
    </CardAction></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">{metadataLoading ? <Skeleton className="h-24 w-full" /> : metadataError
      ? <PageLoadErrorAlert error={metadataError} scope="route-metadata" />
      : rules.length === 0
      ? <EmptySection title={t("policy.route.emptyRulesTitle")} description={t("policy.route.emptyRulesDescription")}
        action={t("policy.route.addRule")} onAdd={() => onEdit(null)} />
      : <>
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor="route-rules-search">{t("policy.route.searchRules")}</label>
            <Input id="route-rules-search" value={search} onChange={(event) => writeQuery(event.target.value)} placeholder={t("policy.route.searchRulesPlaceholder")} className="h-8 sm:max-w-sm" aria-label={t("policy.route.searchRules")} />
            {facetsActive ? <p className="text-sm text-muted-foreground">{t("policy.route.searchRulesCount", { shown: visible.length, total: rules.length })}</p> : null}
          </div>
          <RouteActionSummaryBar summary={actionSummary} filters={filters} onChange={writeFilters} />
        </div>
        {visible.length === 0
          ? <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("policy.route.noMatchRules")}</EmptyTitle>
              <EmptyDescription>{t("policy.route.noMatchRulesDescription")}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button type="button" variant="outline" onClick={() => writeFilters({})}>
                {t("proxy.clearSearch")}
              </Button>
            </EmptyContent>
          </Empty>
          : <div className="flex flex-col gap-2 sm:gap-3">{visible.map(({ item, index }) => <RouteRuleCard key={index} index={index} item={item} metadata={metadata[index]}
            first={index === 0} last={index === rules.length - 1} onEdit={() => onEdit(index)}
            onCopy={() => updateBoth(insertCopy(rules, index), insertMetadataCopy(metadata, index))}
            onMoveUp={() => updateBoth(moveItem(rules, index, -1), moveItem(metadata, index, -1))}
            onMoveDown={() => updateBoth(moveItem(rules, index, 1), moveItem(metadata, index, 1))}
            onDelete={() => updateBoth(rules.filter((_, itemIndex) => itemIndex !== index), metadata.filter((_, itemIndex) => itemIndex !== index))}
            onToggleInvert={() => updateBoth(rules.map((rule, ruleIndex) => ruleIndex === index ? toggleRuleInvert(rule) : rule), metadata)} />)}</div>}
      </>}
    </CardContent><CardFooter><p className="text-muted-foreground">{t("policy.route.rulesCount", { count: rules.length })}</p></CardFooter></Card>
}

function RuleSetSection({ object, onChange, onRulesChange, onEdit }: {
  object: JsonObject; onChange: (object: JsonObject) => void; onEdit: (index: number | null) => void
  onRulesChange?: (object: JsonObject, metadata: RouteRuleMetadata[]) => void
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const ruleSets = routeRuleSets(object)
  const statusQuery = useQuery({ queryKey: ["rule-sets", "status"], queryFn: api.config.ruleSetsStatus })
  const statusItems = Array.isArray(statusQuery.data) ? statusQuery.data : []
  const statusByTag = new Map(statusItems.map((item: RuleSetStatusItem) => [item.tag, item]))
  const [pendingTag, setPendingTag] = useState<string | null>(null)
  const [lastUpdateByTag, setLastUpdateByTag] = useState<Record<string, RuleSetUpdateResult>>({})
  const updateMutation = useMutation({
    mutationFn: (input?: { tags?: string[]; types?: string[] }) => api.config.updateRuleSets(input),
    onSuccess: async (envelope) => {
      const summary = summarizeRuleSetUpdate(envelope.data)
      const message = formatRuleSetUpdateMessage(summary, t)
      const tone = ruleSetUpdateToastTone(summary)
      const payload = ruleSetBatchFailureClipboardText(summary)
      const options = summary.failed > 0 ? {
        description: summary.failedSamples[0]
          ? t(ruleSetErrorHintKey(summary.failedSamples[0].code))
          : undefined,
        action: payload ? {
          label: t("policy.route.copyRuleSetError"),
          onClick: () => {
            void copyText(payload).then(
              () => toast.success(t("policy.route.ruleSetErrorCopied")),
              () => toast.error(t("policy.route.ruleSetErrorCopyFailed")),
            )
          },
        } : undefined,
      } : undefined
      if (tone === "error") toast.error(message, options)
      else if (tone === "warning") toast.warning(message, options)
      else toast.success(message)
      const next: Record<string, RuleSetUpdateResult> = {}
      for (const item of envelope.data?.results ?? []) {
        if (item.tag) next[item.tag] = item
      }
      if (Object.keys(next).length) setLastUpdateByTag((prev) => ({ ...prev, ...next }))
      await queryClient.invalidateQueries({ queryKey: ["rule-sets", "status"] })
      await queryClient.invalidateQueries({ queryKey: ["config"] })
    },
    onError: (error: Error) => {
      const code = classifyRuleSetRequestError(error)
      const payload = ruleSetRequestErrorClipboardText(error)
      toast.error(formatRuleSetRequestErrorToast(error, t, t("policy.route.ruleSetUpdateFailed", { failed: 1 })), {
        description: t(ruleSetErrorHintKey(code)),
        action: payload ? {
          label: t("policy.route.copyRuleSetError"),
          onClick: () => {
            void copyText(payload).then(
              () => toast.success(t("policy.route.ruleSetErrorCopied")),
              () => toast.error(t("policy.route.ruleSetErrorCopyFailed")),
            )
          },
        } : undefined,
      })
    },
    onSettled: () => setPendingTag(null),
  })
  const update = (next: readonly JsonObject[]) => { const nextObject = setRouteRuleSets(object, next); onChange(nextObject); onRulesChange?.(nextObject, []) }
  const updatableCount = ruleSets.filter((item) => {
    const tag = typeof item.tag === "string" ? item.tag : ""
    return statusByTag.get(tag)?.updatable
  }).length
  return <Card size="sm"><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.route.ruleSetsTitle")}</CardTitle><CardDescription>{t("policy.route.ruleSetsDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
        <Button variant="outline" className="w-full sm:w-auto" disabled={updateMutation.isPending || updatableCount === 0}
          onClick={() => { setPendingTag("*"); updateMutation.mutate({}) }}>{t("policy.route.updateAllRuleSets")}</Button>
        <Button className="w-full sm:w-auto" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.route.addRuleSet")}</Button>
      </div>
    </CardAction></CardHeader>
    <CardContent>{ruleSets.length === 0
      ? <EmptySection title={t("policy.route.emptyRuleSetsTitle")} description={t("policy.route.emptyRuleSetsDescription")}
        action={t("policy.route.addRuleSet")} onAdd={() => onEdit(null)} />
      : <div className="flex flex-col gap-2 sm:gap-3">{ruleSets.map((item, index) => {
        const tag = typeof item.tag === "string" ? item.tag : ""
        const status = statusByTag.get(tag)
        return <RouteRuleSetCard key={index} item={item} status={status} lastUpdate={tag ? lastUpdateByTag[tag] : undefined}
          updating={updateMutation.isPending && (pendingTag === tag || pendingTag === "*")}
          onEdit={() => onEdit(index)} onCopy={() => update(insertCopy(ruleSets, index))}
          onDelete={() => update(ruleSets.filter((_, itemIndex) => itemIndex !== index))}
          onUpdate={status?.updatable ? () => { setPendingTag(tag); updateMutation.mutate({ tags: [tag] }) } : undefined} />
      })}</div>}
    </CardContent><CardFooter><p className="text-muted-foreground">{t("policy.route.ruleSetsCount", { count: ruleSets.length })}</p></CardFooter></Card>
}

export function RouteVisualEditor(props: RouteVisualEditorProps): React.ReactNode {
  const { object, onChange, onMetadataChange = () => undefined, onRulesChange, jumpPath, onJumpPathHandled } = props
  const metadata = alignedMetadata(routeRules(object), props.metadata ?? [])
  const [selection, setSelection] = useState<RouteEditorSelection | null>(null)
  const lists = useMemo(() => ({
    rules: routeRules(object),
    ruleSets: routeRuleSets(object),
    metadata,
  }), [metadata, object])
  const selectFromPath = useCallback((next: PolicyDialogSelection) => {
    if (next.kind !== "rule" && next.kind !== "rule-set") return
    setSelection({
      kind: next.kind,
      index: next.index,
      item: next.item,
      metadata: next.kind === "rule" ? next.metadata : undefined,
      jumpPath: next.jumpPath,
    })
  }, [])
  usePolicyVisualPathJump({
    section: "route",
    jumpPath,
    onJumpPathHandled,
    lists,
    onSelect: selectFromPath,
  })
  const editRule = (index: number | null) => setSelection({ kind: "rule", index, item: index === null ? { action: "route" } : routeRules(object)[index], metadata: index === null ? emptyMetadata() : metadata[index] })
  const editRuleSet = (index: number | null) => setSelection({ kind: "rule-set", index, item: index === null ? { type: "inline" } : routeRuleSets(object)[index] })
  const saveSelection = (item: JsonObject, nextMetadata?: RouteRuleMetadata) => {
    if (!selection) return
    const next = selection.kind === "rule"
      ? setRouteRules(object, replaceOrAppend(routeRules(object), selection.index, item))
      : setRouteRuleSets(object, replaceOrAppend(routeRuleSets(object), selection.index, item))
    if (selection.kind === "rule") onMetadataChange(replaceOrAppend(metadata, selection.index, nextMetadata ?? emptyMetadata()))
    onChange(next)
    onRulesChange?.(next, selection.kind === "rule" ? replaceOrAppend(metadata, selection.index, nextMetadata ?? emptyMetadata()) : metadata)
    setSelection(null)
  }
  return <div className="flex min-w-0 flex-col gap-2 sm:gap-3">
    <RouteGlobalCard {...props} outbounds={props.outbounds} />
    <RuleSection object={object} metadata={metadata} metadataLoading={props.metadataLoading} metadataError={props.metadataError}
      onChange={onChange} onMetadataChange={onMetadataChange} onRulesChange={onRulesChange} onEdit={editRule} onInstall={props.onInstall} />
    <RuleSetSection object={object} onChange={onChange} onRulesChange={onRulesChange} onEdit={editRuleSet} />
    <RouteVisualDialogs selection={selection} onClose={() => setSelection(null)}
      onClearJumpPath={() => setSelection((c) => c ? { ...c, jumpPath: undefined } : c)} onSave={saveSelection} />
  </div>
}
