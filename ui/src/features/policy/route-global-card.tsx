import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { routeGlobalFields } from "@/features/policy/route-form-model"
import { transformRouteField } from "@/features/policy/route-form-transform"

export function RouteGlobalCard({ object, revision, onChange, onFieldValidityChange }: PolicyVisualEditorProps) {
  const { t } = useTranslation()
  return <Card>
    <CardHeader><CardTitle>{t("policy.route.globalTitle")}</CardTitle>
      <CardDescription>{t("policy.route.globalDescription")}</CardDescription></CardHeader>
    <CardContent><PolicyFormFields fields={routeGlobalFields} object={object} namespace="policy.route"
      revision={revision} onChange={onChange} onFieldValidityChange={onFieldValidityChange}
      transformField={transformRouteField} /></CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.route.globalFooter")}</p></CardFooter>
  </Card>
}
