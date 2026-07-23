import { useMutation } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { isURLTestDefaultsReady } from "@/features/settings/settings-dirty"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import { api } from "@/lib/api/endpoints"
import type { URLTestDefaults } from "@/lib/api/types"
import { isHTTPURL, isPositiveDuration, isTolerance } from "@/lib/urltest"
import { resolveInitialSpeedTestURL } from "@/lib/speed-test-urls"

export function URLTestDefaultsCard({ defaults }: { defaults: URLTestDefaults }) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(defaults.enabled)
  const [url, setURL] = useState(() => resolveInitialSpeedTestURL(defaults.url))
  const [interval, setInterval] = useState(defaults.interval)
  const [tolerance, setTolerance] = useState(String(defaults.tolerance))
  const isURLInvalid = !isHTTPURL(url)
  const isIntervalInvalid = !isPositiveDuration(interval)
  const toleranceValue = Number(tolerance)
  const isToleranceInvalid = !isTolerance(tolerance)
  const [saved, setSaved] = useState(defaults)
  const ready = isURLTestDefaultsReady({
    enabled,
    url,
    interval,
    tolerance: toleranceValue,
    toleranceInput: tolerance,
  }, saved)
  const save = useMutation({
    mutationFn: async () => {
      const input = { enabled, url: url.trim(), interval: interval.trim(), tolerance: toleranceValue }
      await api.settings.setURLTestDefaults(input)
      await api.nodes.sync()
      return input
    },
    onSuccess: (input) => {
      setSaved(input)
      toast.success(t("settings.urlTestDefaultsSaved"))
    },
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "urltest-defaults",
      fallback: t("settings.urlTestDefaultsFailed"),
    }),
  })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("settings.urlTestDefaultsTitle")}</CardTitle>
        <CardDescription>{t("settings.urlTestDefaultsDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-2 sm:gap-3">
          <Field orientation="horizontal">
            <FieldLabel htmlFor="urltest-enabled">{t("settings.urlTestDefaultsEnabled")}</FieldLabel>
            <Switch id="urltest-enabled" checked={enabled} onCheckedChange={setEnabled} />
          </Field>
          <ProbeURLField
            id="urltest-url"
            label={t("settings.urlTestURL")}
            manualLabel={t("settings.urlTestURLManualInput")}
            value={url}
            onChange={setURL}
            invalid={isURLInvalid}
            description={isURLInvalid ? t("settings.urlTestURLInvalid") : t("settings.urlTestURLDescription")}
          />
          <Field data-invalid={isIntervalInvalid || undefined}>
            <FieldLabel htmlFor="urltest-interval">{t("settings.urlTestInterval")}</FieldLabel>
            <Input
              id="urltest-interval"
              className="h-8"
              aria-invalid={isIntervalInvalid || undefined}
              value={interval}
              onChange={(event) => setInterval(event.target.value)}
            />
            <FieldDescription>
              {isIntervalInvalid ? t("settings.urlTestIntervalInvalid") : t("settings.urlTestIntervalDescription")}
            </FieldDescription>
          </Field>
          <Field data-invalid={isToleranceInvalid || undefined}>
            <FieldLabel htmlFor="urltest-tolerance">{t("settings.urlTestTolerance")}</FieldLabel>
            <Input
              id="urltest-tolerance"
              type="number"
              min={0}
              max={65535}
              className="h-8"
              aria-invalid={isToleranceInvalid || undefined}
              value={tolerance}
              onChange={(event) => setTolerance(event.target.value)}
            />
            <FieldDescription>
              {isToleranceInvalid ? t("settings.urlTestToleranceInvalid") : t("settings.urlTestToleranceDescription")}
            </FieldDescription>
          </Field>
        </FieldGroup>
      </CardContent>
      <CardFooter>
        <Button size="sm" className="h-8 w-full sm:w-auto" disabled={!ready || save.isPending} onClick={() => save.mutate()}>
          {t("settings.saveURLTestDefaults")}
        </Button>
      </CardFooter>
    </Card>
  )
}
