import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { SetupModules } from "@/features/setup/setup-modules"
import type { SetupInput, SetupStatus } from "@/lib/api/setup"

export interface SetupOptionsProps {
  status: SetupStatus
  input: SetupInput
  disabled: boolean
  onChange: (input: SetupInput) => void
}

function SetupMode({ status, input, disabled, onChange }: SetupOptionsProps) {
  const { t } = useTranslation()
  const inactive = disabled || !input.modules.includes("inbounds")
  return <Field data-disabled={inactive}>
    <FieldLabel id="setup-mode-label">{t("setup.mode")}</FieldLabel>
    <ToggleGroup className="grid w-full grid-cols-1 sm:grid-cols-3" value={[input.inbound_mode]} disabled={inactive} aria-labelledby="setup-mode-label"
      onValueChange={(value) => { if (value[0]) onChange({ ...input, inbound_mode: value[0] as SetupInput["inbound_mode"] }) }}>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="proxy">{t("setup.proxy")}</ToggleGroupItem>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="tun" disabled={!status.capabilities.tun_available}>{t("setup.tun")}</ToggleGroupItem>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="preserve">{t("setup.preserve")}</ToggleGroupItem>
    </ToggleGroup>
    <FieldDescription>{t(`setup.${input.inbound_mode}Hint`)}</FieldDescription>
  </Field>
}

function SetupIPv6({ status, input, disabled, onChange }: SetupOptionsProps) {
  const { t } = useTranslation()
  return <Field data-disabled={disabled}>
    <FieldLabel id="setup-ipv6-label">{t("setup.ipv6")}</FieldLabel>
    <ToggleGroup className="grid w-full grid-cols-1 sm:grid-cols-3" value={[input.ipv6]} disabled={disabled} aria-labelledby="setup-ipv6-label"
      onValueChange={(value) => { if (value[0]) onChange({ ...input, ipv6: value[0] as SetupInput["ipv6"] }) }}>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="auto">{t("setup.ipv6Auto")}</ToggleGroupItem>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="off">{t("setup.ipv6Off")}</ToggleGroupItem>
      <ToggleGroupItem className="min-w-0 whitespace-normal" value="on"
        disabled={!status.capabilities.ipv6_available && status.capabilities.ipv6_reason !== "ipv6_unknown"}>{t("setup.ipv6On")}</ToggleGroupItem>
    </ToggleGroup>
  </Field>
}

function SetupReset({ status, input, disabled, onChange }: SetupOptionsProps) {
  const { t } = useTranslation()
  if (!status.config_error) return null
  return <Alert variant="destructive">
    <AlertTitle>{t("setup.invalidConfig")}</AlertTitle>
    <AlertDescription className="flex flex-col gap-2">
      <p>{status.config_error}</p><p>{t("setup.resetDescription")}</p>
      <Field orientation="horizontal" data-disabled={disabled}>
        <Checkbox id="setup-reset" checked={Boolean(input.reset_invalid)} disabled={disabled}
          onCheckedChange={(checked) => onChange({ ...input, reset_invalid: checked === true })} />
        <FieldLabel htmlFor="setup-reset">{t("setup.resetConsent")}</FieldLabel>
      </Field>
    </AlertDescription>
  </Alert>
}

export function SetupOptions(props: SetupOptionsProps) {
  return <FieldGroup className="gap-4">
    <SetupReset {...props} />
    <SetupModules modules={props.status.modules} selected={props.input.modules} disabled={props.disabled}
      onChange={(modules) => props.onChange({ ...props.input, modules,
        inbound_mode: modules.includes("inbounds") ? props.input.inbound_mode : "preserve" })} />
    <SetupMode {...props} /><SetupIPv6 {...props} />
  </FieldGroup>
}
