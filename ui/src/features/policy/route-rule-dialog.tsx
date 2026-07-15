import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { isJsonObject, type JsonObject, type PolicyFieldSpec } from "@/features/policy/policy-form-model"
import { changeRouteAction, changeRouteRuleType, routeActionFields, routeActions, routeMatchFields } from "@/features/policy/route-form-model"
import type { JsonValue } from "@/lib/api/types"

export interface RouteRuleDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const paths = (values: readonly string[]) => routeMatchFields.filter((field) => values.includes(field.path))
const basicFields = paths(["type", "inbound", "ip_version", "network", "auth_user", "protocol", "client", "invert"])
const domainFields = paths(["domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "source_ip_is_private", "ip_cidr", "ip_is_private"])
const processFields = paths(["source_port", "source_port_range", "port", "port_range", "process_name", "process_path", "process_path_regex", "package_name", "user", "user_id"])
const environmentFields = paths(["rule_set", "rule_set_ip_cidr_match_source", "clash_mode", "network_type", "network_is_expensive", "network_is_constrained", "wifi_ssid", "wifi_bssid"])
const logicalFields = [
  { path: "mode", label: "logicalMode", kind: "select", options: ["and", "or"], required: true },
  { path: "rules", label: "logicalRules", kind: "json-array", required: true },
  routeMatchFields.at(-1)!,
] as const satisfies readonly PolicyFieldSpec[]

function parseObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch (error) {
    void error
    return null
  }
}

function optionsWithCurrent(values: readonly string[], current: string) {
  return current && !values.includes(current) ? [current, ...values] : [...values]
}

function requiredActionValue(object: JsonObject): boolean {
  const action = String(object.action ?? "route")
  if (action === "route" || action === "bypass") return typeof object.outbound === "string" && object.outbound.length > 0
  if (action === "resolve") return typeof object.server === "string" && object.server.length > 0
  return true
}

function requiredRuleValue(object: JsonObject): boolean {
  if (object.type !== "logical") return true
  return typeof object.mode === "string" && object.mode.length > 0 && Array.isArray(object.rules)
}

function RuleTypeSelect({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = String(object.type ?? "default")
  const options = useMemo(() => optionsWithCurrent(["default", "logical"], current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup><Field><FieldLabel htmlFor="route-rule-type">{t("policy.route.ruleType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeRouteRuleType(object, String(value)))}>
      <SelectTrigger id="route-rule-type" aria-label={t("policy.route.ruleType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field></FieldGroup>
}

function ActionFields({ object, revision, onChange, onValidity }: {
  object: JsonObject; revision: number; onChange: (item: JsonObject) => void
  onValidity: (path: string, valid: boolean) => void
}) {
  const { t } = useTranslation()
  const current = String(object.action ?? "route")
  const options = useMemo(() => optionsWithCurrent(routeActions, current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <div className="flex flex-col gap-4">
    <FieldGroup><Field><FieldLabel htmlFor="route-rule-action">{t("policy.route.actionType")}</FieldLabel>
      <Select items={items} value={current} onValueChange={(value) => onChange(changeRouteAction(object, String(value)))}>
        <SelectTrigger id="route-rule-action" aria-label={t("policy.route.actionType")} className="w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
    </Field></FieldGroup>
    <PolicyFormFields fields={routeActionFields[current] ?? []} object={object} namespace="policy.route"
      revision={revision} onChange={onChange} onFieldValidityChange={onValidity} />
  </div>
}

interface RuleTabsProps {
  object: JsonObject
  value: string
  title: string
  revision: number
  onChange: (item: JsonObject) => void
  onJSONChange: (value: string) => void
  onValidity: (path: string, valid: boolean) => void
}

function StructuredFields({ object, fields, revision, onChange, onValidity }: {
  object: JsonObject; fields: readonly PolicyFieldSpec[]; revision: number
  onChange: (item: JsonObject) => void; onValidity: (path: string, valid: boolean) => void
}) {
  return <PolicyFormFields fields={fields} object={object} namespace="policy.route" revision={revision}
    onChange={onChange} onFieldValidityChange={onValidity} />
}

function AdvancedJSONField({ value, title, onChange }: {
  value: string; title: string; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.route.advancedJSON")}</FieldLabel>
    <JsonEditor value={value} onChange={onChange} ariaLabel={t("policy.route.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

function RuleTabs(props: RuleTabsProps) {
  const { t } = useTranslation()
  const { object, value, title, revision, onChange, onJSONChange, onValidity } = props
  const logical = object.type === "logical"
  return <Tabs defaultValue="basic" className="min-h-0 min-w-0">
    <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto" variant="line">
      <TabsTrigger value="basic">{t("policy.route.basicTab")}</TabsTrigger><TabsTrigger value="domain">{t("policy.route.domainTab")}</TabsTrigger>
      <TabsTrigger value="process">{t("policy.route.processTab")}</TabsTrigger><TabsTrigger value="environment">{t("policy.route.environmentTab")}</TabsTrigger>
      <TabsTrigger value="action">{t("policy.route.actionTab")}</TabsTrigger><TabsTrigger value="advanced">{t("policy.route.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted>
      <div className="flex flex-col gap-4"><RuleTypeSelect object={object} onChange={onChange} />
        <StructuredFields object={object} fields={logical ? logicalFields : basicFields.slice(1)} revision={revision} onChange={onChange} onValidity={onValidity} />
        {logical ? <Alert><AlertTitle>{t("policy.route.logicalTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.logicalDescription")}</AlertDescription></Alert> : null}
      </div>
    </TabsContent>
    <TabsContent value="domain" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : domainFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="process" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : processFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="environment" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : environmentFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="action" className="pt-4" keepMounted><ActionFields object={object} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="advanced" className="pt-4"><AdvancedJSONField value={value} title={title} onChange={onJSONChange} /></TabsContent>
  </Tabs>
}

export function RouteRuleDialog({ open, item, title, onOpenChange, onSave }: RouteRuleDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const object = parseObject(value)
  const update = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setValue(next); setRevision((current) => current + 1); setInvalidFields(new Set()) }
  const updateValidity = useCallback((path: string, valid: boolean) => setInvalidFields((current) => {
    const next = new Set(current)
    if (valid) next.delete(path); else next.add(path)
    return next
  }), [])
  const requiredValid = Boolean(object && requiredRuleValue(object) && requiredActionValue(object))
  const canSave = requiredValid && invalidFields.size === 0
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.route.ruleDialogDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
        {object && !requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.route.requiredTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.ruleRequiredDescription")}</AlertDescription></Alert> : null}
        {object ? <RuleTabs object={object} value={value} title={title} revision={revision} onChange={update} onJSONChange={updateJSON} onValidity={updateValidity} />
          : <AdvancedJSONField value={value} title={title} onChange={updateJSON} />}
      </div></div>
      <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>{t("policy.route.cancel")}</Button>
        <Button disabled={!canSave} onClick={() => { if (object) onSave(object) }}>{t("policy.route.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
