import { useMemo, useRef, useState, type RefObject } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { usePolicyDialogPathReveal } from "@/features/policy/use-policy-dialog-path-reveal"
import { PolicyDialogFooter } from "@/features/policy/policy-dialog-footer"
import { PolicyLogicalRulesEditor } from "@/features/policy/policy-logical-rules-editor"
import { policyItemErrorRelativePath, usePolicyItemValidate } from "@/features/policy/use-policy-item-validate"
import { useConfigQuery } from "@/features/config/config-hooks"
import { optionsWithCurrent, useDNSDialogState } from "@/features/policy/dns-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { applyDNSRuleFieldChange, changeDNSAction, changeDNSRuleType, dnsActionFields, dnsActions, dnsRuleMatchFields, isDNSRuleComplete, summarizeDNSRule } from "@/features/policy/dns-form-model"
import {
  isJsonObject,
  policyConfigTags,
  policyRuleSetTags,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"

export interface DNSRuleDialogProps {
  open: boolean
  item: JsonObject
  index?: number
  title: string
  serverTags: readonly string[]
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  nested?: boolean
}

const fieldsAt = (paths: readonly string[]) => dnsRuleMatchFields.filter((field) => paths.includes(field.path))
const basicFields = fieldsAt(["inbound", "ip_version", "query_type", "network", "auth_user", "protocol"])
const domainFields = fieldsAt(["domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "source_ip_is_private", "ip_cidr", "ip_is_private", "ip_accept_any"])
const processFields = fieldsAt(["source_port", "source_port_range", "port", "port_range", "process_name", "process_path", "process_path_regex", "package_name", "user", "user_id", "outbound", "clash_mode", "rule_set", "rule_set_ip_cidr_match_source", "network_type", "network_is_expensive", "network_is_constrained", "interface_address", "network_interface_address", "default_interface_address", "wifi_ssid", "wifi_bssid", "rule_set_ip_cidr_accept_empty"])
const logicalFields = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"], required: true },
  { path: "invert", label: "invert", kind: "boolean" },
] as const satisfies readonly PolicyFieldSpec[]
const nestedLogicalFields = [logicalFields[0], { path: "rules", label: "logicalRules", kind: "json-array", required: true }, logicalFields[1]] as const satisfies readonly PolicyFieldSpec[]

function RuleTypeField({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = String(object.type ?? "default")
  const options = useMemo(() => optionsWithCurrent(["default", "logical"], current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <Field><FieldLabel htmlFor="dns-rule-type">{t("policy.dns.ruleType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeDNSRuleType(object, String(value)))}>
      <SelectTrigger id="dns-rule-type" aria-label={t("policy.dns.ruleType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select></Field>
}

function ActionTypeField({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = String(object.action ?? "route")
  const options = useMemo(() => optionsWithCurrent(dnsActions, current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <Field><FieldLabel htmlFor="dns-rule-action">{t("policy.dns.actionType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeDNSAction(object, String(value)))}>
      <SelectTrigger id="dns-rule-action" aria-label={t("policy.dns.actionType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select></Field>
}

function RouteServerField({ object, tags, onChange }: {
  object: JsonObject; tags: readonly string[]; onChange: (item: JsonObject) => void
}) {
  const { t } = useTranslation()
  const current = typeof object.server === "string" ? object.server : ""
  const options = useMemo(() => optionsWithCurrent(tags, current), [current, tags])
  const items = useMemo(() => [{ value: null, label: t("policy.dns.notSet") }, ...options.map((value) => ({ value, label: value }))], [options, t])
  return <Field data-invalid={!current}><FieldLabel htmlFor="dns-rule-server">{t("policy.dns.targetServer")}</FieldLabel>
    <Select items={items} value={current || null} onValueChange={(value) => onChange(setPolicyPath(object, "server", value ? String(value) : undefined))}>
      <SelectTrigger id="dns-rule-server" aria-label={t("policy.dns.targetServer")} aria-invalid={!current} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup><SelectItem value={null}>{t("policy.dns.notSet")}</SelectItem>
        {options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select></Field>
}

function FormFields({ state, fields, context }: {
  state: ReturnType<typeof useDNSDialogState>; fields: readonly PolicyFieldSpec[]
  context?: import("@/features/policy/policy-form-model").PolicyFormContext
}) {
  return <PolicyFormFields fields={fields} object={state.object} namespace="policy.dns" revision={state.revision}
    context={context} onChange={(next) => state.update(applyDNSRuleFieldChange(state.object, next))}
    onFieldValidityChange={state.updateValidity} transformField={state.transform} />
}

function ActionFields({ state, serverTags, context }: {
  state: ReturnType<typeof useDNSDialogState>; serverTags: readonly string[]
  context?: import("@/features/policy/policy-form-model").PolicyFormContext
}) {
  const object = state.object!
  const action = String(object.action ?? "route")
  const fields = action === "route" ? dnsActionFields.route.filter((field) => field.path !== "server") : dnsActionFields[action] ?? []
  const actionContext = { ...context, dnsServerTags: serverTags ? [...serverTags] : context?.dnsServerTags }
  return <FieldGroup className="gap-4"><ActionTypeField object={object} onChange={state.update} />
    {action === "route" ? <RouteServerField object={object} tags={serverTags} onChange={state.update} /> : null}
    <FormFields state={state} fields={fields} context={actionContext} />
  </FieldGroup>
}

function AdvancedJSON({ value, title, revision, onChange, editorRef }: {
  value: string; title: string; revision: number; onChange: (value: string) => void
  editorRef?: RefObject<JsonEditorHandle | null>
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.dns.advancedJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} key={revision} value={value} onChange={onChange} ariaLabel={t("policy.dns.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

function RuleTabs({ state, title, serverTags, activeTab, onTabChange, editorRef, nested }: {
  state: ReturnType<typeof useDNSDialogState>; title: string; serverTags: readonly string[]
  activeTab: string; onTabChange: (tab: string) => void
  editorRef?: RefObject<JsonEditorHandle | null>
  nested: boolean
}) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const logical = state.object.type === "logical"
  const context = {
    inboundTags: policyConfigTags(config.data?.inbounds),
    outboundTags: policyConfigTags(config.data?.outbounds),
    dnsServerTags: serverTags ? [...serverTags] : [],
    ruleSetTags: policyRuleSetTags(config.data?.route),
  }
  return <Tabs value={activeTab} onValueChange={(v) => onTabChange(String(v || "basic"))} className="min-h-0 min-w-0"><TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
    <TabsTrigger value="basic">{t("policy.dns.ruleBasicTab")}</TabsTrigger><TabsTrigger value="domain">{t("policy.dns.domainTab")}</TabsTrigger>
    <TabsTrigger value="process">{t("policy.dns.processTab")}</TabsTrigger><TabsTrigger value="action">{t("policy.dns.actionTab")}</TabsTrigger>
    <TabsTrigger value="advanced">{t("policy.dns.advancedJSON")}</TabsTrigger></TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted><FieldGroup className="gap-4">
      <RuleTypeField object={state.object} onChange={state.update} />
      <FormFields state={state} context={context} fields={logical ? nested ? nestedLogicalFields : logicalFields : [...basicFields, dnsRuleMatchFields.at(-1)!]} />
      {logical && !nested ? <PolicyLogicalRulesEditor section="dns"
        rules={Array.isArray(state.object.rules) ? state.object.rules.filter(isJsonObject) : []}
        onChange={(rules) => state.update({ ...state.object, rules })}
        summarize={(rule) => summarizeDNSRule(rule, {
          logicalMode: (value) => t("policy.dns.summaryLogicalMode", { value }),
          matchLabel: (path) => t(`policy.dns.${dnsRuleMatchFields.find((field) => field.path === path)?.label ?? path}`),
        })}
        renderEditor={(editor) => <DNSRuleDialog nested open item={editor.item} title={editor.title} serverTags={serverTags}
          onOpenChange={editor.onOpenChange} onSave={editor.onSave} />} /> : null}
      {logical ? <Alert><AlertTitle>{t("policy.dns.logicalTitle")}</AlertTitle><AlertDescription>{t("policy.dns.logicalDescription")}</AlertDescription></Alert> : null}
    </FieldGroup></TabsContent>
    <TabsContent value="domain" className="pt-4" keepMounted><FormFields state={state} context={context} fields={logical ? [] : domainFields} /></TabsContent>
    <TabsContent value="process" className="pt-4" keepMounted><FormFields state={state} context={context} fields={logical ? [] : processFields} /></TabsContent>
    <TabsContent value="action" className="pt-4" keepMounted><ActionFields state={state} serverTags={serverTags} context={context} /></TabsContent>
    <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSON value={state.value} title={title}
      revision={state.editorRevision} onChange={state.updateJSON} editorRef={editorRef} /></TabsContent>
  </Tabs>
}

export function DNSRuleDialog({ open, item, index = -1, title, serverTags, jumpPath, onJumpPathHandled, onOpenChange, onSave, nested = false }: DNSRuleDialogProps) {
  const { t } = useTranslation()
  const state = useDNSDialogState(item)
  const [activeTab, setActiveTab] = useState("basic")
  const editorRef = useRef<JsonEditorHandle>(null)
  const revealPath = usePolicyDialogPathReveal(editorRef, setActiveTab, jumpPath, onJumpPathHandled)
  const requiredValid = isDNSRuleComplete(state.object)
  const canSave = Boolean(state.jsonValid && requiredValid && state.invalidFields.size === 0)
  const { validating, validate, ready } = usePolicyItemValidate({
    section: "dns", kind: "rules", index, object: canSave ? state.object : null,
    onReportedError: (err) => {
      const relative = policyItemErrorRelativePath(err.path, "dns", "rules", index)
      if (relative) revealPath(relative)
    },
  })
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
    <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.dns.ruleDialogDescription")}</DialogDescription></DialogHeader>
    <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
      {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.dns.requiredTitle")}</AlertTitle>
        <AlertDescription>{t("policy.dns.ruleRequiredDescription")}</AlertDescription></Alert> : null}
      <RuleTabs state={state} title={title} serverTags={serverTags} activeTab={activeTab} onTabChange={setActiveTab} editorRef={editorRef} nested={nested} />
    </div></div>
    <PolicyDialogFooter canSave={canSave} canValidate={canSave && ready} validating={validating}
      onClose={() => onOpenChange(false)} onSave={() => { if (state.jsonValid) onSave(state.object) }}
      onValidate={() => { void validate() }} />
  </DialogContent></Dialog>
}
