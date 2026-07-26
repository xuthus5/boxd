import { useState } from "react"
import { ArrowDownIcon, ArrowUpIcon, CopyIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { cloneJsonObject, moveItem, type JsonObject } from "@/features/policy/policy-form-model"
import { RouteHeadlessRuleDialog } from "@/features/policy/route-headless-rule-dialog"
import {
  headlessRuleMatchFields,
  headlessRules,
  setHeadlessRules,
  summarizeHeadlessRule,
} from "@/features/policy/route-headless-rule-model"

interface RouteInlineRuleSetEditorProps {
  item: JsonObject
  onChange: (item: JsonObject) => void
}

interface EditingRule {
  index: number
  item: JsonObject
}

function replaceRule(rules: readonly JsonObject[], index: number, item: JsonObject) {
  return index < 0 ? [...rules, item] : rules.map((rule, current) => current === index ? item : rule)
}

function copyRule(rules: readonly JsonObject[], index: number) {
  if (!rules[index]) return [...rules]
  return [...rules.slice(0, index + 1), cloneJsonObject(rules[index]), ...rules.slice(index + 1)]
}

function deleteRule(rules: readonly JsonObject[], index: number) {
  return rules.filter((_, current) => current !== index)
}

function InlineRuleCard({ item, index, total, onEdit, onCopy, onMove, onDelete }: {
  item: JsonObject
  index: number
  total: number
  onEdit: () => void
  onCopy: () => void
  onMove: (direction: -1 | 1) => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const number = index + 1
  const labels = new Map<string, string>(headlessRuleMatchFields.map((field) => [field.path, t(`policy.route.${field.label}`)]))
  const summary = summarizeHeadlessRule(item, { matchLabel: (path) => labels.get(path) ?? path })
  const visibleMatches = summary.matches.slice(0, 5)
  const remaining = summary.matches.length - visibleMatches.length
  const description = summary.type === "logical"
    ? t("policy.route.inlineLogicalRuleSummary", { count: summary.childRules })
    : t("policy.route.inlineDefaultRuleSummary", { count: summary.matches.length })
  return <Card size="sm">
    <CardHeader className="min-w-0 gap-1.5">
      <CardTitle>{t("policy.route.inlineRuleCardTitle", { index: number })}</CardTitle>
      <CardDescription>{description}</CardDescription>
      <CardAction><Button variant="outline" size="xs" aria-label={t("policy.route.editInlineRule", { index: number })} onClick={onEdit}>
        <PencilIcon data-icon="inline-start" />{t("policy.route.edit")}
      </Button></CardAction>
    </CardHeader>
    <CardContent className="flex flex-wrap gap-1.5">
      <Badge variant="secondary">{summary.type}</Badge>
      {visibleMatches.map((match, matchIndex) => <Badge key={`${match}-${matchIndex}`} variant="outline" className="max-w-full truncate">{match}</Badge>)}
      {remaining > 0 ? <Badge variant="outline">+{remaining}</Badge> : null}
      {summary.matches.length === 0 ? <span className="text-xs text-muted-foreground">{t("policy.route.inlineRuleNoMatches")}</span> : null}
    </CardContent>
    <CardFooter className="flex-wrap justify-end gap-1">
      <Button variant="outline" size="icon-xs" aria-label={t("policy.route.copyInlineRule", { index: number })} onClick={onCopy}>
        <CopyIcon data-icon="inline-start" />
      </Button>
      <Button variant="outline" size="icon-xs" aria-label={t("policy.route.moveInlineRuleUp", { index: number })}
        disabled={index === 0} onClick={() => onMove(-1)}><ArrowUpIcon data-icon="inline-start" /></Button>
      <Button variant="outline" size="icon-xs" aria-label={t("policy.route.moveInlineRuleDown", { index: number })}
        disabled={index === total - 1} onClick={() => onMove(1)}><ArrowDownIcon data-icon="inline-start" /></Button>
      <ConfirmAction trigger={<Button variant="destructive" size="icon-xs"
        aria-label={t("policy.route.deleteInlineRule", { index: number })}><Trash2Icon data-icon="inline-start" /></Button>}
        title={t("policy.route.deleteInlineRuleTitle", { index: number })}
        description={t("policy.route.deleteInlineRuleDescription")}
        confirmLabel={t("common.confirmDelete")} confirmVariant="destructive" onConfirm={onDelete} />
    </CardFooter>
  </Card>
}

export function RouteInlineRuleSetEditor({ item, onChange }: RouteInlineRuleSetEditorProps) {
  const { t } = useTranslation()
  const rules = headlessRules(item)
  const [editing, setEditing] = useState<EditingRule | null>(null)
  const updateRules = (next: readonly JsonObject[]) => onChange(setHeadlessRules(item, next))
  const saveEditing = (next: JsonObject) => {
    if (!editing) return
    updateRules(replaceRule(rules, editing.index, next))
    setEditing(null)
  }
  return <section className="flex flex-col gap-3" aria-label={t("policy.route.inlineRulesTitle")}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div className="min-w-0">
        <h3 className="text-sm font-medium">{t("policy.route.inlineRulesTitle")}</h3>
        <p className="text-xs text-muted-foreground">{t("policy.route.inlineRulesDescription")}</p>
        <p className="mt-1 text-xs text-muted-foreground">{t("policy.route.inlineRuleCount", { count: rules.length })}</p>
      </div>
      <Button type="button" size="sm" className="h-8" onClick={() => setEditing({ index: -1, item: {} })}>
        <PlusIcon data-icon="inline-start" />{t("policy.route.addInlineRule")}
      </Button>
    </div>
    {rules.length === 0 ? <div className="rounded-lg border border-dashed p-4 text-center">
      <p className="text-sm font-medium">{t("policy.route.emptyInlineRulesTitle")}</p>
      <p className="mt-1 text-xs text-muted-foreground">{t("policy.route.emptyInlineRulesDescription")}</p>
    </div> : <div className="grid gap-2 lg:grid-cols-2">{rules.map((rule, index) => <InlineRuleCard key={index}
      item={rule} index={index} total={rules.length} onEdit={() => setEditing({ index, item: rule })}
      onCopy={() => updateRules(copyRule(rules, index))} onMove={(direction) => updateRules(moveItem(rules, index, direction))}
      onDelete={() => updateRules(deleteRule(rules, index))} />)}</div>}
    {editing ? <RouteHeadlessRuleDialog open item={editing.item}
      title={editing.index < 0 ? t("policy.route.addInlineRuleTitle") : t("policy.route.editInlineRuleTitle", { index: editing.index + 1 })}
      onOpenChange={(open) => { if (!open) setEditing(null) }} onSave={saveEditing} /> : null}
  </section>
}
