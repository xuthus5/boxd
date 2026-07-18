import { useCallback, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useConfigQuery } from "@/features/config/config-hooks"
import { JsonEditor } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import { InboundFormFields } from "@/features/proxy/inbound-form-fields"
import {
  changeInboundType, getPath, inboundTypes, listenFields, multiplexFields, multiplexTypes, protocolFields, tlsFields, tlsTypes,
  transportTypeFields, transportTypes, tunFields, type JsonObject,
} from "@/features/proxy/inbound-form-model"
import { configTags, dnsServerTags } from "@/features/proxy/proxy-form-model"

interface InboundEditorDialogProps {
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
  return type && !inboundTypes.includes(type as typeof inboundTypes[number]) ? [type, ...inboundTypes] : [...inboundTypes]
}

function BaseFields({ object, onChange }: { object: JsonObject; onChange: (object: JsonObject) => void }) {
  const { t } = useTranslation()
  const type = String(object.type ?? "")
  const options = useMemo(() => typeOptions(type), [type])
  const items = useMemo(() => options.map((value) => ({ value, label: value })), [options])
  return <FieldGroup className="grid gap-4 sm:grid-cols-2">
    <Field><FieldLabel htmlFor="inbound-tag">Tag</FieldLabel><Input id="inbound-tag" value={String(object.tag ?? "")} onChange={(event) => onChange({ ...object, tag: event.target.value })} /></Field>
    <Field><FieldLabel htmlFor="inbound-type">{t("common.type")}</FieldLabel><Select items={items} value={type || null} onValueChange={(value) => onChange(changeInboundType(object, String(value)))}><SelectTrigger id="inbound-type" aria-label={t("common.type")} className="w-full"><SelectValue placeholder={t("proxy.inbound.selectType")} /></SelectTrigger><SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent></Select></Field>
    <div className="sm:col-span-2">{type === "tun" ? <InboundFormFields fields={tunFields.slice(0, 4)} object={object} type={type} onChange={onChange} /> : <InboundFormFields fields={listenFields.slice(0, 2)} object={object} type={type} onChange={onChange} />}</div>
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

function FormTabs({ object, value, title, revision, onChange, onJSONChange, onFieldValidityChange }: FormTabsProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const type = String(object.type ?? "")
  const transportType = String(getPath(object, "transport.type") ?? "")
  const hasTLS = tlsTypes.has(type)
  const hasTransport = transportTypes.has(type) || multiplexTypes.has(type)
  const currentTag = String(object.tag ?? "")
  const context = {
    currentTag,
    inboundType: type,
    inboundTags: configTags(config.data?.inbounds, currentTag),
    outboundTags: configTags(config.data?.outbounds, currentTag),
    dnsServerTags: dnsServerTags(config.data?.dns),
  }
  return <Tabs defaultValue="basic" className="min-h-0">
    <TabsList className="h-auto w-full justify-start overflow-x-auto" variant="line">
      <TabsTrigger value="basic">{t("proxy.inbound.basic")}</TabsTrigger>
      <TabsTrigger value="listen">{t(type === "tun" ? "proxy.inbound.tun" : "proxy.inbound.listenAndConnection")}</TabsTrigger>
      <TabsTrigger value="protocol">{t("proxy.inbound.protocol")}</TabsTrigger>
      {hasTLS ? <TabsTrigger value="tls">{t("proxy.inbound.tlsReality")}</TabsTrigger> : null}
      {hasTransport ? <TabsTrigger value="transport">{t("proxy.inbound.transportMultiplex")}</TabsTrigger> : null}
      <TabsTrigger value="advanced">{t("proxy.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-4"><BaseFields object={object} onChange={onChange} /></TabsContent>
    <TabsContent value="listen" className="pt-4"><InboundFormFields fields={type === "tun" ? tunFields.slice(4) : listenFields.slice(2)} object={object} type={type} context={context} onChange={onChange} /></TabsContent>
    <TabsContent value="protocol" className="pt-4" keepMounted><InboundFormFields fields={protocolFields(type)} object={object} type={type} revision={revision} context={context} onChange={onChange} onFieldValidityChange={onFieldValidityChange} /></TabsContent>
    {hasTLS ? <TabsContent value="tls" className="pt-4"><InboundFormFields fields={tlsFields} object={object} type={type} context={context} onChange={onChange} /></TabsContent> : null}
    {hasTransport ? <TabsContent value="transport" className="pt-4" keepMounted><InboundFormFields fields={[...(transportTypes.has(type) ? transportTypeFields(transportType) : []), ...(multiplexTypes.has(type) ? multiplexFields : [])]} object={object} type={type} revision={revision} context={context} onChange={onChange} onFieldValidityChange={onFieldValidityChange} /></TabsContent> : null}
    <TabsContent value="advanced" className="pt-4"><Field><FieldLabel className="sr-only">{t("proxy.advancedJSON")}</FieldLabel><JsonEditor value={value} onChange={onJSONChange} ariaLabel={`${title} JSON`} /></Field></TabsContent>
  </Tabs>
}

export function InboundEditorDialog({ title, item, onClose, onSave }: InboundEditorDialogProps) {
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
      <DialogHeader><DialogTitle>{title}</DialogTitle><DialogDescription>{t("proxy.inbound.editorDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">
        {object ? <FormTabs object={object} value={value} title={title} revision={revision} onChange={update} onJSONChange={updateJSON} onFieldValidityChange={updateValidity} /> : <JsonEditor value={value} onChange={updateJSON} ariaLabel={`${title} JSON`} />}
      </div>
      <DialogFooter><Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button><Button disabled={!object || typeof object.type !== "string" || !object.type || invalidFields.size > 0} onClick={() => { if (object) onSave(object) }}>{t("common.save")}</Button></DialogFooter>
    </DialogContent>
  </Dialog>
}
