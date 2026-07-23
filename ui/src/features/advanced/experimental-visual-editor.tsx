import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  experimentalCacheFields,
  experimentalClashFields,
  experimentalV2RayFields,
} from "@/features/advanced/experimental-form-model"
import { useConfigQuery } from "@/features/config/config-hooks"
import { PolicyFormFields } from "@/features/policy/policy-form-fields"
import { policyConfigTags, type JsonObject } from "@/features/policy/policy-form-model"

export interface ExperimentalVisualEditorProps {
  object: JsonObject
  revision: number
  onChange: (object: JsonObject) => void
  onFieldValidityChange: (path: string, valid: boolean) => void
}

export function ExperimentalVisualEditor({
  object, revision, onChange, onFieldValidityChange,
}: ExperimentalVisualEditorProps) {
  const { t } = useTranslation()
  const config = useConfigQuery()
  const context = useMemo(() => ({
    inboundTags: policyConfigTags(config.data?.inbounds),
    outboundTags: policyConfigTags(config.data?.outbounds),
  }), [config.data?.inbounds, config.data?.outbounds])
  const cards = [
    {
      key: "cache",
      title: t("advanced.experimental.cacheTitle"),
      description: t("advanced.experimental.cacheDescription"),
      fields: experimentalCacheFields,
    },
    {
      key: "clash",
      title: t("advanced.experimental.clashTitle"),
      description: t("advanced.experimental.clashDescription"),
      fields: experimentalClashFields,
    },
    {
      key: "v2ray",
      title: t("advanced.experimental.v2rayTitle"),
      description: t("advanced.experimental.v2rayDescription"),
      fields: experimentalV2RayFields,
    },
  ] as const

  return (
    <div className="flex flex-col gap-2 sm:gap-3">
      {cards.map((card) => (
        <Card key={card.key}>
          <CardHeader>
            <CardTitle>{card.title}</CardTitle>
            <CardDescription>{card.description}</CardDescription>
          </CardHeader>
          <CardContent>
            <PolicyFormFields
              fields={card.fields}
              object={object}
              namespace="advanced.experimental"
              revision={revision}
              context={context}
              onChange={onChange}
              onFieldValidityChange={onFieldValidityChange}
            />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
