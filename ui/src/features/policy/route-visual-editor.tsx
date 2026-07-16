import { useState } from "react"
import { ListPlusIcon, RouteIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { RouteGlobalCard } from "@/features/policy/route-global-card"
import { cloneJsonObject, moveItem, type JsonObject } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { RouteRuleCard } from "@/features/policy/route-rule-card"
import { RouteRuleDialog } from "@/features/policy/route-rule-dialog"
import { RouteRuleSetCard } from "@/features/policy/route-rule-set-card"
import { RouteRuleSetDialog } from "@/features/policy/route-rule-set-dialog"
import { routeRuleSets, routeRules, setRouteRuleSets, setRouteRules } from "@/features/policy/route-form-model"
import type { JsonValue, RouteRuleMetadata } from "@/lib/api/types"

interface EditorSelection {
  kind: "rule" | "rule-set"
  index: number | null
  item: JsonObject
  metadata?: RouteRuleMetadata
}

interface RouteVisualEditorProps extends PolicyVisualEditorProps {
  outbounds?: JsonValue
  metadata?: RouteRuleMetadata[]
  metadataLoading?: boolean
  metadataError?: string
  onMetadataChange?: (metadata: RouteRuleMetadata[]) => void
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

function RuleSection({ object, metadata, metadataLoading, metadataError, onChange, onMetadataChange, onEdit }: {
  object: JsonObject; metadata: RouteRuleMetadata[]; onChange: (object: JsonObject) => void
  metadataLoading?: boolean; metadataError?: string
  onMetadataChange: (metadata: RouteRuleMetadata[]) => void; onEdit: (index: number | null) => void
}) {
  const { t } = useTranslation()
  const rules = routeRules(object)
  const update = (next: readonly JsonObject[]) => onChange(setRouteRules(object, next))
  const updateBoth = (nextRules: readonly JsonObject[], nextMetadata: RouteRuleMetadata[]) => {
    update(nextRules); onMetadataChange(nextMetadata)
  }
  return <Card><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.route.rulesTitle")}</CardTitle><CardDescription>{t("policy.route.rulesDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <Button className="w-full sm:w-auto" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.route.addRule")}</Button>
    </CardAction></CardHeader>
    <CardContent>{metadataLoading ? <Skeleton className="h-24 w-full" /> : metadataError
      ? <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{metadataError}</AlertDescription></Alert>
      : rules.length === 0
      ? <EmptySection title={t("policy.route.emptyRulesTitle")} description={t("policy.route.emptyRulesDescription")}
        action={t("policy.route.addRule")} onAdd={() => onEdit(null)} />
      : <div className="flex flex-col gap-3">{rules.map((item, index) => <RouteRuleCard key={index} index={index} item={item} metadata={metadata[index]}
        first={index === 0} last={index === rules.length - 1} onEdit={() => onEdit(index)}
        onCopy={() => updateBoth(insertCopy(rules, index), insertMetadataCopy(metadata, index))}
        onMoveUp={() => updateBoth(moveItem(rules, index, -1), moveItem(metadata, index, -1))}
        onMoveDown={() => updateBoth(moveItem(rules, index, 1), moveItem(metadata, index, 1))}
        onDelete={() => updateBoth(rules.filter((_, itemIndex) => itemIndex !== index), metadata.filter((_, itemIndex) => itemIndex !== index))} />)}</div>}
    </CardContent><CardFooter><p className="text-muted-foreground">{t("policy.route.rulesCount", { count: rules.length })}</p></CardFooter></Card>
}

function RuleSetSection({ object, onChange, onEdit }: {
  object: JsonObject; onChange: (object: JsonObject) => void; onEdit: (index: number | null) => void
}) {
  const { t } = useTranslation()
  const ruleSets = routeRuleSets(object)
  const update = (next: readonly JsonObject[]) => onChange(setRouteRuleSets(object, next))
  return <Card><CardHeader className="min-w-0 grid-cols-1 has-data-[slot=card-action]:grid-cols-1 sm:has-data-[slot=card-action]:grid-cols-[1fr_auto]">
    <CardTitle>{t("policy.route.ruleSetsTitle")}</CardTitle><CardDescription>{t("policy.route.ruleSetsDescription")}</CardDescription>
    <CardAction className="col-start-1 row-start-auto w-full justify-self-start sm:col-start-2 sm:row-start-1 sm:w-auto sm:justify-self-end">
      <Button className="w-full sm:w-auto" onClick={() => onEdit(null)}><ListPlusIcon data-icon="inline-start" />{t("policy.route.addRuleSet")}</Button>
    </CardAction></CardHeader>
    <CardContent>{ruleSets.length === 0
      ? <EmptySection title={t("policy.route.emptyRuleSetsTitle")} description={t("policy.route.emptyRuleSetsDescription")}
        action={t("policy.route.addRuleSet")} onAdd={() => onEdit(null)} />
      : <div className="flex flex-col gap-3">{ruleSets.map((item, index) => <RouteRuleSetCard key={index} item={item}
        onEdit={() => onEdit(index)} onCopy={() => update(insertCopy(ruleSets, index))}
        onDelete={() => update(ruleSets.filter((_, itemIndex) => itemIndex !== index))} />)}</div>}
    </CardContent><CardFooter><p className="text-muted-foreground">{t("policy.route.ruleSetsCount", { count: ruleSets.length })}</p></CardFooter></Card>
}

export function RouteVisualEditor(props: RouteVisualEditorProps): React.ReactNode {
  const { t } = useTranslation()
  const { object, onChange, onMetadataChange = () => undefined } = props
  const metadata = alignedMetadata(routeRules(object), props.metadata ?? [])
  const [selection, setSelection] = useState<EditorSelection | null>(null)
  const editRule = (index: number | null) => setSelection({ kind: "rule", index, item: index === null ? { action: "route" } : routeRules(object)[index], metadata: index === null ? emptyMetadata() : metadata[index] })
  const editRuleSet = (index: number | null) => setSelection({ kind: "rule-set", index, item: index === null ? { type: "inline" } : routeRuleSets(object)[index] })
  const saveSelection = (item: JsonObject, nextMetadata?: RouteRuleMetadata) => {
    if (!selection) return
    const next = selection.kind === "rule"
      ? setRouteRules(object, replaceOrAppend(routeRules(object), selection.index, item))
      : setRouteRuleSets(object, replaceOrAppend(routeRuleSets(object), selection.index, item))
    if (selection.kind === "rule") onMetadataChange(replaceOrAppend(metadata, selection.index, nextMetadata ?? emptyMetadata()))
    onChange(next)
    setSelection(null)
  }
  return <div className="flex min-w-0 flex-col gap-4">
    <RouteGlobalCard {...props} outbounds={props.outbounds} />
    <RuleSection object={object} metadata={metadata} metadataLoading={props.metadataLoading} metadataError={props.metadataError}
      onChange={onChange} onMetadataChange={onMetadataChange} onEdit={editRule} />
    <RuleSetSection object={object} onChange={onChange} onEdit={editRuleSet} />
    {selection?.kind === "rule" ? <RouteRuleDialog key={`${selection.index}:${JSON.stringify(selection.item)}`} open item={selection.item} metadata={selection.metadata}
      title={selection.index === null ? t("policy.route.addRuleTitle") : t("policy.route.editRuleTitle", { index: selection.index + 1 })}
      onOpenChange={(open) => { if (!open) setSelection(null) }} onSave={saveSelection} /> : null}
    {selection?.kind === "rule-set" ? <RouteRuleSetDialog key={`${selection.index}:${JSON.stringify(selection.item)}`} open item={selection.item}
      title={selection.index === null ? t("policy.route.addRuleSetTitle") : t("policy.route.editRuleSetTitle")}
      onOpenChange={(open) => { if (!open) setSelection(null) }} onSave={saveSelection} /> : null}
  </div>
}
