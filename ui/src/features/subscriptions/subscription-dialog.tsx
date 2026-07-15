import { useMutation } from "@tanstack/react-query"
import { type Dispatch, type SetStateAction, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

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
  const overrides: URLTestOverrides = {}
  if (state.policy !== "inherit") overrides.enabled = state.policy === "enabled"
  if (state.url.trim()) overrides.url = state.url.trim()
  if (state.interval.trim()) overrides.interval = state.interval.trim()
  if (state.tolerance.trim()) overrides.tolerance = Number(state.tolerance)
  return Object.keys(overrides).length ? overrides : null
}

function isURLTestStateInvalid(state: URLTestFormState) {
  const isURLInvalid = Boolean(state.url) && !isHTTPURL(state.url)
  const isIntervalInvalid = Boolean(state.interval) && !isPositiveDuration(state.interval)
  const isToleranceInvalid = Boolean(state.tolerance) && !isTolerance(state.tolerance)
  return isURLInvalid || isIntervalInvalid || isToleranceInvalid
}

function URLTestFields({ defaults, state, setState, onReset }: URLTestFieldsProps) {
  const { t } = useTranslation()
  const update = (values: Partial<URLTestFormState>) => setState((current) => ({ ...current, ...values }))
  const isURLInvalid = Boolean(state.url) && !isHTTPURL(state.url)
  const isIntervalInvalid = Boolean(state.interval) && !isPositiveDuration(state.interval)
  const isToleranceInvalid = Boolean(state.tolerance) && !isTolerance(state.tolerance)
  return <FieldGroup>
    <Field orientation="responsive"><FieldTitle id="subscription-urltest-policy">{t("subscriptions.urlTestPolicy")}</FieldTitle><ToggleGroup className="grid w-full grid-cols-3" aria-labelledby="subscription-urltest-policy" value={[state.policy]} onValueChange={(value) => { if (value[0]) update({ policy: value[0] as URLTestPolicy }) }}><ToggleGroupItem className="min-w-0 whitespace-normal" value="inherit">{t("subscriptions.urlTestInherit")}</ToggleGroupItem><ToggleGroupItem className="min-w-0 whitespace-normal" value="enabled">{t("subscriptions.urlTestEnabled")}</ToggleGroupItem><ToggleGroupItem className="min-w-0 whitespace-normal" value="disabled">{t("subscriptions.urlTestDisabled")}</ToggleGroupItem></ToggleGroup></Field>
    <Field data-invalid={isURLInvalid || undefined}><FieldLabel htmlFor="subscription-urltest-url">{t("settings.urlTestURL")}</FieldLabel><Input id="subscription-urltest-url" aria-invalid={isURLInvalid || undefined} placeholder={defaults.url} value={state.url} onChange={(event) => update({ url: event.target.value })} /><FieldDescription>{isURLInvalid ? t("settings.urlTestURLInvalid") : t("subscriptions.leaveBlankToInherit")}</FieldDescription></Field>
    <Field data-invalid={isIntervalInvalid || undefined}><FieldLabel htmlFor="subscription-urltest-interval">{t("settings.urlTestInterval")}</FieldLabel><Input id="subscription-urltest-interval" aria-invalid={isIntervalInvalid || undefined} placeholder={defaults.interval} value={state.interval} onChange={(event) => update({ interval: event.target.value })} /><FieldDescription>{isIntervalInvalid ? t("settings.urlTestIntervalInvalid") : t("subscriptions.leaveBlankToInherit")}</FieldDescription></Field>
    <Field data-invalid={isToleranceInvalid || undefined}><FieldLabel htmlFor="subscription-urltest-tolerance">{t("settings.urlTestTolerance")}</FieldLabel><Input id="subscription-urltest-tolerance" type="number" min={0} max={65535} aria-invalid={isToleranceInvalid || undefined} placeholder={String(defaults.tolerance)} value={state.tolerance} onChange={(event) => update({ tolerance: event.target.value })} /><FieldDescription>{isToleranceInvalid ? t("settings.urlTestToleranceInvalid") : t("subscriptions.leaveBlankToInherit")}</FieldDescription></Field>
    <Field><Button type="button" variant="outline" onClick={onReset}>{t("subscriptions.resetURLTest")}</Button></Field>
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
  return <Dialog open onOpenChange={(open) => { if (!open) onClose() }}><DialogContent className="max-h-[calc(100svh-2rem)] overflow-y-auto sm:max-w-lg"><DialogHeader><DialogTitle>{item ? t("subscriptions.edit") : t("subscriptions.add")}</DialogTitle><DialogDescription>{t("subscriptions.dialogDescription")}</DialogDescription></DialogHeader><FieldGroup>
    <Field><FieldLabel htmlFor="subscription-name">{t("subscriptions.name")}</FieldLabel><Input id="subscription-name" value={name} onChange={(event) => setName(event.target.value)} /></Field>
    <Field><FieldLabel htmlFor="subscription-url">{t("subscriptions.subscriptionURL")}</FieldLabel><Input id="subscription-url" value={url} onChange={(event) => setURL(event.target.value)} /></Field>
    <Field><FieldLabel htmlFor="subscription-interval">{t("subscriptions.interval")}</FieldLabel><Input id="subscription-interval" type="number" min={1} value={interval} onChange={(event) => setInterval(Number(event.target.value))} /></Field>
  </FieldGroup><URLTestFields defaults={defaults} state={urlTest} setState={setURLTest} onReset={resetURLTest} /><DialogFooter><Button variant="outline" onClick={onClose}>{t("common.cancel")}</Button><Button disabled={isInvalid || request.isPending} onClick={save}>{t("common.save")}</Button></DialogFooter></DialogContent></Dialog>
}
