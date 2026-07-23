import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { useConfigQuery } from "@/features/config/config-hooks"
import {
  applyDNSFakeIPFieldChange,
  applyDNSGlobalFieldChange,
  dnsGlobalFields,
  legacyFakeIPFields,
  transformDNSField,
} from "@/features/policy/dns-form-model"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { policyDNSServerTags } from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"

export function DNSGlobalCard(props: PolicyVisualEditorProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const context = useMemo(() => ({
    dnsServerTags: policyDNSServerTags(config.data?.dns ?? props.object),
  }), [config.data?.dns, props.object])
  return <Card size="sm">
    <CardHeader className="gap-1.5">
      <CardTitle className="truncate">{t("policy.dns.globalTitle")}</CardTitle>
      <CardDescription>{t("policy.dns.globalDescription")}</CardDescription>
    </CardHeader>
    <CardContent>
      <PolicyFormFields
        fields={dnsGlobalFields}
        object={props.object}
        namespace="policy.dns"
        revision={props.revision}
        context={context}
        onChange={(next) => props.onChange(applyDNSGlobalFieldChange(props.object, next))}
        onFieldValidityChange={props.onFieldValidityChange}
        transformField={transformDNSField}
      />
    </CardContent>
    <CardFooter className="flex-wrap justify-between gap-2">
      <p className="text-muted-foreground">{t("policy.dns.globalFooter")}</p>
      {/* c8 ignore next */}
      <Button size="sm" className="h-8" onClick={() => props.onGlobalSave?.(props.object)}>{t("policy.save")}</Button>
    </CardFooter>
  </Card>
}

export function DNSFakeIPCard(props: PolicyVisualEditorProps) {
  const { t } = useTranslation()
  return <Card size="sm">
    <CardHeader className="gap-1.5">
      <CardTitle className="truncate">{t("policy.dns.fakeIPTitle")}</CardTitle>
      <CardDescription>{t("policy.dns.fakeIPDescription")}</CardDescription>
    </CardHeader>
    <CardContent>
      <PolicyFormFields
        fields={legacyFakeIPFields}
        object={props.object}
        namespace="policy.dns"
        revision={props.revision}
        onChange={(next) => props.onChange(applyDNSFakeIPFieldChange(props.object, next))}
        onFieldValidityChange={props.onFieldValidityChange}
      />
    </CardContent>
    <CardFooter>
      <p className="text-muted-foreground">{t("policy.dns.fakeIPFooter")}</p>
    </CardFooter>
  </Card>
}
