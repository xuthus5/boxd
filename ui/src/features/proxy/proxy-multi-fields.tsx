import { useId, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { api } from "@/lib/api/endpoints"
import type { NetworkInterfaceInfo } from "@/lib/api/types"

type Namespace = "proxy.inbound" | "proxy.outbound"

function SimpleHeading({ id, label }: { id: string; label: string; namespace?: Namespace; labelKey?: string }) {
  return <FieldLabel htmlFor={id}>{label}</FieldLabel>
}

function interfaceLabel(item: NetworkInterfaceInfo) {
  const ips = item.ips?.length ? ` (${item.ips.join(", ")})` : ""
  return `${item.name}${ips}`
}

export function NetworkMultiField({
  label,
  namespace,
  labelKey,
  value,
  onChange,
}: {
  label: string
  namespace: Namespace
  labelKey: string
  value: string[]
  onChange: (value: string[] | undefined) => void
}) {
  const id = useId()
  const options = ["tcp", "udp"]
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return (
    <Field className="sm:col-span-2">
      <SimpleHeading id={id} label={label} namespace={namespace} labelKey={labelKey} />
      <div className="flex flex-wrap gap-6" role="group" aria-label={label}>
        {options.map((option) => (
          <Field key={option} orientation="horizontal" className="w-auto">
            <FieldLabel>{option}</FieldLabel>
            <Switch
              aria-label={`${label} ${option}`}
              checked={value.includes(option)}
              onCheckedChange={(checked) => toggle(option, checked)}
            />
          </Field>
        ))}
      </div>
    </Field>
  )
}

export function InterfaceMultiField({
  label,
  namespace,
  labelKey,
  value,
  onChange,
}: {
  label: string
  namespace: Namespace
  labelKey: string
  value: string[]
  onChange: (value: string[] | undefined) => void
}) {
  const id = useId()
  const query = useQuery({ queryKey: ["network", "interfaces"], queryFn: api.network.interfaces })
  const interfaces = useMemo(() => query.data?.interfaces ?? [], [query.data?.interfaces])
  const options = useMemo(
    () => [...new Set([...interfaces.map((item) => item.name), ...value])],
    [interfaces, value],
  )
  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...new Set([...value, option])] : value.filter((item) => item !== option)
    onChange(next.length ? next : undefined)
  }
  return (
    <Field className="sm:col-span-2">
      <SimpleHeading id={id} label={label} namespace={namespace} labelKey={labelKey} />
      <div className="flex flex-col gap-2" role="group" aria-label={label}>
        {options.length === 0
          ? <p className="text-sm text-muted-foreground">—</p>
          : options.map((option) => {
            const meta = interfaces.find((item) => item.name === option)
            return (
              <label key={option} className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={value.includes(option)}
                  onCheckedChange={(next) => toggle(option, next === true)}
                  aria-label={`${label} ${option}`}
                />
                <span>{meta ? interfaceLabel(meta) : option}</span>
              </label>
            )
          })}
      </div>
    </Field>
  )
}
