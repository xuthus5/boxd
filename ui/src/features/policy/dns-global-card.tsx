import { useId } from "react"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Switch } from "@/components/ui/switch"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { dnsGlobalFields, legacyFakeIPFields, transformDNSField } from "@/features/policy/dns-form-model"
import { getPolicyPath, setPolicyPath } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"

export function DNSGlobalCard(props: PolicyVisualEditorProps) {
  const { t } = useTranslation()
  return <Card><CardHeader><CardTitle>{t("policy.dns.globalTitle")}</CardTitle>
    <CardDescription>{t("policy.dns.globalDescription")}</CardDescription></CardHeader>
    <CardContent><PolicyFormFields fields={dnsGlobalFields} object={props.object} namespace="policy.dns"
      revision={props.revision} onChange={props.onChange} onFieldValidityChange={props.onFieldValidityChange}
      transformField={transformDNSField} /></CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.dns.globalFooter")}</p></CardFooter>
  </Card>
}

export function DNSFakeIPCard(props: PolicyVisualEditorProps) {
  const { t } = useTranslation()
  const id = useId()
  const enabled = getPolicyPath(props.object, "fakeip.enabled") === true
  const updateEnabled = (checked: boolean) => props.onChange(setPolicyPath(
    props.object,
    "fakeip.enabled",
    checked ? true : undefined,
  ))
  return <Card><CardHeader><CardTitle>{t("policy.dns.fakeIPTitle")}</CardTitle>
    <CardDescription>{t("policy.dns.fakeIPDescription")}</CardDescription></CardHeader>
    <CardContent><FieldGroup className="gap-4"><Field orientation="horizontal">
      <FieldLabel htmlFor={id}>{t("policy.dns.fakeIPEnabled")}</FieldLabel>
      <Switch id={id} aria-label={t("policy.dns.fakeIPEnabled")} checked={enabled} onCheckedChange={updateEnabled} />
    </Field><PolicyFormFields fields={legacyFakeIPFields.slice(1)} object={props.object} namespace="policy.dns"
      revision={props.revision} onChange={props.onChange} onFieldValidityChange={props.onFieldValidityChange} />
    </FieldGroup></CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.dns.fakeIPFooter")}</p></CardFooter>
  </Card>
}
