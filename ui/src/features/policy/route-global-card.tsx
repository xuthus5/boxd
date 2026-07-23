import { CircleHelpIcon } from "lucide-react"
import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldLabel } from "@/components/ui/field"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useConfigQuery } from "@/features/config/config-hooks"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import {
  getPolicyPath, policyConfigTags, policyDNSServerTags, setPolicyPath, type JsonObject,
} from "@/features/policy/policy-form-model"
import type { PolicyVisualEditorProps } from "@/features/policy/policy-page"
import {
  applyRouteGlobalFieldChange, managedRouteGlobalFields,
} from "@/features/policy/route-form-model"
import { transformRouteField } from "@/features/policy/route-form-transform"
import type { JsonValue } from "@/lib/api/types"

function outboundTags(value: JsonValue | undefined) {
  if (!Array.isArray(value)) return []
  return value.flatMap((item) => item && typeof item === "object" && !Array.isArray(item) && typeof item.tag === "string" ? [item.tag] : [])
}

function FinalOutboundField({ object, outbounds, onChange }: {
  object: JsonObject; outbounds?: JsonValue; onChange: (object: JsonObject) => void
}) {
  const { t } = useTranslation()
  const value = getPolicyPath(object, "final")
  const current = typeof value === "string" ? value : ""
  const tags = outboundTags(outbounds)
  const options = current && !tags.includes(current) ? [current, ...tags] : tags
  const update = (next: string | null) => onChange(setPolicyPath(object, "final", next || undefined))
  return <Field>
    <div className="flex items-center gap-1.5">
      <FieldLabel htmlFor="route-final-outbound">{t("policy.route.final")}</FieldLabel>
      <Tooltip>
        <TooltipTrigger type="button" className="inline-flex size-4 items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50" aria-label={t("common.fieldHelp")}>
          <CircleHelpIcon className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent className="max-w-xs text-left leading-relaxed">{t("policy.route.finalHelp")}</TooltipContent>
      </Tooltip>
    </div>
    <Select items={[{ value: null, label: t("policy.route.notSet") }, ...options.map((tag) => ({ value: tag, label: tag }))]} value={current || null} onValueChange={update}>
      <SelectTrigger id="route-final-outbound" aria-label={t("policy.route.final")} className="w-full"><SelectValue /></SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value={null}>{t("policy.route.notSet")}</SelectItem>
        {options.map((tag) => <SelectItem key={tag} value={tag}>{tag}</SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
  </Field>
}

export function RouteGlobalCard({
  object, outbounds, revision, onChange, onFieldValidityChange, onGlobalSave,
}: PolicyVisualEditorProps & { outbounds?: JsonValue }) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const fields = useMemo(() => managedRouteGlobalFields(), [])
  const context = useMemo(() => ({
    outboundTags: outboundTags(outbounds).length ? outboundTags(outbounds) : policyConfigTags(config.data?.outbounds),
    dnsServerTags: policyDNSServerTags(config.data?.dns),
  }), [config.data?.dns, config.data?.outbounds, outbounds])
  return <Card size="sm">
    <CardHeader className="gap-1.5">
      <CardTitle className="truncate">{t("policy.route.globalTitle")}</CardTitle>
      <CardDescription>{t("policy.route.globalDescription")}</CardDescription>
    </CardHeader>
    <CardContent>
      <PolicyFormFields
        fields={fields}
        leading={<FinalOutboundField object={object} outbounds={outbounds ?? config.data?.outbounds} onChange={(next) => onChange(applyRouteGlobalFieldChange(object, next))} />}
        object={object}
        namespace="policy.route"
        revision={revision}
        context={context}
        onChange={(next) => onChange(applyRouteGlobalFieldChange(object, next))}
        onFieldValidityChange={onFieldValidityChange}
        transformField={transformRouteField}
      />
    </CardContent>
    <CardFooter className="flex-wrap justify-between gap-2">
      <p className="text-muted-foreground">{t("policy.route.globalFooter")}</p>
      {/* c8 ignore next */}
      <Button size="sm" className="h-8" onClick={() => onGlobalSave?.(object)}>{t("policy.save")}</Button>
    </CardFooter>
  </Card>
}
