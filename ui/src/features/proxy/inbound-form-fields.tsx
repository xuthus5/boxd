import { useCallback } from "react"

import { applyInboundFieldChange, changeTransportType, type FieldSpec, type JsonObject } from "@/features/proxy/inbound-form-model"
import { ProxyFormFields } from "@/features/proxy/proxy-form-fields"
import type { FormFieldContext } from "@/features/proxy/proxy-form-model"

interface InboundFormFieldsProps {
  fields: FieldSpec[]
  object: JsonObject
  type: string
  revision?: number
  context?: FormFieldContext
  onChange: (object: JsonObject) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
}

export function InboundFormFields({ fields, object, type, revision, context, onChange, onFieldValidityChange }: InboundFormFieldsProps) {
  const transformField = useCallback((current: JsonObject, field: FieldSpec, raw: string) => {
    if (field.path === "transport.type") return applyInboundFieldChange(current, changeTransportType(current, raw), type)
    return undefined
  }, [type])
  return <ProxyFormFields
    fields={fields}
    object={object}
    namespace="proxy.inbound"
    revision={revision}
    context={{ ...context, inboundType: type }}
    onChange={(next) => onChange(applyInboundFieldChange(object, next, type))}
    onFieldValidityChange={onFieldValidityChange}
    transformField={transformField}
  />
}
