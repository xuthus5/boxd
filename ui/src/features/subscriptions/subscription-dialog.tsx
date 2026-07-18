import { useMutation } from "@tanstack/react-query"
import { type Dispatch, type SetStateAction, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { api, type SubscriptionInput } from "@/lib/api/endpoints"
import type { Subscription, URLTestDefaults, URLTestOverrides } from "@/lib/api/types"
import { isHTTPURL, isPositiveDuration, isTolerance } from "@/lib/urltest"

type URLTestPolicy = "inherit" | "enabled" | "disabled"

interface URLTestFormState {
  policy: URLTestPolicy
  url: string
  interval: string
  tolerance: string
}

interface URLTestFieldsProps {
  defaults: URLTestDefaults
  state: URLTestFormState
  setState: Dispatch<SetStateAction<URLTestFormState>>
  onReset: () => void
}

function initialURLTestState(item?: Subscription): URLTestFormState {
  const enabled = item?.urltest?.enabled
  const policy = enabled === undefined ? "inherit" : enabled ? "enabled" : "disabled"
  return {
    policy,
    url: item?.urltest?.url ?? "",
    interval: item?.urltest?.interval ?? "",
    tolerance: item?.urltest?.tolerance === undefined ? "" : String(item.urltest.tolerance),
  }
}

function buildURLTestOverrides(state: URLTestFormState): URLTestOverrides | null {
  if (state.policy === "inherit") return null
  if (state.policy === "disabled") return { enabled: false }
  const overrides: URLTestOverrides = { enabled: true }
  if (state.url.trim()) overrides.url = state.url.trim()
  if (state.interval.trim()) overrides.interval = state.interval.trim()
  if (state.tolerance.trim()) overrides.tolerance = Number(state.tolerance)
  return overrides
}

function isURLTestStateInvalid(state: URLTestFormState) {
  if (state.policy !== "enabled") return false
  const isURLInvalid = Boolean(state.url) && !isHTTPURL(state.url)
  const isIntervalInvalid = Boolean(state.interval) && !isPositiveDuration(state.interval)
  const isToleranceInvalid = Boolean(state.tolerance) && !isTolerance(state.tolerance)
  return isURLInvalid || isIntervalInvalid || isToleranceInvalid
}

function URLTestFields({ defaults, state, setState, onReset }: URLTestFieldsProps) {
  const { t } = useTranslation()
  const update = (values: Partial<URLTestFormState>) => setState((current) => ({ ...current, ...values }))
  const inherit = state.policy === "inherit"
  const disabled = state.policy === "disabled"
  const enabled = state.policy === "enabled"
  const isURLInvalid = enabled && Boolean(state.url) && !isHTTPURL(state.url)
  const isIntervalInvalid = enabled && Boolean(state.interval) && !isPositiveDuration(state.interval)
  const isToleranceInvalid = enabled && Boolean(state.tolerance) && !isTolerance(state.tolerance)
  const urlValue = inherit ? defaults.url : state.url
  const intervalValue = inherit ? defaults.interval : state.interval
  const toleranceValue = inherit ? String(defaults.tolerance) : state.tolerance
  return <FieldGroup>
    <Field orientation="responsive">
      <FieldTitle id="subscription-urltest-policy">{t("subscriptions.urlTestPolicy")}</FieldTitle>
      <ToggleGroup
        className="grid w-full grid-cols-3"
        aria-labelledby="subscription-urltest-policy"
        value={[state.policy]}
        onValueChange={(value) => {
          if (!value[0]) return
          update({ policy: value[0] as URLTestPolicy })
        }}
      >
        <ToggleGroupItem className="min-w-0 whitespace-normal" value="inherit">{t("subscriptions.urlTestInherit")}</ToggleGroupItem>
        <ToggleGroupItem className="min-w-0 whitespace-normal" value="enabled">{t("subscriptions.urlTestEnabled")}</ToggleGroupItem>
        <ToggleGroupItem className="min-w-0 whitespace-normal" value="disabled">{t("subscriptions.urlTestDisabled")}</ToggleGroupItem>
      </ToggleGroup>
    </Field>
    {disabled ? <Alert>
      <AlertTitle>{t("subscriptions.urlTestDisabledTitle")}</AlertTitle>
      <AlertDescription>{t("subscriptions.urlTestDisabledDescription")}</AlertDescription>
    </Alert> : null}
    {!disabled ? <>
      <ProbeURLField
        id="subscription-urltest-url"
        label={t("settings.urlTestURL")}
        manualLabel={t("settings.urlTestURLManualInput")}
        value={urlValue}
        onChange={(value) => update({ url: value })}
        disabled={inherit}
        invalid={isURLInvalid}
        allowEmpty={!inherit}
        emptyLabel={t("settings.urlTestURLEmpty")}
        placeholder={defaults.url}
        description={
          isURLInvalid
            ? t("settings.urlTestURLInvalid")
            : inherit
              ? t("subscriptions.urlTestInheritHint")
              : t("subscriptions.leaveBlankToInherit")
        }
      />
      <Field data-invalid={isIntervalInvalid || undefined}>
        <FieldLabel htmlFor="subscription-urltest-interval">{t("settings.urlTestInterval")}</FieldLabel>
        <Input
          id="subscription-urltest-interval"
          disabled={inherit}
          aria-invalid={isIntervalInvalid || undefined}
          placeholder={defaults.interval}
          value={intervalValue}
          onChange={(event) => update({ interval: event.target.value })}
        />
        <FieldDescription>
          {isIntervalInvalid
            ? t("settings.urlTestIntervalInvalid")
            : inherit
              ? t("subscriptions.urlTestInheritHint")
              : t("subscriptions.leaveBlankToInherit")}
        </FieldDescription>
      </Field>
      <Field data-invalid={isToleranceInvalid || undefined}>
        <FieldLabel htmlFor="subscription-urltest-tolerance">{t("settings.urlTestTolerance")}</FieldLabel>
        <Input
          id="subscription-urltest-tolerance"
          type="number"
          min={0}
          max={65535}
          disabled={inherit}
          aria-invalid={isToleranceInvalid || undefined}
          placeholder={String(defaults.tolerance)}
          value={toleranceValue}
          onChange={(event) => update({ tolerance: event.target.value })}
        />
        <FieldDescription>
          {isToleranceInvalid
            ? t("settings.urlTestToleranceInvalid")
            : inherit
              ? t("subscriptions.urlTestInheritHint")
              : t("subscriptions.leaveBlankToInherit")}
        </FieldDescription>
      </Field>
    </> : null}
    {!inherit ? <Field>
      <Button type="button" variant="outline" onClick={onReset}>{t("subscriptions.resetURLTest")}</Button>
    </Field> : null}
  </FieldGroup>
}

interface SubscriptionDialogProps {
  defaults: URLTestDefaults
  item?: Subscription
  onClose: () => void
  onSaved: () => void
}

export function SubscriptionDialog({ defaults, item, onClose, onSaved }: SubscriptionDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(item?.name ?? "")
  const [url, setURL] = useState(item?.url ?? "")
  const [interval, setInterval] = useState(item?.interval_min ?? 60)
  const [urlTest, setURLTest] = useState(() => initialURLTestState(item))
  const request = useMutation({
    mutationFn: async (input: SubscriptionInput) => {
      if (item) await api.subscriptions.update(item.id, input)
      else await api.subscriptions.create(input)
      await api.nodes.sync()
    },
    onSuccess: () => { toast.success(t("subscriptions.saved")); onSaved() },
    onError: (error: Error) => toast.error(error.message),
  })
  const save = () => request.mutate({
    name,
    url,
    interval_min: interval,
    urltest: buildURLTestOverrides(urlTest),
  })
  const resetURLTest = () => setURLTest({ policy: "inherit", url: "", interval: "", tolerance: "" })
  const isInvalid = !name || !url || interval <= 0 || isURLTestStateInvalid(urlTest)
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}>
    <DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{item ? t("subscriptions.edit") : t("subscriptions.add")}</DialogTitle>
        <DialogDescription>{t("subscriptions.dialogDescription")}</DialogDescription>
      </DialogHeader>
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="subscription-name">{t("subscriptions.name")}</FieldLabel>
          <Input id="subscription-name" value={name} onChange={(event) => setName(event.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="subscription-url">{t("subscriptions.subscriptionURL")}</FieldLabel>
          <Input id="subscription-url" value={url} onChange={(event) => setURL(event.target.value)} />
        </Field>
        <Field>
          <FieldLabel htmlFor="subscription-interval">{t("subscriptions.interval")}</FieldLabel>
          <Input id="subscription-interval" type="number" min={1} value={interval} onChange={(event) => setInterval(Number(event.target.value))} />
        </Field>
      </FieldGroup>
      <URLTestFields defaults={defaults} state={urlTest} setState={setURLTest} onReset={resetURLTest} />
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button>
        <Button disabled={isInvalid || request.isPending} onClick={save}>{t("common.save")}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
}
