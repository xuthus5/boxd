import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { JsonEditor } from "@/features/config/json-editor"
import { usePolicyDialogState } from "@/features/policy/policy-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import {
  isNonEmptyJsonObjectArray,
  type JsonObject,
  type PolicyFieldSpec,
  type PolicyFieldTransform,
} from "@/features/policy/policy-form-model"
import { changeRouteAction, changeRouteRuleType, routeActionFields, routeActions, routeMatchFields } from "@/features/policy/route-form-model"
import { transformRouteField } from "@/features/policy/route-form-transform"
import type { RouteRuleMetadata } from "@/lib/api/types"

export interface RouteRuleDialogProps {
  open: boolean
  item: JsonObject
  metadata?: RouteRuleMetadata
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject, metadata: RouteRuleMetadata) => void
}

const emptyMetadata: RouteRuleMetadata = { name: "", description: "" }

function MetadataFields({ metadata, onChange }: { metadata: RouteRuleMetadata; onChange: (value: RouteRuleMetadata) => void }) {
  const { t } = useTranslation()
  return <FieldGroup className="grid gap-4 sm:grid-cols-2"><Field><FieldLabel htmlFor="route-rule-name">{t("policy.route.ruleName")}</FieldLabel>
    <Input id="route-rule-name" maxLength={100} value={metadata.name} placeholder={t("policy.route.ruleNamePlaceholder")}
      onChange={(event) => onChange({ ...metadata, name: event.target.value })} />
    <FieldDescription>{t("policy.route.ruleNameDescription")}</FieldDescription></Field>
    <Field><FieldLabel htmlFor="route-rule-description">{t("policy.route.ruleDescription")}</FieldLabel>
      <Textarea id="route-rule-description" maxLength={500} value={metadata.description} placeholder={t("policy.route.ruleDescriptionPlaceholder")}
        onChange={(event) => onChange({ ...metadata, description: event.target.value })} />
    </Field></FieldGroup>
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
  return typeof object.mode === "string" && object.mode.length > 0 && isNonEmptyJsonObjectArray(object.rules)
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

function ActionFields({ object, revision, onChange, onValidity, transform }: {
  object: JsonObject; revision: number; onChange: (item: JsonObject) => void
  onValidity: (path: string, valid: boolean) => void
  transform: PolicyFieldTransform
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
      revision={revision} onChange={onChange} onFieldValidityChange={onValidity} transformField={transform} />
  </div>
}

interface RuleTabsProps {
  object: JsonObject
  value: string
  title: string
  revision: number
  editorRevision: number
  onChange: (item: JsonObject) => void
  onJSONChange: (value: string) => void
  onValidity: (path: string, valid: boolean) => void
  transform: PolicyFieldTransform
}

function StructuredFields({ object, fields, revision, onChange, onValidity, transform }: {
  object: JsonObject; fields: readonly PolicyFieldSpec[]; revision: number
  onChange: (item: JsonObject) => void; onValidity: (path: string, valid: boolean) => void
  transform: PolicyFieldTransform
}) {
  return <PolicyFormFields fields={fields} object={object} namespace="policy.route" revision={revision}
    onChange={onChange} onFieldValidityChange={onValidity} transformField={transform} />
}

function AdvancedJSONField({ value, title, revision, onChange }: {
  value: string; title: string; revision: number; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.route.advancedJSON")}</FieldLabel>
    <JsonEditor key={revision} value={value} onChange={onChange} ariaLabel={t("policy.route.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

function RuleTabs(props: RuleTabsProps) {
  const { t } = useTranslation()
  const { object, value, title, revision, editorRevision, onChange, onJSONChange, onValidity, transform } = props
  const logical = object.type === "logical"
  return <Tabs defaultValue="basic" className="min-h-0 min-w-0">
    <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
      <TabsTrigger value="basic">{t("policy.route.basicTab")}</TabsTrigger><TabsTrigger value="domain">{t("policy.route.domainTab")}</TabsTrigger>
      <TabsTrigger value="process">{t("policy.route.processTab")}</TabsTrigger><TabsTrigger value="environment">{t("policy.route.environmentTab")}</TabsTrigger>
      <TabsTrigger value="action">{t("policy.route.actionTab")}</TabsTrigger><TabsTrigger value="advanced">{t("policy.route.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted>
      <div className="flex flex-col gap-4"><RuleTypeSelect object={object} onChange={onChange} />
        <StructuredFields object={object} fields={logical ? logicalFields : basicFields.slice(1)} revision={revision} onChange={onChange} onValidity={onValidity} transform={transform} />
        {logical ? <Alert><AlertTitle>{t("policy.route.logicalTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.logicalDescription")}</AlertDescription></Alert> : null}
      </div>
    </TabsContent>
    <TabsContent value="domain" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : domainFields} revision={revision} onChange={onChange} onValidity={onValidity} transform={transform} /></TabsContent>
    <TabsContent value="process" className="pt-4" keepMounted>
      <div className="flex flex-col gap-4">
        {logical ? null : <Alert>
          <AlertTitle>{t("policy.route.processMatchTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.processMatchDescription")}</AlertDescription>
        </Alert>}
        <StructuredFields object={object} fields={logical ? [] : processFields} revision={revision} onChange={onChange} onValidity={onValidity} transform={transform} />
      </div>
    </TabsContent>
    <TabsContent value="environment" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : environmentFields} revision={revision} onChange={onChange} onValidity={onValidity} transform={transform} /></TabsContent>
    <TabsContent value="action" className="pt-4" keepMounted><ActionFields object={object} revision={revision} onChange={onChange} onValidity={onValidity} transform={transform} /></TabsContent>
    <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSONField value={value} title={title}
      revision={editorRevision} onChange={onJSONChange} /></TabsContent>
  </Tabs>
}

export function RouteRuleDialog({ open, item, metadata = emptyMetadata, title, onOpenChange, onSave }: RouteRuleDialogProps) {
  const { t } = useTranslation()
  const state = usePolicyDialogState(item, transformRouteField)
  const [details, setDetails] = useState(metadata)
  const requiredValid = requiredRuleValue(state.object) && requiredActionValue(state.object)
  const canSave = state.jsonValid && requiredValid && state.invalidFields.size === 0
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.route.ruleDialogDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
        {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.route.requiredTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.ruleRequiredDescription")}</AlertDescription></Alert> : null}
        <MetadataFields metadata={details} onChange={setDetails} />
        <RuleTabs object={state.object} value={state.value} title={title} revision={state.revision}
          editorRevision={state.editorRevision} onChange={state.update} onJSONChange={state.updateJSON}
          onValidity={state.updateValidity} transform={state.transform} />
      </div></div>
      <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>{t("policy.route.cancel")}</Button>
        <Button disabled={!canSave} onClick={() => { if (state.jsonValid) onSave(state.object, details) }}>{t("policy.route.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
