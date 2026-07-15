import { useCallback, useMemo, useState } from "react"

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
  const current = String(object.type ?? "default")
  const options = useMemo(() => optionsWithCurrent(["default", "logical"], current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup><Field><FieldLabel htmlFor="route-rule-type">规则类型</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeRouteRuleType(object, String(value)))}>
      <SelectTrigger id="route-rule-type" aria-label="规则类型" className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field></FieldGroup>
}

function ActionFields({ object, revision, onChange, onValidity }: {
  object: JsonObject; revision: number; onChange: (item: JsonObject) => void
  onValidity: (path: string, valid: boolean) => void
}) {
  const current = String(object.action ?? "route")
  const options = useMemo(() => optionsWithCurrent(routeActions, current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <>
    <FieldGroup><Field><FieldLabel htmlFor="route-rule-action">执行动作</FieldLabel>
      <Select items={items} value={current} onValueChange={(value) => onChange(changeRouteAction(object, String(value)))}>
        <SelectTrigger id="route-rule-action" aria-label="执行动作" className="w-full"><SelectValue /></SelectTrigger>
        <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
    </Field></FieldGroup>
    <PolicyFormFields fields={routeActionFields[current] ?? []} object={object} namespace="policy.route"
      revision={revision} onChange={onChange} onFieldValidityChange={onValidity} />
  </>
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
  return <FieldGroup><Field><FieldLabel className="sr-only">高级 JSON</FieldLabel>
    <JsonEditor value={value} onChange={onChange} ariaLabel={`${title} JSON`} />
  </Field></FieldGroup>
}

function RuleTabs(props: RuleTabsProps) {
  const { object, value, title, revision, onChange, onJSONChange, onValidity } = props
  const logical = object.type === "logical"
  return <Tabs defaultValue="basic" className="min-h-0">
    <TabsList className="h-auto w-full justify-start overflow-x-auto" variant="line">
      <TabsTrigger value="basic">基础与网络</TabsTrigger><TabsTrigger value="domain">域名与地址</TabsTrigger>
      <TabsTrigger value="process">端口与进程</TabsTrigger><TabsTrigger value="environment">规则集与网络环境</TabsTrigger>
      <TabsTrigger value="action">执行动作</TabsTrigger><TabsTrigger value="advanced">高级 JSON</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted>
      <><RuleTypeSelect object={object} onChange={onChange} />
        <StructuredFields object={object} fields={logical ? logicalFields : basicFields.slice(1)} revision={revision} onChange={onChange} onValidity={onValidity} />
      </>
    </TabsContent>
    <TabsContent value="domain" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : domainFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="process" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : processFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="environment" className="pt-4" keepMounted><StructuredFields object={object} fields={logical ? [] : environmentFields} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="action" className="pt-4" keepMounted><ActionFields object={object} revision={revision} onChange={onChange} onValidity={onValidity} /></TabsContent>
    <TabsContent value="advanced" className="pt-4"><AdvancedJSONField value={value} title={title} onChange={onJSONChange} /></TabsContent>
  </Tabs>
}

export function RouteRuleDialog({ open, item, title, onOpenChange, onSave }: RouteRuleDialogProps) {
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
  const canSave = Boolean(object && requiredRuleValue(object) && requiredActionValue(object) && invalidFields.size === 0)
  return <Dialog open={open} onOpenChange={onOpenChange}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>常用匹配与动作可视化编辑，未知字段保留在高级 JSON 中。</DialogDescription></DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">
        {object ? <RuleTabs object={object} value={value} title={title} revision={revision} onChange={update} onJSONChange={updateJSON} onValidity={updateValidity} />
          : <AdvancedJSONField value={value} title={title} onChange={updateJSON} />}
      </div>
      <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
        <Button disabled={!canSave} onClick={() => { if (object) onSave(object) }}>保存</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
