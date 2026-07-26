import { useState, type ReactNode } from "react"
import { ArrowDownIcon, ArrowUpIcon, CopyIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty"
import { cloneJsonObject, moveItem, type JsonObject, type PolicySection } from "@/features/policy/policy-form-model"

export interface PolicyLogicalRuleEditorRenderProps {
  item: JsonObject
  index: number
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

interface PolicyLogicalRuleSummary {
  matches: string[]
  action: string
}

interface PolicyLogicalRulesEditorProps {
  section: PolicySection
  rules: readonly JsonObject[]
  onChange: (rules: JsonObject[]) => void
  summarize: (rule: JsonObject) => PolicyLogicalRuleSummary
  renderEditor: (props: PolicyLogicalRuleEditorRenderProps) => ReactNode
}

interface EditingRule {
  index: number
  item: JsonObject
}

function replaceRule(rules: readonly JsonObject[], index: number, item: JsonObject) {
  return index < 0 ? [...rules, item] : rules.map((rule, current) => current === index ? item : rule)
}

function copyRule(rules: readonly JsonObject[], index: number) {
  return [...rules.slice(0, index + 1), cloneJsonObject(rules[index]), ...rules.slice(index + 1)]
}

function deleteRule(rules: readonly JsonObject[], index: number) {
  return rules.filter((_, current) => current !== index)
}

function translationKey(section: PolicySection, key: string) {
  return `policy.${section}.${key}`
}

function LogicalRuleCard({ section, item, index, total, summarize, onEdit, onCopy, onMove, onDelete }: {
  section: PolicySection
  item: JsonObject
  index: number
  total: number
  summarize: (rule: JsonObject) => PolicyLogicalRuleSummary
  onEdit: () => void
  onCopy: () => void
  onMove: (direction: -1 | 1) => void
  onDelete: () => void
}) {
  const { t } = useTranslation()
  const number = index + 1
  const summary = summarize(item)
  const type = String(item.type ?? "default")
  const childCount = Array.isArray(item.rules) ? item.rules.length : 0
  const visibleMatches = summary.matches.slice(0, 4)
  const remaining = summary.matches.length - visibleMatches.length
  const description = type === "logical"
    ? t(translationKey(section, "logicalRuleChildrenSummary"), { count: childCount })
    : t(translationKey(section, "logicalRuleActionSummary"), { action: summary.action })
  return <Card size="sm">
    <CardHeader className="min-w-0 gap-1.5">
      <CardTitle>{t(translationKey(section, "logicalRuleCardTitle"), { index: number })}</CardTitle>
      <CardDescription>{description}</CardDescription>
      <CardAction><Button variant="outline" size="xs" aria-label={t(translationKey(section, "editLogicalRule"), { index: number })} onClick={onEdit}>
        <PencilIcon data-icon="inline-start" />{t(translationKey(section, "edit"))}
      </Button></CardAction>
    </CardHeader>
    <CardContent className="flex flex-wrap gap-1.5">
      <Badge variant="secondary">{type}</Badge>
      <Badge variant="outline">{summary.action}</Badge>
      {visibleMatches.map((match, matchIndex) => <Badge key={`${match}-${matchIndex}`} variant="outline" className="max-w-full truncate">{match}</Badge>)}
      {remaining > 0 ? <Badge variant="outline">+{remaining}</Badge> : null}
      {summary.matches.length === 0 ? <span className="text-xs text-muted-foreground">{t(translationKey(section, "logicalRuleNoMatches"))}</span> : null}
    </CardContent>
    <CardFooter className="flex-wrap justify-end gap-1">
      <Button variant="outline" size="icon-xs" aria-label={t(translationKey(section, "copyLogicalRule"), { index: number })} onClick={onCopy}>
        <CopyIcon data-icon="inline-start" />
      </Button>
      <Button variant="outline" size="icon-xs" aria-label={t(translationKey(section, "moveLogicalRuleUp"), { index: number })}
        disabled={index === 0} onClick={() => onMove(-1)}><ArrowUpIcon data-icon="inline-start" /></Button>
      <Button variant="outline" size="icon-xs" aria-label={t(translationKey(section, "moveLogicalRuleDown"), { index: number })}
        disabled={index === total - 1} onClick={() => onMove(1)}><ArrowDownIcon data-icon="inline-start" /></Button>
      <ConfirmAction trigger={<Button variant="destructive" size="icon-xs"
        aria-label={t(translationKey(section, "deleteLogicalRule"), { index: number })}><Trash2Icon data-icon="inline-start" /></Button>}
        title={t(translationKey(section, "deleteLogicalRuleTitle"), { index: number })}
        description={t(translationKey(section, "deleteLogicalRuleDescription"))}
        confirmLabel={t("common.confirmDelete")} confirmVariant="destructive" onConfirm={onDelete} />
    </CardFooter>
  </Card>
}

export function PolicyLogicalRulesEditor({ section, rules, onChange, summarize, renderEditor }: PolicyLogicalRulesEditorProps) {
  const { t } = useTranslation()
  const [editing, setEditing] = useState<EditingRule | null>(null)
  const titleId = `${section}-logical-rules-title`
  return <section className="flex flex-col gap-3" aria-labelledby={titleId}>
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div className="min-w-0">
        <h3 id={titleId} className="text-sm font-medium">{t(translationKey(section, "logicalRulesTitle"))}</h3>
        <p className="text-xs text-muted-foreground">{t(translationKey(section, "logicalRulesDescription"))}</p>
        <p className="mt-1 text-xs text-muted-foreground">{t(translationKey(section, "logicalRuleCount"), { count: rules.length })}</p>
      </div>
      <Button type="button" size="sm" className="h-8" onClick={() => setEditing({ index: -1, item: {} })}>
        <PlusIcon data-icon="inline-start" />{t(translationKey(section, "addLogicalRule"))}
      </Button>
    </div>
    {rules.length === 0 ? <Empty className="border py-6"><EmptyHeader>
      <EmptyTitle>{t(translationKey(section, "emptyLogicalRulesTitle"))}</EmptyTitle>
      <EmptyDescription>{t(translationKey(section, "emptyLogicalRulesDescription"))}</EmptyDescription>
    </EmptyHeader></Empty> : <div className="grid gap-2 lg:grid-cols-2">{rules.map((rule, index) => <LogicalRuleCard key={index}
      section={section} item={rule} index={index} total={rules.length} summarize={summarize}
      onEdit={() => setEditing({ index, item: rule })} onCopy={() => onChange(copyRule(rules, index))}
      onMove={(direction) => onChange(moveItem(rules, index, direction))} onDelete={() => onChange(deleteRule(rules, index))} />)}</div>}
    {editing ? renderEditor({
      item: editing.item,
      index: editing.index,
      title: editing.index < 0
        ? t(translationKey(section, "addLogicalRuleTitle"))
        : t(translationKey(section, "editLogicalRuleTitle"), { index: editing.index + 1 }),
      onOpenChange: () => setEditing(null),
      onSave: (item) => {
        onChange(replaceRule(rules, editing.index, item))
        setEditing(null)
      },
    }) : null}
  </section>
}
