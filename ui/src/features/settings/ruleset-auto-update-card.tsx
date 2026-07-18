import { useMutation } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { api } from "@/lib/api/endpoints"
import type { RuleSetAutoUpdate } from "@/lib/api/types"

export function RuleSetAutoUpdateCard({ defaults }: { defaults: RuleSetAutoUpdate }) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(defaults.enabled)
  const [interval, setInterval] = useState(defaults.interval || "24h")
  const invalid = !interval.trim() || !/^\d+(\.\d+)?(ns|us|µs|ms|s|m|h)$/.test(interval.trim())
  const save = useMutation({
    mutationFn: () => api.config.setRuleSetsAutoUpdate({ enabled, interval: interval.trim() }),
    onSuccess: () => toast.success(t("settings.ruleSetAutoUpdateSaved")),
    onError: (error: Error) => toast.error(error.message),
  })
  return <Card>
    <CardHeader>
      <CardTitle>{t("settings.ruleSetAutoUpdateTitle")}</CardTitle>
      <CardDescription>{t("settings.ruleSetAutoUpdateDescription")}</CardDescription>
    </CardHeader>
    <CardContent>
      <FieldGroup>
        <Field orientation="horizontal">
          <FieldLabel htmlFor="ruleset-auto-enabled">{t("settings.ruleSetAutoUpdateEnabled")}</FieldLabel>
          <Switch id="ruleset-auto-enabled" checked={enabled} onCheckedChange={setEnabled} />
        </Field>
        <Field data-invalid={invalid || undefined}>
          <FieldLabel htmlFor="ruleset-auto-interval">{t("settings.ruleSetAutoUpdateInterval")}</FieldLabel>
          <Input id="ruleset-auto-interval" value={interval} onChange={(event) => setInterval(event.target.value)} placeholder="24h" aria-invalid={invalid || undefined} />
          <FieldDescription>{invalid ? t("settings.urlTestIntervalInvalid") : "24h / 12h / 1h"}</FieldDescription>
        </Field>
      </FieldGroup>
    </CardContent>
    <CardFooter>
      <Button disabled={invalid || save.isPending} onClick={() => save.mutate()}>{t("common.save")}</Button>
    </CardFooter>
  </Card>
}
