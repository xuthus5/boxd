import { useMutation } from "@tanstack/react-query"
import { GaugeIcon, ListPlusIcon, ServerIcon } from "lucide-react"
import { useMemo, useState, type ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { DNSActionSummaryBar } from "@/features/policy/dns-action-summary"
import {
  dnsRuleFiltersActive,
  dnsServerFiltersActive,
  matchesDNSRule,
  matchesDNSRuleAction,
  matchesDNSServer,
  matchesDNSServerType,
  parseDNSSearchParams,
  summarizeDNSRuleActions,
  summarizeDNSServerTypes,
  toDNSSearchParams,
  type DNSSearchFilters,
} from "@/features/policy/dns-filter"
import { DNSTypeSummaryBar } from "@/features/policy/dns-type-summary"
import { toggleRuleInvert } from "@/features/policy/rule-invert"
import { DNSFakeIPCard, DNSGlobalCard } from "@/features/policy/dns-global-card"
import { DNSRuleCard } from "@/features/policy/dns-rule-card"
import { DNSRuleDialog } from "@/features/policy/dns-rule-dialog"
import { DNSServerCard } from "@/features/policy/dns-server-card"
import { DNSServerDialog } from "@/features/policy/dns-server-dialog"
import {
  dnsProbeBatchToastTone,
  dnsProbeInput,
  formatDNSProbeBatchMessage,
  mapDNSProbeBatchResults,
  summarizeDNSProbeResults,
} from "@/features/policy/dns-probe"
import { dnsRules, dnsServers, setDNSRules, setDNSServers } from "@/features/policy/dns-form-model"
import { cloneJsonObject, moveItem, type JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { api } from "@/lib/api/endpoints"
import type { DNSProbeResult } from "@/lib/api/types"

interface EditorSelection {
  kind: "server" | "rule"
  index: number | null
  item: JsonObject
}

function replaceOrAppend(items: readonly JsonObject[], index: number | null, item: JsonObject) {
  if (index === null) return [...items, item]
  return items.map((current, currentIndex) => currentIndex === index ? item : current)
}

function insertCopy(items: readonly JsonObject[], index: number) {
  return [...items.slice(0, index + 1), cloneJsonObject(items[index]), ...items.slice(index + 1)]
}

function EmptySection({ title, description, action, onAdd }: {
  title: string; description: string; action: string; onAdd: () => void
}) {
  return <Empty><EmptyHeader><EmptyMedia variant="icon"><ServerIcon /></EmptyMedia>
    <EmptyTitle>{title}</EmptyTitle><EmptyDescription>{description}</EmptyDescription></EmptyHeader>
    <EmptyContent><Button onClick={onAdd}><ListPlusIcon data-icon="inline-start" />{action}</Button></EmptyContent>
  </Empty>
}

function probeKey(item: JsonObject, index: number) {
  return typeof item.tag === "string" && item.tag ? item.tag : `idx:${index}`
}

function ServerSection({ object, onChange, onRulesChange, onEdit, onInstall }: {
  object: JsonObject; onChange: (object: JsonObject) => void; onEdit: (index: number | null) => void
  onRulesChange?: (object: JsonObject, metadata: never[]) => void; onInstall?: () => void
}) {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseDNSSearchParams(searchParams), [searchParams])
  const search = filters.servers ?? ""
  const typeFilter = filters.serverType
  const [probeResults, setProbeResults] = useState<Record<string, DNSProbeResult>>({})
  const servers = dnsServers(object)
  const normalized = search.trim().toLowerCase()
  const writeFilters = (next: DNSSearchFilters) => {
    setSearchParams(toDNSSearchParams(next), { replace: true })
  }
  const writeServersQuery = (value: string) => {
    writeFilters({
      servers: value,
      rules: filters.rules,
      serverType: typeFilter,
      ruleAction: filters.ruleAction,
    })
  }
  const visible = servers
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => matchesDNSServer(item, normalized) && matchesDNSServerType(item, typeFilter))
  const typeSummary = summarizeDNSServerTypes(servers, search)
  const facetsActive = dnsServerFiltersActive({ servers: search, serverType: typeFilter })
  const inputs = servers.flatMap((item) => {
    const input = dnsProbeInput(item)
    return input ? [input] : []
  })
  const batchMutation = useMutation({
    mutationFn: async () => api.runtime.probeDNSBatch(inputs, 6),
    onSuccess: (payload) => {
      const mapped = mapDNSProbeBatchResults(
        inputs,
        payload.results ?? [],
        (input, index) => (input.tag?.trim() ? input.tag.trim() : `idx:${index}`),
      )
      setProbeResults((prev) => ({ ...prev, ...mapped }))
      const summary = summarizeDNSProbeResults(payload.results)
      const message = formatDNSProbeBatchMessage(summary, t)
      const tone = dnsProbeBatchToastTone(summary)
      if (tone === "error") toast.error(message)
      else if (tone === "warning") toast.warning(message)
      else toast.success(message)
    },
    onError: (error: Error) => toast.error(error.message),
  })
  /* c8 ignore next */
  const update = (next: readonly JsonObject[]) => { const nextObject = setDNSServers(object, next); onChange(nextObject); onRulesChange?.(nextObject, []) }
  return <Card size="sm"><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.dns.serversTitle")}</CardTitle><CardDescription>{t("policy.dns.serversDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <div className="grid w-full grid-cols-2 gap-2 sm:flex sm:w-auto">
        <Button variant="outline" className="h-8 w-full sm:w-auto" disabled={!inputs.length || batchMutation.isPending} onClick={() => batchMutation.mutate()}>
          <GaugeIcon data-icon="inline-start" />{batchMutation.isPending ? t("policy.dns.probing") : t("policy.dns.probeAll")}
        </Button>
        <Button variant="outline" className="h-8 w-full sm:w-auto" onClick={onInstall}>{t("policy.installDNS")}</Button>
        <Button className="h-8 w-full sm:w-auto" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.dns.addServer")}</Button>
      </div>
    </CardAction></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">{servers.length === 0
      ? <EmptySection title={t("policy.dns.emptyServersTitle")} description={t("policy.dns.emptyServersDescription")}
        action={t("policy.dns.addServer")} onAdd={() => onEdit(null)} />
      : <>
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor="dns-servers-search">{t("policy.dns.searchServers")}</label>
            <Input id="dns-servers-search" value={search} onChange={(event) => writeServersQuery(event.target.value)} placeholder={t("policy.dns.searchServersPlaceholder")} className="h-8 sm:max-w-sm" aria-label={t("policy.dns.searchServers")} />
            {facetsActive ? <p className="text-sm text-muted-foreground">{t("policy.dns.searchCount", { shown: visible.length, total: servers.length })}</p> : null}
          </div>
          <DNSTypeSummaryBar summary={typeSummary} filters={filters} onChange={writeFilters} />
        </div>
        {visible.length === 0
          ? <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("policy.dns.noMatch")}</EmptyTitle>
              <EmptyDescription>{t("policy.dns.noMatchDescription")}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button type="button" variant="outline" onClick={() => writeFilters({ rules: filters.rules, ruleAction: filters.ruleAction })}>
                {t("proxy.clearSearch")}
              </Button>
            </EmptyContent>
          </Empty>
          : <div className="flex flex-col gap-2 sm:gap-3">{visible.map(({ item, index }) => {
            const key = probeKey(item, index)
            return <DNSServerCard key={index} item={item}
              probeResult={probeResults[key] ?? (typeof item.tag === "string" ? probeResults[item.tag] : undefined)}
              onProbeResult={(result) => setProbeResults((prev) => ({ ...prev, [key]: result, ...(result.tag ? { [result.tag]: result } : {}) }))}
              onEdit={() => onEdit(index)} onCopy={() => update(insertCopy(servers, index))}
              onDelete={() => update(servers.filter((_, itemIndex) => itemIndex !== index))} />
          })}</div>}
      </>}</CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.dns.serversCount", { count: servers.length })}</p></CardFooter></Card>
}

function RuleSection({ object, onChange, onRulesChange, onEdit }: {
  object: JsonObject; onChange: (object: JsonObject) => void; onEdit: (index: number | null) => void
  onRulesChange?: (object: JsonObject, metadata: never[]) => void
}) {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const filters = useMemo(() => parseDNSSearchParams(searchParams), [searchParams])
  const search = filters.rules ?? ""
  const actionFilter = filters.ruleAction
  const rules = dnsRules(object)
  const normalized = search.trim().toLowerCase()
  const writeFilters = (next: DNSSearchFilters) => {
    setSearchParams(toDNSSearchParams(next), { replace: true })
  }
  const writeRulesQuery = (value: string) => {
    writeFilters({
      servers: filters.servers,
      rules: value,
      serverType: filters.serverType,
      ruleAction: actionFilter,
    })
  }
  const visible = rules
    .map((item, index) => ({ item, index }))
    .filter(({ item }) => matchesDNSRule(item, normalized) && matchesDNSRuleAction(item, actionFilter))
  const actionSummary = summarizeDNSRuleActions(rules, search)
  const facetsActive = dnsRuleFiltersActive({ rules: search, ruleAction: actionFilter })
  /* c8 ignore next */
  const update = (next: readonly JsonObject[]) => { const nextObject = setDNSRules(object, next); onChange(nextObject); onRulesChange?.(nextObject, []) }
  return <Card size="sm"><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.dns.rulesTitle")}</CardTitle><CardDescription>{t("policy.dns.rulesDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <Button className="h-8 w-full sm:w-auto" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.dns.addRule")}</Button>
    </CardAction></CardHeader>
    <CardContent className="flex flex-col gap-2 sm:gap-3">{rules.length === 0
      ? <EmptySection title={t("policy.dns.emptyRulesTitle")} description={t("policy.dns.emptyRulesDescription")}
        action={t("policy.dns.addRule")} onAdd={() => onEdit(null)} />
      : <>
        <div className="flex flex-col gap-2">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
            <label className="sr-only" htmlFor="dns-rules-search">{t("policy.dns.searchRules")}</label>
            <Input id="dns-rules-search" value={search} onChange={(event) => writeRulesQuery(event.target.value)} placeholder={t("policy.dns.searchRulesPlaceholder")} className="h-8 sm:max-w-sm" aria-label={t("policy.dns.searchRules")} />
            {facetsActive ? <p className="text-sm text-muted-foreground">{t("policy.dns.searchCount", { shown: visible.length, total: rules.length })}</p> : null}
          </div>
          <DNSActionSummaryBar summary={actionSummary} filters={filters} onChange={writeFilters} />
        </div>
        {visible.length === 0
          ? <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("policy.dns.noMatch")}</EmptyTitle>
              <EmptyDescription>{t("policy.dns.noMatchDescription")}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button type="button" variant="outline" onClick={() => writeFilters({ servers: filters.servers, serverType: filters.serverType })}>
                {t("proxy.clearSearch")}
              </Button>
            </EmptyContent>
          </Empty>
          : <div className="flex flex-col gap-2 sm:gap-3">{visible.map(({ item, index }) => <DNSRuleCard key={index} index={index} item={item}
            first={index === 0} last={index === rules.length - 1} onEdit={() => onEdit(index)}
            onCopy={() => update(insertCopy(rules, index))}
            onMoveUp={() => update(moveItem(rules, index, -1))}
            onMoveDown={() => update(moveItem(rules, index, 1))}
            onDelete={() => update(rules.filter((_, itemIndex) => itemIndex !== index))}
            onToggleInvert={() => update(rules.map((rule, ruleIndex) => ruleIndex === index ? toggleRuleInvert(rule) : rule))} />)}</div>}
      </>}</CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.dns.rulesCount", { count: rules.length })}</p></CardFooter></Card>
}

/* c8 ignore start */
export function DNSVisualEditor(props: PolicyVisualEditorProps): ReactNode {
  const { t } = useTranslation()
  const { object, onChange } = props
  const [selection, setSelection] = useState<EditorSelection | null>(null)
  const editServer = (index: number | null) => setSelection({ kind: "server", index, item: index === null ? {} : dnsServers(object)[index] })
  const editRule = (index: number | null) => setSelection({ kind: "rule", index, item: index === null ? { action: "route" } : dnsRules(object)[index] })
  const saveSelection = (item: JsonObject) => {
    if (!selection) return
    const next = selection.kind === "server"
      ? setDNSServers(object, replaceOrAppend(dnsServers(object), selection.index, item))
      : setDNSRules(object, replaceOrAppend(dnsRules(object), selection.index, item))
    onChange(next)
    /* c8 ignore next */
    props.onRulesChange?.(next, [])
    setSelection(null)
  }
  const serverTags = dnsServers(object).flatMap((server) => typeof server.tag === "string" && server.tag ? [server.tag] : [])
  return <div className="flex min-w-0 flex-col gap-2 sm:gap-3"><DNSGlobalCard {...props} /><DNSFakeIPCard {...props} />
    <ServerSection object={object} onChange={onChange} onRulesChange={props.onRulesChange} onEdit={editServer} onInstall={props.onInstall} />
    <RuleSection object={object} onChange={onChange} onRulesChange={props.onRulesChange} onEdit={editRule} />
    {selection?.kind === "server" ? <DNSServerDialog key={`${selection.index}:${JSON.stringify(selection.item)}`} open
      item={selection.item} title={selection.index === null ? t("policy.dns.addServerTitle") : t("policy.dns.editServerTitle")}
      onOpenChange={(open) => { if (!open) setSelection(null) }} onSave={saveSelection} /> : null}
    {selection?.kind === "rule" ? <DNSRuleDialog key={`${selection.index}:${JSON.stringify(selection.item)}`} open
      item={selection.item} title={selection.index === null ? t("policy.dns.addRuleTitle") : t("policy.dns.editRuleTitle", { index: selection.index + 1 })} serverTags={serverTags}
      onOpenChange={(open) => { if (!open) setSelection(null) }} onSave={saveSelection} /> : null}
  </div>
}
/* c8 ignore stop */
