import { TriangleAlertIcon } from "lucide-react"
import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  ntpBasicFields,
  ntpDialerFields,
  transformNTPField,
} from "@/features/advanced/ntp-form-model"
import { useConfigQuery } from "@/features/config/config-hooks"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import {
  policyConfigTags,
  policyDNSServerTags,
  type JsonObject,
} from "@/features/policy/policy-form-model"

export interface NTPVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

function SystemTimeWarning() {
  const { t } = useTranslation()
  return (
    <Alert>
      <TriangleAlertIcon />
      <AlertTitle>{t("advanced.ntp.systemTimeWarningTitle")}</AlertTitle>
      <AlertDescription>{t("advanced.ntp.systemTimeWarningDescription")}</AlertDescription>
    </Alert>
  )
}

export function NTPVisualEditor({ object, revision, onChange, onFieldValidityChange }: NTPVisualEditorProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const context = useMemo(() => ({
    outboundTags: policyConfigTags(config.data?.outbounds),
    dnsServerTags: policyDNSServerTags(config.data?.dns),
  }), [config.data?.dns, config.data?.outbounds])
  const shared = { object, revision, context, onChange, onFieldValidityChange, transformField: transformNTPField }
  return (
    <div className="flex flex-col gap-2 sm:gap-3">
      <Card>
        <CardHeader>
          <CardTitle>{t("advanced.ntp.basicTitle")}</CardTitle>
          <CardDescription>{t("advanced.ntp.basicDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          {object.enabled === true && object.write_to_system === true ? <SystemTimeWarning /> : null}
          <PolicyFormFields fields={ntpBasicFields} namespace="advanced.ntp" {...shared} />
        </CardContent>
      </Card>
      {object.enabled === true ? <Card>
        <CardHeader>
          <CardTitle>{t("advanced.ntp.dialerTitle")}</CardTitle>
          <CardDescription>{t("advanced.ntp.dialerDescription")}</CardDescription>
        </CardHeader>
        <CardContent>
          <PolicyFormFields fields={ntpDialerFields} namespace="advanced.ntp" {...shared} />
        </CardContent>
      </Card> : null}
    </div>
  )
}
