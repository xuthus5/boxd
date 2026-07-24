import { useCallback, useMemo, useRef, useState, type RefObject } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useConfigQuery } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import { InboundFormFields } from "@/features/proxy/inbound-form-fields"
import {
  changeInboundType, getPath, inboundTypes, listenFields, multiplexFields, multiplexTypes, protocolFields, tlsFields, tlsTypes,
  transportTypeFields, transportTypes, tunFields, type JsonObject,
} from "@/features/proxy/inbound-form-model"
import { collectInboundBaseInvalid } from "@/features/proxy/proxy-base-validation"
import { ProxyEditorFooter } from "@/features/proxy/proxy-editor-footer"
import { useProxyItemValidate } from "@/features/proxy/use-proxy-item-validate"
import { useProxyItemPathReveal, useProxyJumpPath } from "@/features/proxy/use-proxy-item-path-reveal"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { configTags, dnsServerTags } from "@/features/proxy/proxy-form-model"

interface InboundEditorDialogProps {
  title: string
  item: JsonObject
  index?: number
  onClose: () => void
  onSave: (item: JsonObject) => void
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
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
  const invalid = new Set(collectInboundBaseInvalid(object))
  return <FieldGroup className="grid gap-2 sm:grid-cols-2 sm:gap-3">
    <Field data-invalid={invalid.has("tag") || undefined}>
      <FieldLabel htmlFor="inbound-tag">Tag</FieldLabel>
      <Input id="inbound-tag" className="h-8" aria-invalid={invalid.has("tag") || undefined} value={String(object.tag ?? "")} onChange={(event) => { const next = { ...object }; const tag = event.target.value.trim(); if (tag) next.tag = tag; else delete next.tag; onChange(next) }} />
      {invalid.has("tag") ? <FieldDescription>{t("proxy.inbound.requiredTag")}</FieldDescription> : null}
    </Field>
    <Field data-invalid={invalid.has("type") || undefined}>
      <FieldLabel htmlFor="inbound-type">{t("common.type")}</FieldLabel>
      <Select items={items} value={type || null} onValueChange={(value) => onChange(changeInboundType(object, String(value)))}>
        <SelectTrigger id="inbound-type" aria-invalid={invalid.has("type") || undefined} aria-label={t("common.type")} className="h-8 w-full"><SelectValue placeholder={t("proxy.inbound.selectType")} /></SelectTrigger>
        <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
      {invalid.has("type") ? <FieldDescription>{t("proxy.inbound.requiredType")}</FieldDescription> : null}
    </Field>
    <div className="sm:col-span-2">
      {type === "tun"
        ? <InboundFormFields fields={tunFields.slice(0, 4)} object={object} type={type} onChange={onChange} />
        : <InboundFormFields fields={listenFields.slice(0, 2)} object={object} type={type} onChange={onChange} />}
      {invalid.has("listen_port") ? <FieldError>{t("proxy.inbound.requiredPort")}</FieldError> : null}
    </div>
  </FieldGroup>
}

interface FormTabsProps {
  object: JsonObject
  value: string
  title: string
  revision: number
  activeTab: string
  onTabChange: (value: string) => void
  editorRef: RefObject<JsonEditorHandle | null>
  onChange: (object: JsonObject) => void
  onJSONChange: (value: string) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

function FormTabs({
  object, value, title, revision, activeTab, onTabChange, editorRef,
  onChange, onJSONChange, onFieldValidityChange,
}: FormTabsProps) {
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
  return <Tabs value={activeTab} onValueChange={(next) => onTabChange(String(next || "basic"))} className="min-h-0 min-w-0">
    <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
      <TabsTrigger value="basic">{t("proxy.inbound.basic")}</TabsTrigger>
      <TabsTrigger value="listen">{t(type === "tun" ? "proxy.inbound.tun" : "proxy.inbound.listenAndConnection")}</TabsTrigger>
      <TabsTrigger value="protocol">{t("proxy.inbound.protocol")}</TabsTrigger>
      {hasTLS ? <TabsTrigger value="tls">{t("proxy.inbound.tlsReality")}</TabsTrigger> : null}
      {hasTransport ? <TabsTrigger value="transport">{t("proxy.inbound.transportMultiplex")}</TabsTrigger> : null}
      <TabsTrigger value="advanced">{t("proxy.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-3 sm:pt-4"><BaseFields object={object} onChange={onChange} /></TabsContent>
    <TabsContent value="listen" className="pt-3 sm:pt-4"><InboundFormFields fields={type === "tun" ? tunFields.slice(4) : listenFields.slice(2)} object={object} type={type} context={context} onChange={onChange} /></TabsContent>
    <TabsContent value="protocol" className="pt-3 sm:pt-4" keepMounted><InboundFormFields fields={protocolFields(type)} object={object} type={type} revision={revision} context={context} onChange={onChange} onFieldValidityChange={onFieldValidityChange} /></TabsContent>
    {hasTLS ? <TabsContent value="tls" className="pt-3 sm:pt-4"><InboundFormFields fields={tlsFields} object={object} type={type} context={context} onChange={onChange} /></TabsContent> : null}
    {hasTransport ? <TabsContent value="transport" className="pt-3 sm:pt-4" keepMounted><InboundFormFields fields={[...(transportTypes.has(type) ? transportTypeFields(transportType) : []), ...(multiplexTypes.has(type) ? multiplexFields : [])]} object={object} type={type} revision={revision} context={context} onChange={onChange} onFieldValidityChange={onFieldValidityChange} /></TabsContent> : null}
    <TabsContent value="advanced" className="pt-3 sm:pt-4" keepMounted><Field><FieldLabel className="sr-only">{t("proxy.advancedJSON")}</FieldLabel><JsonEditor ref={editorRef} value={value} onChange={onJSONChange} ariaLabel={`${title} JSON`} /></Field></TabsContent>
  </Tabs>
}

export function InboundEditorDialog({ title, item, index = -1, onClose, onSave, jumpPath, onJumpPathHandled, reportError, clearSaveError }: InboundEditorDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState<Set<string>>(() => new Set())
  const [activeTab, setActiveTab] = useState("basic")
  const editorRef = useRef<JsonEditorHandle>(null)
  const object = parseObject(value)
  const baseInvalid = object ? collectInboundBaseInvalid(object).length > 0 : true
  const update = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setValue(next); setRevision((current) => current + 1) }
  const updateValidity = useCallback((path: string, valid: boolean) => setInvalidFields((current) => { const next = new Set(current); if (valid) next.delete(path); else next.add(path); return next }), [])
  const canSave = Boolean(object && !baseInvalid && invalidFields.size === 0)
  const reveal = useProxyItemPathReveal(editorRef, "inbounds", index, setActiveTab)
  const { validating, validate, ready } = useProxyItemValidate({
    kind: "inbounds",
    index,
    object,
    reportError,
    clearSaveError,
    onReportedError: (err) => { if (err.path) reveal(err.path) },
  })
  useProxyJumpPath(jumpPath, onJumpPathHandled, reveal)
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
    <DialogContent className="max-h-[calc(100dvh-1rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 p-3 sm:max-h-[calc(100dvh-2rem)] sm:max-w-5xl sm:gap-4 sm:p-4">
      <DialogHeader><DialogTitle className="truncate">{title}</DialogTitle><DialogDescription>{t("proxy.inbound.editorDescription")}</DialogDescription></DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">
        <div className="flex flex-col gap-2 sm:gap-3">
          {baseInvalid ? <Alert variant="destructive"><AlertTitle>{t("proxy.inbound.requiredTitle")}</AlertTitle><AlertDescription>{t("proxy.inbound.requiredDescription")}</AlertDescription></Alert> : null}
          {object
            ? <FormTabs
              object={object}
              value={value}
              title={title}
              revision={revision}
              activeTab={activeTab}
              onTabChange={setActiveTab}
              editorRef={editorRef}
              onChange={update}
              onJSONChange={updateJSON}
              onFieldValidityChange={updateValidity}
            />
            : <JsonEditor ref={editorRef} value={value} onChange={updateJSON} ariaLabel={`${title} JSON`} />}
        </div>
      </div>
      <ProxyEditorFooter
        canSave={canSave}
        canValidate={canSave && ready}
        validating={validating}
        onClose={onClose}
        onSave={() => { if (object) onSave(object) }}
        onValidate={() => { void validate() }}
      />
    </DialogContent>
  </Dialog>
}
