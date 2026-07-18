import { PlusIcon, Trash2Icon } from "lucide-react"
import { useId } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { userSchema, type UserFieldKey } from "@/features/proxy/inbound-form-model"
import type { JsonObject } from "@/features/proxy/proxy-form-model"
import type { JsonValue } from "@/lib/api/types"

interface InboundUsersFieldProps {
  label: string
  type: string
  value: JsonValue | undefined
  onChange: (value: JsonValue | undefined) => void
  onFieldValidityChange?: (path: string, valid: boolean) => void
  path?: string
}

function asUsers(value: JsonValue | undefined): JsonObject[] | null {
  if (value === undefined) return []
  if (!Array.isArray(value)) return null
  if (!value.every((item) => item && typeof item === "object" && !Array.isArray(item))) return null
  return value as JsonObject[]
}

function emptyUser(keys: UserFieldKey[]): JsonObject {
  return Object.fromEntries(keys.map((key) => [key, ""]))
}

function fieldLabel(t: (key: string) => string, key: UserFieldKey) {
  if (key === "alterId") return t("proxy.inbound.alterId")
  if (key === "flow") return t("proxy.inbound.flow")
  if (key === "uuid") return t("proxy.inbound.uuid")
  if (key === "username") return t("proxy.inbound.username")
  if (key === "password") return t("proxy.inbound.password")
  return t("proxy.inbound.userName")
}

export function InboundUsersField({ label, type, value, onChange, onFieldValidityChange, path = "users" }: InboundUsersFieldProps) {
  const { t } = useTranslation()
  const id = useId()
  const keys = userSchema(type)
  const users = asUsers(value)

  if (users === null) {
    const raw = value === undefined ? "" : JSON.stringify(value, null, 2)
    return <Field className="sm:col-span-2">
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Textarea id={id} aria-label={label} value={raw} onChange={(event) => {
        const next = event.target.value.trim()
        if (!next) {
          onFieldValidityChange?.(path, true)
          onChange(undefined)
          return
        }
        try {
          const parsed: unknown = JSON.parse(next)
          const valid = Array.isArray(parsed) && parsed.every((item) => item && typeof item === "object" && !Array.isArray(item))
          onFieldValidityChange?.(path, valid)
          if (valid) onChange(parsed as JsonValue)
        } catch {
          onFieldValidityChange?.(path, false)
        }
      }} />
      <FieldDescription>{t("proxy.inbound.usersJSONHint")}</FieldDescription>
    </Field>
  }

  const updateAt = (index: number, key: UserFieldKey, nextValue: string) => {
    const next = users.map((user, userIndex) => {
      if (userIndex !== index) return user
      const copy = { ...user }
      if (!nextValue) delete copy[key]
      else copy[key] = key === "alterId" ? Number(nextValue) : nextValue
      return copy
    })
    onFieldValidityChange?.(path, true)
    onChange(next.length ? next : undefined)
  }

  const removeAt = (index: number) => {
    const next = users.filter((_, userIndex) => userIndex !== index)
    onChange(next.length ? next : undefined)
  }

  return <Field className="sm:col-span-2">
    <div className="flex items-center justify-between gap-2">
      <FieldLabel>{label}</FieldLabel>
      <Button type="button" size="xs" variant="outline" onClick={() => onChange([...(users ?? []), emptyUser(keys)])}>
        <PlusIcon data-icon="inline-start" />{t("proxy.inbound.addUser")}
      </Button>
    </div>
    {users.length === 0 ? <p className="text-sm text-muted-foreground">{t("proxy.inbound.usersEmpty")}</p> : null}
    <div className="flex flex-col gap-3">
      {users.map((user, index) => <div key={`${index}-${keys.join("-")}`} className="grid gap-3 rounded-lg border p-3 sm:grid-cols-2">
        {keys.map((key) => {
          const inputId = `${id}-${index}-${key}`
          const current = user[key]
          const text = current === undefined || current === null ? "" : String(current)
          return <Field key={key}>
            <FieldLabel htmlFor={inputId}>{fieldLabel(t, key)}</FieldLabel>
            <Input id={inputId} aria-label={`${label} ${index + 1} ${fieldLabel(t, key)}`} type={key === "alterId" ? "number" : "text"} value={text} onChange={(event) => updateAt(index, key, event.target.value)} />
          </Field>
        })}
        <div className="sm:col-span-2 flex justify-end">
          <Button type="button" size="xs" variant="destructive" onClick={() => removeAt(index)}>
            <Trash2Icon data-icon="inline-start" />{t("proxy.inbound.removeUser")}
          </Button>
        </div>
      </div>)}
    </div>
  </Field>
}
