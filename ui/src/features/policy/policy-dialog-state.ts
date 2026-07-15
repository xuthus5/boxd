import { useCallback, useState } from "react"

import {
  isJsonObject,
  type JsonObject,
  type PolicyFieldTransform,
} from "@/features/policy/policy-form-model"
import type { JsonValue } from "@/lib/api/types"

function parseObject(value: string): JsonObject | null {
  try {
    const parsed = JSON.parse(value) as JsonValue
    return isJsonObject(parsed) ? parsed : null
  } catch (error) {
    void error
    return null
  }
}

export function usePolicyDialogState(item: JsonObject, transformField?: PolicyFieldTransform) {
  const [value, setValue] = useState(() => JSON.stringify(item, null, 2))
  const [object, setObject] = useState(item)
  const [jsonValid, setJSONValid] = useState(true)
  const [editorRevision, setEditorRevision] = useState(0)
  const [revision, setRevision] = useState(0)
  const [invalidFields, setInvalidFields] = useState(() => new Set<string>())
  const update = useCallback((next: JsonObject) => {
    setObject(next)
    setValue(JSON.stringify(next, null, 2))
    setJSONValid(true)
    setEditorRevision((current) => current + 1)
  }, [])
  const updateJSON = useCallback((next: string) => {
    setValue(next)
    const parsed = parseObject(next)
    if (!parsed) {
      setJSONValid(false)
      return
    }
    setObject(parsed)
    setJSONValid(true)
    setRevision((current) => current + 1)
    setInvalidFields(new Set())
  }, [])
  const updateValidity = useCallback((path: string, valid: boolean) => {
    setInvalidFields((current) => {
      const next = new Set(current)
      if (valid) next.delete(path)
      else next.add(path)
      return next
    })
  }, [])
  const transform = useCallback<PolicyFieldTransform>((current, field, raw) => {
    const next = transformField?.(current, field, raw)
    if (next !== undefined) updateValidity(field.path, next !== null)
    return next
  }, [transformField, updateValidity])
  return {
    value, revision, editorRevision, invalidFields, object, jsonValid,
    update, updateJSON, updateValidity, transform,
  }
}
