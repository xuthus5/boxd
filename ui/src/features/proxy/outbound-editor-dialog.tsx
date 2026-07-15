import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import {
  changeOutboundTransportType, changeOutboundType, dialerFields, dialerTypes, groupFields, groupTypes,
  outboundMultiplexFields, outboundMultiplexTypes, outboundTLSFields, outboundTLSTypes, outboundTransportTypes,
  outboundTypes, protocolFields, serverTypes, transportTypeFields,
} from "@/features/proxy/outbound-form-model"
import { ProxyFormFields } from "@/features/proxy/proxy-form-fields"
import { getPath, type FieldSpec, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"

interface OutboundEditorDialogProps {
  title: string
  item: JsonObject
  onClose: () => void
  onSave: (item: JsonObject) => void
}

function parseObject(value: string) {
  if (!isValidJSON(value)) return null
  const parsed: unknown = JSON.parse(value)
  return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as JsonObject : null
}

function typeOptions(type: string) {
  return type && !outboundTypes.includes(type as typeof outboundTypes[number]) ? [type, ...outboundTypes] : [...outboundTypes]
}

function BaseFields({ object, onChange }: { object: JsonObject; onChange: (object: JsonObject) => void }) {
  const { t } = useTranslation()
  const type = String(object.type ?? "")
  const options = useMemo(() => typeOptions(type), [type])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup className="grid gap-4 sm:grid-cols-2">
    <Field><FieldLabel htmlFor="outbound-tag">Tag</FieldLabel><Input id="outbound-tag" value={String(object.tag ?? "")} onChange={(event) => onChange(setPath(object, "tag", event.target.value || undefined))} /></Field>
    <Field><FieldLabel htmlFor="outbound-type">{t("common.type")}</FieldLabel><Select items={items} value={type || null} onValueChange={(value) => onChange(changeOutboundType(object, String(value)))}><SelectTrigger id="outbound-type" aria-label={t("common.type")} className="w-full"><SelectValue placeholder={t("proxy.outbound.selectType")} /></SelectTrigger><SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent></Select></Field>
    {serverTypes.has(type) ? <><Field><FieldLabel htmlFor="outbound-server">{t("proxy.outbound.server")}</FieldLabel><Input id="outbound-server" value={String(object.server ?? "")} onChange={(event) => onChange(setPath(object, "server", event.target.value || undefined))} /></Field><Field><FieldLabel htmlFor="outbound-port">{t("proxy.outbound.serverPort")}</FieldLabel><Input id="outbound-port" type="number" value={String(object.server_port ?? "")} onChange={(event) => onChange(setPath(object, "server_port", event.target.value ? Number(event.target.value) : undefined))} /></Field></> : null}
  </FieldGroup>
}

interface FormTabsProps {
  object: JsonObject
  value: string
  title: string
  revision: number
  onChange: (object: JsonObject) => void
  onJSONChange: (value: string) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

function ManagedGroupAlert() {
  const { t } = useTranslation()
  return <Alert><AlertTitle>{t("proxy.outbound.managedGroupTitle")}</AlertTitle><AlertDescription>{t("proxy.outbound.managedGroupDescription")}</AlertDescription></Alert>
}

function FormTabs({ object, value, title, revision, onChange, onJSONChange, onFieldValidityChange }: FormTabsProps) {
  const { t } = useTranslation()
  const type = String(object.type ?? "")
  const transportType = String(getPath(object, "transport.type") ?? "")
  const protocol = groupTypes.has(type) ? groupFields(type) : protocolFields(type)
  const transformField = useCallback((current: JsonObject, field: FieldSpec, raw: string) => field.path === "transport.type" ? changeOutboundTransportType(current, raw) : undefined, [])
  const hasTransport = outboundTransportTypes.has(type) || outboundMultiplexTypes.has(type)
  return <Tabs defaultValue="basic" className="min-h-0">
    <TabsList className="h-auto w-full justify-start overflow-x-auto" variant="line">
      <TabsTrigger value="basic">{t("proxy.outbound.basic")}</TabsTrigger>
      {dialerTypes.has(type) ? <TabsTrigger value="dialer">{t("proxy.outbound.dialing")}</TabsTrigger> : null}
      <TabsTrigger value="protocol">{t(groupTypes.has(type) ? "proxy.outbound.group" : "proxy.outbound.protocol")}</TabsTrigger>
      {outboundTLSTypes.has(type) ? <TabsTrigger value="tls">{t("proxy.outbound.tlsReality")}</TabsTrigger> : null}
      {hasTransport ? <TabsTrigger value="transport">{t("proxy.outbound.transportMultiplex")}</TabsTrigger> : null}
      <TabsTrigger value="advanced">{t("proxy.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-4"><BaseFields object={object} onChange={onChange} /></TabsContent>
    {dialerTypes.has(type) ? <TabsContent value="dialer" className="pt-4"><ProxyFormFields fields={dialerFields} object={object} namespace="proxy.outbound" onChange={onChange} /></TabsContent> : null}
    <TabsContent value="protocol" className="pt-4" keepMounted><FieldGroup>{groupTypes.has(type) ? <ManagedGroupAlert /> : null}<ProxyFormFields fields={protocol} object={object} namespace="proxy.outbound" revision={revision} onChange={onChange} onFieldValidityChange={onFieldValidityChange} /></FieldGroup></TabsContent>
    {outboundTLSTypes.has(type) ? <TabsContent value="tls" className="pt-4"><ProxyFormFields fields={outboundTLSFields} object={object} namespace="proxy.outbound" onChange={onChange} /></TabsContent> : null}
    {hasTransport ? <TabsContent value="transport" className="pt-4" keepMounted><ProxyFormFields fields={[...(outboundTransportTypes.has(type) ? transportTypeFields(transportType) : []), ...(outboundMultiplexTypes.has(type) ? outboundMultiplexFields : [])]} object={object} namespace="proxy.outbound" revision={revision} onChange={onChange} onFieldValidityChange={onFieldValidityChange} transformField={transformField} /></TabsContent> : null}
    <TabsContent value="advanced" className="pt-4"><Field><FieldLabel className="sr-only">{t("proxy.advancedJSON")}</FieldLabel><JsonEditor value={value} onChange={onJSONChange} ariaLabel={`${title} JSON`} /></Field></TabsContent>
  </Tabs>
}

export function OutboundEditorDialog({ title, item, onClose, onSave }: OutboundEditorDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState<Set<string>>(() => new Set())
  const object = parseObject(value)
  const update = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setValue(next); setRevision((current) => current + 1) }
  const updateValidity = useCallback((path: string, valid: boolean) => setInvalidFields((current) => { const next = new Set(current); if (valid) next.delete(path); else next.add(path); return next }), [])
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
    <DialogContent className="max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("proxy.outbound.editorDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">{object ? <FormTabs object={object} value={value} title={title} revision={revision} onChange={update} onJSONChange={updateJSON} onFieldValidityChange={updateValidity} /> : <JsonEditor value={value} onChange={updateJSON} ariaLabel={`${title} JSON`} />}</div>
      <DialogFooter><Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button><Button disabled={!object || typeof object.type !== "string" || !object.type || invalidFields.size > 0} onClick={() => { if (object) onSave(object) }}>{t("common.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
