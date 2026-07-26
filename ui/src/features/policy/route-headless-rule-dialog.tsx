import { useMemo, useRef, useState, type RefObject } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { transformDNSField } from "@/features/policy/dns-form-model"
import { usePolicyDialogState } from "@/features/policy/policy-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import type { JsonObject, PolicyFieldSpec, PolicyFieldTransform } from "@/features/policy/policy-form-model"
import {
  changeHeadlessRuleType,
  headlessRuleMatchFields,
  isHeadlessRuleComplete,
  logicalHeadlessRuleFields,
} from "@/features/policy/route-headless-rule-model"
import { transformRouteField } from "@/features/policy/route-form-transform"

interface RouteHeadlessRuleDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const paths = (values: readonly string[]) => headlessRuleMatchFields.filter((field) => values.includes(field.path))
const basicFields = paths(["query_type", "network", "invert"])
const domainFields = paths(["domain", "domain_suffix", "domain_keyword", "domain_regex", "source_ip_cidr", "ip_cidr"])
const processFields = paths([
  "source_port", "source_port_range", "port", "port_range", "process_name", "process_path",
  "process_path_regex", "package_name",
])
const environmentFields = paths([
  "network_type", "network_is_expensive", "network_is_constrained", "network_interface_address",
  "default_interface_address", "wifi_ssid", "wifi_bssid",
])
const transformHeadlessRuleField: PolicyFieldTransform = (object, field, raw) => (
  field.path === "query_type" ? transformDNSField(object, field, raw) : transformRouteField(object, field, raw)
)

function optionsWithCurrent(current: string) {
  return current && !["default", "logical"].includes(current) ? [current, "default", "logical"] : ["default", "logical"]
}

function RuleTypeSelect({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = String(object.type ?? "default")
  const options = useMemo(() => optionsWithCurrent(current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup><Field><FieldLabel htmlFor="route-headless-rule-type">{t("policy.route.ruleType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeHeadlessRuleType(object, String(value)))}>
      <SelectTrigger id="route-headless-rule-type" aria-label={t("policy.route.ruleType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field></FieldGroup>
}

function StructuredFields({ object, fields, revision, onChange, onValidity, transform }: {
  object: JsonObject
  fields: readonly PolicyFieldSpec[]
  revision: number
  onChange: (item: JsonObject) => void
  onValidity: (path: string, valid: boolean) => void
  transform: PolicyFieldTransform
}) {
  return <PolicyFormFields fields={fields} object={object} namespace="policy.route" revision={revision}
    onChange={onChange} onFieldValidityChange={onValidity} transformField={transform} />
}

function AdvancedJSONField({ value, title, revision, onChange, editorRef }: {
  value: string
  title: string
  revision: number
  onChange: (value: string) => void
  editorRef: RefObject<JsonEditorHandle | null>
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.route.advancedJSON")}</FieldLabel>
    <JsonEditor ref={editorRef} key={revision} value={value} onChange={onChange}
      ariaLabel={t("policy.route.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

export function RouteHeadlessRuleDialog({ open, item, title, onOpenChange, onSave }: RouteHeadlessRuleDialogProps) {
  const { t } = useTranslation()
  const state = usePolicyDialogState(item, transformHeadlessRuleField)
  const [activeTab, setActiveTab] = useState("basic")
  const editorRef = useRef<JsonEditorHandle>(null)
  const logical = state.object.type === "logical"
  const requiredValid = isHeadlessRuleComplete(state.object)
  const canSave = state.jsonValid && requiredValid && state.invalidFields.size === 0
  const changeType = (next: JsonObject) => {
    state.update(next)
    setActiveTab("basic")
  }
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-4xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.route.headlessRuleDialogDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
        {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.route.requiredTitle")}</AlertTitle>
          <AlertDescription>{t("policy.route.headlessRuleRequiredDescription")}</AlertDescription></Alert> : null}
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(String(value || "basic"))} className="min-w-0">
          <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
            <TabsTrigger value="basic">{t("policy.route.basicTab")}</TabsTrigger>
            {!logical ? <TabsTrigger value="domain">{t("policy.route.domainTab")}</TabsTrigger> : null}
            {!logical ? <TabsTrigger value="process">{t("policy.route.processTab")}</TabsTrigger> : null}
            {!logical ? <TabsTrigger value="environment">{t("policy.route.environmentTab")}</TabsTrigger> : null}
            <TabsTrigger value="advanced">{t("policy.route.advancedJSON")}</TabsTrigger>
          </TabsList>
          <TabsContent value="basic" className="pt-4" keepMounted><div className="flex flex-col gap-4">
            <RuleTypeSelect object={state.object} onChange={changeType} />
            <StructuredFields object={state.object} fields={logical ? logicalHeadlessRuleFields : basicFields}
              revision={state.revision} onChange={state.update} onValidity={state.updateValidity} transform={state.transform} />
            {logical ? <Alert><AlertTitle>{t("policy.route.logicalTitle")}</AlertTitle>
              <AlertDescription>{t("policy.route.logicalDescription")}</AlertDescription></Alert> : null}
          </div></TabsContent>
          <TabsContent value="domain" className="pt-4" keepMounted><StructuredFields object={state.object} fields={domainFields}
            revision={state.revision} onChange={state.update} onValidity={state.updateValidity} transform={state.transform} /></TabsContent>
          <TabsContent value="process" className="pt-4" keepMounted><div className="flex flex-col gap-4">
            <Alert><AlertTitle>{t("policy.route.processMatchTitle")}</AlertTitle>
              <AlertDescription>{t("policy.route.processMatchDescription")}</AlertDescription></Alert>
            <StructuredFields object={state.object} fields={processFields} revision={state.revision}
              onChange={state.update} onValidity={state.updateValidity} transform={state.transform} />
          </div></TabsContent>
          <TabsContent value="environment" className="pt-4" keepMounted><StructuredFields object={state.object} fields={environmentFields}
            revision={state.revision} onChange={state.update} onValidity={state.updateValidity} transform={state.transform} /></TabsContent>
          <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSONField value={state.value} title={title}
            revision={state.editorRevision} onChange={state.updateJSON} editorRef={editorRef} /></TabsContent>
        </Tabs>
      </div></div>
      <DialogFooter className="gap-2">
        <Button variant="outline" size="sm" className="h-8" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
        <Button size="sm" className="h-8" disabled={!canSave}
          onClick={() => { if (state.jsonValid) onSave(state.object) }}>{t("common.save")}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
}
