import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { useConfigQuery } from "@/features/config/config-hooks"
import { optionsWithCurrent, useDNSDialogState } from "@/features/policy/dns-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { changeDNSAction, changeDNSRuleType, dnsActionFields, dnsActions, dnsRuleMatchFields } from "@/features/policy/dns-form-model"
import {
  isNonEmptyJsonObjectArray,
  policyConfigTags,
  policyRuleSetTags,
  setPolicyPath,
  type JsonObject,
  type PolicyFieldSpec,
} from "@/features/policy/policy-form-model"

export interface DNSRuleDialogProps {
  open: boolean
  item: JsonObject
  title: string
  serverTags: readonly string[]
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const fieldsAt = (paths: readonly string[]) => dnsRuleMatchFields.filter((field) => paths.includes(field.path))
const basicFields = fieldsAt(["inbound", "ip_version", "query_type", "network", "auth_user", "protocol"])
const domainFields = fieldsAt(["domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "source_ip_is_private", "ip_cidr", "ip_is_private"])
const processFields = fieldsAt(["source_port", "source_port_range", "port", "port_range", "process_name", "process_path", "process_path_regex", "package_name", "user", "user_id", "outbound", "clash_mode", "rule_set", "rule_set_ip_cidr_match_source", "network_type", "network_is_expensive", "network_is_constrained", "wifi_ssid", "wifi_bssid"])
const logicalFields = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"], required: true },
  { path: "invert", label: "invert", kind: "boolean" },
] as const satisfies readonly PolicyFieldSpec[]

function requiredRuleValues(object: JsonObject): boolean {
  if (object.type !== "logical") return true
  return typeof object.mode === "string" && Boolean(object.mode) && isNonEmptyJsonObjectArray(object.rules)
}

function requiredActionValue(object: JsonObject): boolean {
  const action = String(object.action ?? "route")
  if (action === "route") return typeof object.server === "string" && Boolean(object.server.trim())
  return true
}

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
    context={context} onChange={state.update} onFieldValidityChange={state.updateValidity} transformField={state.transform} />
}

function ActionFields({ state, serverTags }: { state: ReturnType<typeof useDNSDialogState>; serverTags: readonly string[] }) {
  const object = state.object!
  const action = String(object.action ?? "route")
  const fields = action === "route" ? dnsActionFields.route.filter((field) => field.path !== "server") : dnsActionFields[action] ?? []
  return <FieldGroup className="gap-4"><ActionTypeField object={object} onChange={state.update} />
    {action === "route" ? <RouteServerField object={object} tags={serverTags} onChange={state.update} /> : null}
    <FormFields state={state} fields={fields} />
  </FieldGroup>
}

function AdvancedJSON({ value, title, revision, onChange }: {
  value: string; title: string; revision: number; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.dns.advancedJSON")}</FieldLabel>
    <JsonEditor key={revision} value={value} onChange={onChange} ariaLabel={t("policy.dns.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

function RuleTabs({ state, title, serverTags }: {
  state: ReturnType<typeof useDNSDialogState>; title: string; serverTags: readonly string[]
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
  return <Tabs defaultValue="basic" className="min-h-0 min-w-0"><TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
    <TabsTrigger value="basic">{t("policy.dns.ruleBasicTab")}</TabsTrigger><TabsTrigger value="domain">{t("policy.dns.domainTab")}</TabsTrigger>
    <TabsTrigger value="process">{t("policy.dns.processTab")}</TabsTrigger><TabsTrigger value="action">{t("policy.dns.actionTab")}</TabsTrigger>
    <TabsTrigger value="advanced">{t("policy.dns.advancedJSON")}</TabsTrigger></TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted><FieldGroup className="gap-4">
      <RuleTypeField object={state.object} onChange={state.update} />
      <FormFields state={state} context={context} fields={logical ? logicalFields : [...basicFields, dnsRuleMatchFields.at(-1)!]} />
      {logical ? <Alert><AlertTitle>{t("policy.dns.logicalTitle")}</AlertTitle><AlertDescription>{t("policy.dns.logicalDescription")}</AlertDescription></Alert> : null}
    </FieldGroup></TabsContent>
    <TabsContent value="domain" className="pt-4" keepMounted><FormFields state={state} context={context} fields={logical ? [] : domainFields} /></TabsContent>
    <TabsContent value="process" className="pt-4" keepMounted><FormFields state={state} context={context} fields={logical ? [] : processFields} /></TabsContent>
    <TabsContent value="action" className="pt-4" keepMounted><ActionFields state={state} serverTags={serverTags} /></TabsContent>
    <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSON value={state.value} title={title}
      revision={state.editorRevision} onChange={state.updateJSON} /></TabsContent>
  </Tabs>
}

export function DNSRuleDialog({ open, item, title, serverTags, onOpenChange, onSave }: DNSRuleDialogProps) {
  const { t } = useTranslation()
  const state = useDNSDialogState(item)
  const requiredValid = requiredRuleValues(state.object) && requiredActionValue(state.object)
  const canSave = Boolean(state.jsonValid && requiredValid && state.invalidFields.size === 0)
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
    <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.dns.ruleDialogDescription")}</DialogDescription></DialogHeader>
    <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
      {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.dns.requiredTitle")}</AlertTitle>
        <AlertDescription>{t("policy.dns.ruleRequiredDescription")}</AlertDescription></Alert> : null}
      <RuleTabs state={state} title={title} serverTags={serverTags} />
    </div></div>
    <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>{t("policy.dns.cancel")}</Button>
      <Button disabled={!canSave} onClick={() => { if (state.jsonValid) onSave(state.object) }}>{t("policy.dns.save")}</Button></DialogFooter>
  </DialogContent></Dialog>
}
