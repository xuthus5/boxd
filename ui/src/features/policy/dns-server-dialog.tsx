import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { useDNSDialogState, optionsWithCurrent } from "@/features/policy/dns-dialog-state"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { changeDNSServerType, dnsServerFields, dnsServerTypes, inferDNSServerType } from "@/features/policy/dns-form-model"
import type { JsonObject, PolicyFieldSpec } from "@/features/policy/policy-form-model"

export interface DNSServerDialogProps {
  open: boolean
  item: JsonObject
  title: string
  onOpenChange: (open: boolean) => void
  onSave: (item: JsonObject) => void
}

const tagField = [{ path: "tag", label: "tag", required: true }] as const satisfies readonly PolicyFieldSpec[]
const basicPaths = new Set(["address", "server", "server_port"])
const tlsPaths = new Set(["tls.enabled", "tls.disable_sni", "tls.server_name", "tls.insecure", "tls.alpn", "tls.certificate", "tls.certificate_path"])
const httpPaths = new Set(["path", "method", "headers"])
const specialPaths: Record<string, Set<string>> = {
  local: new Set(["prefer_go"]), hosts: new Set(["path", "predefined"]),
  dhcp: new Set(["prefer_go", "interface"]), fakeip: new Set(["inet4_range", "inet6_range"]),
}
const remoteTypes = new Set(["udp", "tcp", "tls", "quic", "https", "h3"])

function serverFields(type: string, section: "basic" | "dialer" | "tls" | "special") {
  const fields = dnsServerFields[type] ?? []
  if (section === "basic") return fields.filter((field) => basicPaths.has(field.path))
  if (section === "tls") return fields.filter((field) => tlsPaths.has(field.path)
    || (["https", "h3"].includes(type) && httpPaths.has(field.path)))
  if (section === "special") return fields.filter((field) => specialPaths[type]?.has(field.path))
  return fields.filter((field) => !basicPaths.has(field.path) && !tlsPaths.has(field.path)
    && !httpPaths.has(field.path) && !specialPaths[type]?.has(field.path))
}

function requiredServerValues(object: JsonObject): boolean {
  const type = inferDNSServerType(object)
  if (typeof object.tag !== "string" || !object.tag.trim()) return false
  if (type === "legacy") return typeof object.address === "string" && Boolean(object.address.trim())
  if (remoteTypes.has(type)) return typeof object.server === "string" && Boolean(object.server.trim())
  if (type === "fakeip") return [object.inet4_range, object.inet6_range].some((value) => typeof value === "string" && value.trim())
  return true
}

function ServerTypeField({ object, onChange }: { object: JsonObject; onChange: (item: JsonObject) => void }) {
  const { t } = useTranslation()
  const current = inferDNSServerType(object)
  const options = useMemo(() => optionsWithCurrent(dnsServerTypes, current), [current])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <Field><FieldLabel htmlFor="dns-server-type">{t("policy.dns.serverType")}</FieldLabel>
    <Select items={items} value={current} onValueChange={(value) => onChange(changeDNSServerType(object, String(value)))}>
      <SelectTrigger id="dns-server-type" aria-label={t("policy.dns.serverType")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
  </Field>
}

interface ServerFieldsProps {
  object: JsonObject
  type: string
  section: "basic" | "dialer" | "tls" | "special"
  revision: number
  onChange: (item: JsonObject) => void
  onValidity: (path: string, valid: boolean) => void
  transform: ReturnType<typeof useDNSDialogState>["transform"]
}

function ServerFields(props: ServerFieldsProps) {
  const fields = serverFields(props.type, props.section)
  return <PolicyFormFields fields={fields} object={props.object} namespace="policy.dns" revision={props.revision}
    onChange={props.onChange} onFieldValidityChange={props.onValidity} transformField={props.transform} />
}

function AdvancedJSON({ value, title, revision, onChange }: {
  value: string; title: string; revision: number; onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  return <FieldGroup><Field><FieldLabel className="sr-only">{t("policy.dns.advancedJSON")}</FieldLabel>
    <JsonEditor key={revision} value={value} onChange={onChange} ariaLabel={t("policy.dns.advancedJSONLabel", { title })} />
  </Field></FieldGroup>
}

function ServerTabs({ state, title }: { state: ReturnType<typeof useDNSDialogState>; title: string }) {
  const { t } = useTranslation()
  const object = state.object
  const type = inferDNSServerType(object)
  const fieldProps = { object, type, revision: state.revision, onChange: state.update,
    onValidity: state.updateValidity, transform: state.transform }
  return <Tabs defaultValue="basic" className="min-h-0 min-w-0"><TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto" variant="line">
    <TabsTrigger value="basic">{t("policy.dns.basicTab")}</TabsTrigger><TabsTrigger value="dialer">{t("policy.dns.dialerTab")}</TabsTrigger>
    <TabsTrigger value="tls">{t("policy.dns.tlsTab")}</TabsTrigger><TabsTrigger value="special">{t("policy.dns.specialTab")}</TabsTrigger>
    <TabsTrigger value="advanced">{t("policy.dns.advancedJSON")}</TabsTrigger></TabsList>
    <TabsContent value="basic" className="pt-4" keepMounted><FieldGroup className="gap-4">
      <ServerTypeField object={object} onChange={state.update} />
      <PolicyFormFields fields={tagField} object={object} namespace="policy.dns" revision={state.revision}
        onChange={state.update} onFieldValidityChange={state.updateValidity} transformField={state.transform} />
      <ServerFields {...fieldProps} section="basic" />
    </FieldGroup></TabsContent>
    <TabsContent value="dialer" className="pt-4" keepMounted><ServerFields {...fieldProps} section="dialer" /></TabsContent>
    <TabsContent value="tls" className="pt-4" keepMounted><ServerFields {...fieldProps} section="tls" /></TabsContent>
    <TabsContent value="special" className="pt-4" keepMounted><ServerFields {...fieldProps} section="special" /></TabsContent>
    <TabsContent value="advanced" className="pt-4" keepMounted><AdvancedJSON value={state.value} title={title}
      revision={state.editorRevision} onChange={state.updateJSON} /></TabsContent>
  </Tabs>
}

export function DNSServerDialog({ open, item, title, onOpenChange, onSave }: DNSServerDialogProps) {
  const { t } = useTranslation()
  const state = useDNSDialogState(item)
  const requiredValid = requiredServerValues(state.object)
  const canSave = Boolean(state.jsonValid && requiredValid && state.invalidFields.size === 0)
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[calc(100dvh-2rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
    <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("policy.dns.serverDialogDescription")}</DialogDescription></DialogHeader>
    <div className="min-h-0 min-w-0 overflow-y-auto pr-1"><div className="flex min-w-0 flex-col gap-4">
      {!requiredValid ? <Alert variant="destructive"><AlertTitle>{t("policy.dns.requiredTitle")}</AlertTitle>
        <AlertDescription>{t("policy.dns.serverRequiredDescription")}</AlertDescription></Alert> : null}
      <ServerTabs state={state} title={title} />
    </div></div>
    <DialogFooter><Button variant="outline" onClick={() => onOpenChange(false)}>{t("policy.dns.cancel")}</Button>
      <Button disabled={!canSave} onClick={() => { if (state.jsonValid) onSave(state.object) }}>{t("policy.dns.save")}</Button></DialogFooter>
  </DialogContent></Dialog>
}
