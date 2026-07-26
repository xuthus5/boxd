import { useCallback, useMemo, useRef, useState, type RefObject } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useConfigQuery } from "@/features/config/config-hooks"
import { JsonEditor, type JsonEditorHandle } from "@/features/config/json-editor"
import { isValidJSON } from "@/features/config/json-utils"
import { OutboundFormFields } from "@/features/proxy/outbound-form-fields"
import {
  changeOutboundType, dialerFields, dialerTypes, groupFields, groupTypes,
  outboundMultiplexFields, outboundMultiplexTypes, outboundTLSFields, outboundTLSTypes, outboundTransportTypes,
  outboundTypes, protocolFields, serverTypes, transportTypeFields,
} from "@/features/proxy/outbound-form-model"
import { collectOutboundBaseInvalid } from "@/features/proxy/proxy-base-validation"
import { ProxyEditorFooter } from "@/features/proxy/proxy-editor-footer"
import { useProxyItemValidate } from "@/features/proxy/use-proxy-item-validate"
import { useProxyItemPathReveal, useProxyJumpPath } from "@/features/proxy/use-proxy-item-path-reveal"
import type { ConfigSaveErrorState } from "@/features/config/config-save-error"
import { configTags, dnsServerTags, getPath, type JsonObject, setPath } from "@/features/proxy/proxy-form-model"

interface OutboundEditorDialogProps {
  title: string
  item: JsonObject
  index?: number
  onClose: () => void
  onSave: (item: JsonObject) => void
  jumpPath?: string | null
  onJumpPathHandled?: () => void
  reportError?: (error: unknown) => ConfigSaveErrorState
  clearSaveError?: () => void
  saving?: boolean
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
  const invalid = new Set(collectOutboundBaseInvalid(object))
  const showServer = serverTypes.has(type)
  return <FieldGroup className="grid gap-2 sm:grid-cols-2 sm:gap-3">
    <Field data-invalid={invalid.has("tag") || undefined}>
      <FieldLabel htmlFor="outbound-tag">Tag</FieldLabel>
      <Input id="outbound-tag" className="h-8" aria-invalid={invalid.has("tag") || undefined} value={String(object.tag ?? "")} onChange={(event) => onChange(setPath(object, "tag", event.target.value || undefined))} />
      {invalid.has("tag") ? <FieldDescription>{t("proxy.outbound.requiredTag")}</FieldDescription> : null}
    </Field>
    <Field data-invalid={invalid.has("type") || undefined}>
      <FieldLabel htmlFor="outbound-type">{t("common.type")}</FieldLabel>
      <Select items={items} value={type || null} onValueChange={(value) => onChange(changeOutboundType(object, String(value)))}>
        <SelectTrigger id="outbound-type" aria-invalid={invalid.has("type") || undefined} aria-label={t("common.type")} className="h-8 w-full"><SelectValue placeholder={t("proxy.outbound.selectType")} /></SelectTrigger>
        <SelectContent><SelectGroup>{options.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
      {invalid.has("type") ? <FieldDescription>{t("proxy.outbound.requiredType")}</FieldDescription> : null}
    </Field>
    {showServer ? <>
      <Field data-invalid={invalid.has("server") || undefined}>
        <FieldLabel htmlFor="outbound-server">{t("proxy.outbound.server")}</FieldLabel>
        <Input id="outbound-server" className="h-8" aria-invalid={invalid.has("server") || undefined} value={String(object.server ?? "")} onChange={(event) => onChange(setPath(object, "server", event.target.value || undefined))} />
        {invalid.has("server") ? <FieldDescription>{t("proxy.outbound.requiredServer")}</FieldDescription> : null}
      </Field>
      <Field data-invalid={invalid.has("server_port") || undefined}>
        <FieldLabel htmlFor="outbound-port">{t("proxy.outbound.serverPort")}</FieldLabel>
        <Input id="outbound-port" type="number" className="h-8" aria-invalid={invalid.has("server_port") || undefined} value={String(object.server_port ?? "")} onChange={(event) => onChange(setPath(object, "server_port", event.target.value ? Number(event.target.value) : undefined))} />
        {invalid.has("server_port") ? <FieldDescription>{t("proxy.outbound.requiredPort")}</FieldDescription> : null}
      </Field>
    </> : null}
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

function ManagedGroupAlert() {
  const { t } = useTranslation()
  return <Alert>
    <AlertTitle>{t("proxy.outbound.managedGroupTitle")}</AlertTitle>
    <AlertDescription>{t("proxy.outbound.managedGroupDescription")}</AlertDescription>
  </Alert>
}

function GroupFields({ type, object, onChange }: { type: string; object: JsonObject; onChange: (object: JsonObject) => void }) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const members = useMemo(
    () => Array.isArray(object.outbounds) ? object.outbounds.filter((item): item is string => typeof item === "string") : [],
    [object.outbounds],
  )
  const candidates = useMemo(() => {
    /* c8 ignore next */
    if (!Array.isArray(config.data?.outbounds)) return [] as string[]
    return config.data.outbounds
      .map((item) => typeof item === "object" && item && !Array.isArray(item) ? String(item.tag ?? "") : "")
      .filter((tag) => tag && tag !== String(object.tag ?? "") && !members.includes(tag))
  }, [config.data, members, object.tag])
  const setMembers = (next: string[]) => {
    const nextObject = setPath(object, "outbounds", next.length ? next : undefined)
    const currentDefault = typeof nextObject.default === "string" ? nextObject.default : ""
    onChange(currentDefault && !next.includes(currentDefault) ? setPath(nextObject, "default", undefined) : nextObject)
  }
  const candidateItems = useMemo(() => candidates.map((value) => ({ value, label: value })), [candidates])
  const defaultItems = useMemo(() => members.map((value) => ({ value, label: value })), [members])
  return <FieldGroup className="flex flex-col gap-2 sm:gap-3">
    <Field>
      <FieldLabel>{t("proxy.outbound.groupOutbounds")}</FieldLabel>
      <div className="flex flex-wrap gap-1.5 sm:gap-2">
        {members.map((member) => <Badge key={member} variant="secondary" className="cursor-pointer" onClick={() => setMembers(members.filter((item) => item !== member))}>{member}</Badge>)}
        {candidates.length ? null : <span className="text-sm text-muted-foreground">—</span>}
      </div>
    </Field>
    {candidates.length ? <Field>
      <FieldLabel htmlFor="outbound-group-add">{t("proxy.outbound.groupOutbounds")}</FieldLabel>
      <Select items={candidateItems} value={null} onValueChange={(value) => setMembers([...members, String(value)])}>
        <SelectTrigger id="outbound-group-add" aria-label={t("proxy.outbound.groupOutbounds")} className="h-8 w-full"><SelectValue placeholder={t("proxy.outbound.selectType")} /></SelectTrigger>
        <SelectContent><SelectGroup>{candidates.map((value) => <SelectItem key={value} value={value}>{value}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
    </Field> : null}
    {type === "selector" ? <Field>
      <FieldLabel htmlFor="outbound-default">{t("proxy.outbound.groupDefault")}</FieldLabel>
      <Select items={defaultItems} value={typeof object.default === "string" ? object.default : null} onValueChange={(value) => onChange(setPath(object, "default", value ? String(value) : undefined))}>
        <SelectTrigger id="outbound-default" aria-label={t("proxy.outbound.groupDefault")} className="h-8 w-full"><SelectValue placeholder={t("proxy.outbound.notSet")} /></SelectTrigger>
        <SelectContent><SelectGroup>{members.map((member) => <SelectItem key={member} value={member}>{member}</SelectItem>)}</SelectGroup></SelectContent>
      </Select>
    </Field> : null}
    <OutboundFormFields
      fields={groupFields(type).filter((field) => field.path !== "outbounds" && field.path !== "default")}
      object={object}
      type={type}
      onChange={onChange}
    />
  </FieldGroup>
}

function FormTabs({ object, value, title, revision, activeTab, onTabChange, editorRef, onChange, onJSONChange, onFieldValidityChange }: FormTabsProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const type = String(object.type ?? "")
  const transportType = String(getPath(object, "transport.type") ?? "")
  const protocol = groupTypes.has(type) ? groupFields(type) : protocolFields(type)
  const hasTransport = outboundTransportTypes.has(type) || outboundMultiplexTypes.has(type)
  const currentTag = String(object.tag ?? "")
  const context = {
    currentTag,
    outboundTags: configTags(config.data?.outbounds, currentTag),
    dnsServerTags: dnsServerTags(config.data?.dns),
  }
  return <Tabs value={activeTab} onValueChange={(next) => onTabChange(String(next || "basic"))} className="min-h-0 min-w-0">
    <TabsList activateOnFocus className="h-auto max-w-full justify-start overflow-x-auto overflow-y-hidden" variant="line">
      <TabsTrigger value="basic">{t("proxy.outbound.basic")}</TabsTrigger>
      {dialerTypes.has(type) ? <TabsTrigger value="dialer">{t("proxy.outbound.dialing")}</TabsTrigger> : null}
      <TabsTrigger value="protocol">{t(groupTypes.has(type) ? "proxy.outbound.group" : "proxy.outbound.protocol")}</TabsTrigger>
      {outboundTLSTypes.has(type) ? <TabsTrigger value="tls">{t("proxy.outbound.tlsReality")}</TabsTrigger> : null}
      {hasTransport ? <TabsTrigger value="transport">{t("proxy.outbound.transportMultiplex")}</TabsTrigger> : null}
      <TabsTrigger value="advanced">{t("proxy.advancedJSON")}</TabsTrigger>
    </TabsList>
    <TabsContent value="basic" className="pt-3 sm:pt-4"><BaseFields object={object} onChange={onChange} /></TabsContent>
    {dialerTypes.has(type) ? <TabsContent value="dialer" className="pt-3 sm:pt-4">
      <OutboundFormFields fields={dialerFields} object={object} type={type} context={context} onChange={onChange} />
    </TabsContent> : null}
    <TabsContent value="protocol" className="pt-3 sm:pt-4" keepMounted>
      <FieldGroup>
        {groupTypes.has(type) ? <ManagedGroupAlert /> : null}
        {groupTypes.has(type)
          ? <GroupFields type={type} object={object} onChange={onChange} />
          : <OutboundFormFields fields={protocol} object={object} type={type} revision={revision} context={context} onChange={onChange} onFieldValidityChange={onFieldValidityChange} />}
      </FieldGroup>
    </TabsContent>
    {outboundTLSTypes.has(type) ? <TabsContent value="tls" className="pt-3 sm:pt-4">
      <OutboundFormFields fields={outboundTLSFields} object={object} type={type} context={context} onChange={onChange} />
    </TabsContent> : null}
    {hasTransport ? <TabsContent value="transport" className="pt-3 sm:pt-4" keepMounted>
      <OutboundFormFields
        fields={[...(outboundTransportTypes.has(type) ? transportTypeFields(transportType) : []), ...(outboundMultiplexTypes.has(type) ? outboundMultiplexFields : [])]}
        object={object}
        type={type}
        revision={revision}
        context={context}
        onChange={onChange}
        onFieldValidityChange={onFieldValidityChange}
      />
    </TabsContent> : null}
    <TabsContent value="advanced" className="pt-3 sm:pt-4" keepMounted>
      <Field>
        <FieldLabel className="sr-only">{t("proxy.advancedJSON")}</FieldLabel>
        <JsonEditor ref={editorRef} value={value} onChange={onJSONChange} ariaLabel={`${title} JSON`} />
      </Field>
    </TabsContent>
  </Tabs>
}

export function OutboundEditorDialog({ title, item, index = -1, onClose, onSave, jumpPath, onJumpPathHandled, reportError, clearSaveError, saving = false }: OutboundEditorDialogProps) {
  const { t } = useTranslation()
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState<Set<string>>(() => new Set())
  const [activeTab, setActiveTab] = useState("basic")
  const editorRef = useRef<JsonEditorHandle>(null)
  const object = parseObject(value)
  const baseInvalid = object ? collectOutboundBaseInvalid(object).length > 0 : true
  const update = (next: JsonObject) => setValue(JSON.stringify(next, null, 2))
  const updateJSON = (next: string) => { setValue(next); setRevision((current) => current + 1) }
  const updateValidity = useCallback((path: string, valid: boolean) => setInvalidFields((current) => {
    const next = new Set(current); if (valid) next.delete(path); else next.add(path); return next
  }), [])
  const canSave = Boolean(object && !baseInvalid && invalidFields.size === 0)
  const reveal = useProxyItemPathReveal(editorRef, "outbounds", index, setActiveTab)
  const { validating, validate, ready } = useProxyItemValidate({
    kind: "outbounds",
    index,
    object,
    reportError,
    clearSaveError,
    onReportedError: (err) => { if (err.path) reveal(err.path) },
  })
  useProxyJumpPath(jumpPath, onJumpPathHandled, reveal)
  return <Dialog open onOpenChange={(open) => { if (!open && !saving) onClose() }}>
    <DialogContent className="max-h-[calc(100dvh-1rem)] min-w-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-3 p-3 sm:max-h-[calc(100dvh-2rem)] sm:max-w-5xl sm:gap-4 sm:p-4">
      <DialogHeader>
        <DialogTitle className="truncate">{title}</DialogTitle>
        <DialogDescription>{t("proxy.outbound.editorDescription")}</DialogDescription>
      </DialogHeader>
      <div className="min-h-0 overflow-y-auto pr-1">
        <div className="flex flex-col gap-2 sm:gap-3">
          {baseInvalid ? <Alert variant="destructive"><AlertTitle>{t("proxy.outbound.requiredTitle")}</AlertTitle><AlertDescription>{t("proxy.outbound.requiredDescription")}</AlertDescription></Alert> : null}
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
        saving={saving}
      />
    </DialogContent>
  </Dialog>
}
