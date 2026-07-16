import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import { routeGlobalFields } from "@/features/policy/route-form-model"
import { getPolicyPath, setPolicyPath, type JsonObject } from "@/features/policy/policy-form-model"
import { transformRouteField } from "@/features/policy/route-form-transform"
import type { JsonValue } from "@/lib/api/types"

function outboundTags(value: JsonValue | undefined) {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => item && typeof item === "object" && !Array.isArray(item) && typeof item.tag === "string" ? [item.tag] : [])
}

function FinalOutboundField({ object, outbounds, onChange }: { object: JsonObject; outbounds?: JsonValue; onChange: (object: JsonObject) => void }) {
  const { t } = useTranslation()
  const value = getPolicyPath(object, "final")
  const current = typeof value === "string" ? value : ""
  const tags = outboundTags(outbounds)
  const options = current && !tags.includes(current) ? [current, ...tags] : tags
  const update = (next: string | null) => onChange(setPolicyPath(object, "final", next ?? ""))
  return <Field>
    <FieldLabel htmlFor="route-final-outbound">{t("policy.route.final")}</FieldLabel>
    <Select items={[{ value: null, label: t("policy.route.notSet") }, ...options.map((tag) => ({ value: tag, label: tag }))]} value={current || null} onValueChange={update}>
      <SelectTrigger id="route-final-outbound" aria-label={t("policy.route.final")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup><SelectItem value={null}>{t("policy.route.notSet")}</SelectItem>{options.map((tag) => <SelectItem key={tag} value={tag}>{tag}</SelectItem>)}</SelectGroup></SelectContent>
    </Select>
    <FieldDescription>{t("policy.route.outboundDescription")}</FieldDescription>
  </Field>
}

export function RouteGlobalCard({ object, outbounds, revision, onChange, onFieldValidityChange }: PolicyVisualEditorProps & { outbounds?: JsonValue }) {
  const { t } = useTranslation()
  const fields = routeGlobalFields.filter((field) => field.path !== "final")
  return <Card>
    <CardHeader><CardTitle>{t("policy.route.globalTitle")}</CardTitle>
      <CardDescription>{t("policy.route.globalDescription")}</CardDescription></CardHeader>
    <CardContent><PolicyFormFields fields={fields} leading={<FinalOutboundField object={object} outbounds={outbounds} onChange={onChange} />} object={object} namespace="policy.route"
      revision={revision} onChange={onChange} onFieldValidityChange={onFieldValidityChange}
      transformField={transformRouteField} /></CardContent>
    <CardFooter><p className="text-muted-foreground">{t("policy.route.globalFooter")}</p></CardFooter>
  </Card>
}
