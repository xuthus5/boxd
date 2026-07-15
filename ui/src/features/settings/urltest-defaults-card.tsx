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
import type { URLTestDefaults } from "@/lib/api/types"
import { isHTTPURL, isPositiveDuration, isTolerance } from "@/lib/urltest"

export function URLTestDefaultsCard({ defaults }: { defaults: URLTestDefaults }) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(defaults.enabled)
  const [url, setURL] = useState(defaults.url)
  const [interval, setInterval] = useState(defaults.interval)
  const [tolerance, setTolerance] = useState(String(defaults.tolerance))
  const isURLInvalid = !isHTTPURL(url)
  const isIntervalInvalid = !isPositiveDuration(interval)
  const toleranceValue = Number(tolerance)
  const isToleranceInvalid = !isTolerance(tolerance)
  const isInvalid = isURLInvalid || isIntervalInvalid || isToleranceInvalid
  const save = useMutation({
    mutationFn: async () => {
      const input = { enabled, url, interval, tolerance: toleranceValue }
      const result = await api.settings.setURLTestDefaults(input)
      await api.nodes.sync()
      return result
    },
    onSuccess: () => toast.success(t("settings.urlTestDefaultsSaved")),
    onError: (error: Error) => toast.error(error.message),
  })

  return <Card><CardHeader><CardTitle>{t("settings.urlTestDefaultsTitle")}</CardTitle><CardDescription>{t("settings.urlTestDefaultsDescription")}</CardDescription></CardHeader><CardContent><FieldGroup>
    <Field orientation="horizontal"><FieldLabel htmlFor="urltest-enabled">{t("settings.urlTestDefaultsEnabled")}</FieldLabel><Switch id="urltest-enabled" checked={enabled} onCheckedChange={setEnabled} /></Field>
    <Field data-invalid={isURLInvalid || undefined}><FieldLabel htmlFor="urltest-url">{t("settings.urlTestURL")}</FieldLabel><Input id="urltest-url" aria-invalid={isURLInvalid || undefined} value={url} onChange={(event) => setURL(event.target.value)} /><FieldDescription>{isURLInvalid ? t("settings.urlTestURLInvalid") : t("settings.urlTestURLDescription")}</FieldDescription></Field>
    <Field data-invalid={isIntervalInvalid || undefined}><FieldLabel htmlFor="urltest-interval">{t("settings.urlTestInterval")}</FieldLabel><Input id="urltest-interval" aria-invalid={isIntervalInvalid || undefined} value={interval} onChange={(event) => setInterval(event.target.value)} /><FieldDescription>{isIntervalInvalid ? t("settings.urlTestIntervalInvalid") : t("settings.urlTestIntervalDescription")}</FieldDescription></Field>
    <Field data-invalid={isToleranceInvalid || undefined}><FieldLabel htmlFor="urltest-tolerance">{t("settings.urlTestTolerance")}</FieldLabel><Input id="urltest-tolerance" type="number" min={0} max={65535} aria-invalid={isToleranceInvalid || undefined} value={tolerance} onChange={(event) => setTolerance(event.target.value)} /><FieldDescription>{isToleranceInvalid ? t("settings.urlTestToleranceInvalid") : t("settings.urlTestToleranceDescription")}</FieldDescription></Field>
  </FieldGroup></CardContent><CardFooter><Button disabled={isInvalid || save.isPending} onClick={() => save.mutate()}>{t("settings.saveURLTestDefaults")}</Button></CardFooter></Card>
}
