import { useCallback } from "react"

import { changeTransportType, type FieldSpec, type JsonObject } from "@/features/proxy/inbound-form-model"
import { ProxyFormFields } from "@/features/proxy/proxy-form-fields"

interface InboundFormFieldsProps {
  fields: FieldSpec[]
  object: JsonObject
  type: string
  revision?: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
}

export function InboundFormFields({ fields, object, revision, onChange, onFieldValidityChange }: InboundFormFieldsProps) {
  const transformField = useCallback((current: JsonObject, field: FieldSpec, raw: string) => field.path === "transport.type" ? changeTransportType(current, raw) : undefined, [])
  return <ProxyFormFields fields={fields} object={object} namespace="proxy.inbound" revision={revision} onChange={onChange} onFieldValidityChange={onFieldValidityChange} transformField={transformField} />
}
