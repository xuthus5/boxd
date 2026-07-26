import { useMutation, useQueries } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { usePreferences } from "@/features/preferences/preferences-provider"
import { BackupExportCard } from "@/features/settings/backup-export-card"
import { SupportBundleCard } from "@/features/settings/support-bundle-card"
import { RuleSetAutoUpdateCard } from "@/features/settings/ruleset-auto-update-card"
import { AccountSecurityCard } from "@/features/settings/account-security-card"
import { URLTestDefaultsCard } from "@/features/settings/urltest-defaults-card"
import { api } from "@/lib/api/endpoints"
import { isTestURLReady } from "@/features/settings/settings-dirty"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import { resolveInitialSpeedTestURL } from "@/lib/speed-test-urls"
import { isHTTPURL } from "@/lib/urltest"
import type { Language, LogThreshold, Theme } from "@/lib/storage"
import { PageLoadErrorAlert } from "@/features/common/page-load-error-alert"

function AppearanceCard() {
  const preferences = usePreferences()
  const { t } = useTranslation()
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("settings.appearanceTitle")}</CardTitle>
        <CardDescription>{t("settings.appearanceDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-2 sm:gap-3">
          <Field orientation="responsive" className="gap-2 sm:justify-between">
            <FieldTitle id="theme-label" className="shrink-0">{t("settings.theme")}</FieldTitle>
            <ToggleGroup
              aria-labelledby="theme-label"
              className="w-full max-w-full flex-wrap justify-start sm:w-auto sm:justify-end"
              value={[preferences.theme]}
              onValueChange={(value) => { if (value[0]) preferences.setTheme(value[0] as Theme) }}
            >
              <ToggleGroupItem value="light">{t("settings.light")}</ToggleGroupItem>
              <ToggleGroupItem value="dark">{t("settings.dark")}</ToggleGroupItem>
              <ToggleGroupItem value="system">{t("settings.system")}</ToggleGroupItem>
            </ToggleGroup>
          </Field>
          <Field orientation="responsive" className="gap-2 sm:justify-between">
            <FieldTitle id="language-label" className="shrink-0">{t("settings.language")}</FieldTitle>
            <ToggleGroup
              aria-labelledby="language-label"
              className="w-full max-w-full flex-wrap justify-start sm:w-auto sm:justify-end"
              value={[preferences.language]}
              onValueChange={(value) => { if (value[0]) preferences.setLanguage(value[0] as Language) }}
            >
              <ToggleGroupItem value="zh">中文</ToggleGroupItem>
              <ToggleGroupItem value="en">English</ToggleGroupItem>
            </ToggleGroup>
          </Field>
          <Field orientation="responsive" className="gap-2 sm:justify-between">
            <FieldContent className="min-w-0 gap-1">
              <FieldTitle id="minimum-log-level-label" className="shrink-0">{t("settings.minimumLogLevel")}</FieldTitle>
              <FieldDescription>{t("settings.minimumLogLevelDescription")}</FieldDescription>
            </FieldContent>
            <ToggleGroup
              aria-labelledby="minimum-log-level-label"
              className="w-full max-w-full flex-wrap justify-start sm:w-auto sm:max-w-[22rem] sm:justify-end"
              value={[preferences.minimumLogLevel]}
              onValueChange={(value) => { if (value[0]) preferences.setMinimumLogLevel(value[0] as LogThreshold) }}
            >
              <ToggleGroupItem value="all">{t("observability.allLevels")}</ToggleGroupItem>
              <ToggleGroupItem value="debug">Debug</ToggleGroupItem>
              <ToggleGroupItem value="info">Info</ToggleGroupItem>
              <ToggleGroupItem value="warn">Warn</ToggleGroupItem>
              <ToggleGroupItem value="error">Error</ToggleGroupItem>
            </ToggleGroup>
          </Field>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

function RuntimeSettingsCard({ url, enabled }: { url: string; enabled: boolean }) {
  const { t } = useTranslation()
  const [savedURL, setSavedURL] = useState(url)
  const [testURL, setTestURL] = useState(() => resolveInitialSpeedTestURL(url))
  const [autostart, setAutostart] = useState(enabled)
  const urlReady = isTestURLReady(testURL, savedURL)
  const urlInvalid = Boolean(testURL.trim()) && !isHTTPURL(testURL.trim())
  const saveURL = useMutation({
    mutationFn: () => api.settings.setTestURL(testURL.trim()),
    onSuccess: () => {
      setSavedURL(testURL.trim())
      toast.success(t("settings.testURLSaved"))
    },
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "test-url",
      fallback: t("settings.testURLFailed"),
    }),
  })
  const saveAutostart = (checked: boolean) => {
    const previous = autostart
    setAutostart(checked)
    api.settings.setAutostart(checked).then(() => toast.success(t("settings.autostartSaved"))).catch((error: Error) => {
      setAutostart(previous)
      reportSettingsRequestError(error, t, {
        scope: "autostart",
        fallback: t("settings.autostartFailed"),
      })
    })
  }
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("settings.runtimeTitle")}</CardTitle>
        <CardDescription>{t("settings.runtimeDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-2 sm:gap-3">
          <div className="grid gap-2">
            <ProbeURLField
              id="test-url"
              label={t("settings.testURL")}
              value={testURL}
              onChange={setTestURL}
              invalid={urlInvalid}
              description={urlInvalid ? t("settings.urlTestURLInvalid") : t("settings.testURLDescription")}
            />
            <Button size="sm" className="h-8 w-full sm:w-auto" onClick={() => saveURL.mutate()} disabled={!urlReady || saveURL.isPending}>
              {t("settings.saveTestURL")}
            </Button>
          </div>
          <Field orientation="horizontal">
            <FieldLabel htmlFor="autostart">{t("settings.autostart")}</FieldLabel>
            <Switch id="autostart" checked={autostart} onCheckedChange={saveAutostart} />
          </Field>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

export function SettingsPage() {
  const { t } = useTranslation()
  const preferences = usePreferences()
  const [password, jwt, testURL, autostart, urlTestDefaults, ruleSetAuto] = useQueries({ queries: [
    { queryKey: ["settings", "password"], queryFn: api.settings.password },
    { queryKey: ["settings", "jwt"], queryFn: api.settings.jwt },
    { queryKey: ["settings", "url"], queryFn: api.settings.testURL },
    { queryKey: ["settings", "autostart"], queryFn: api.settings.autostart },
    { queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults },
    { queryKey: ["settings", "ruleset-auto-update"], queryFn: api.config.ruleSetsAutoUpdate },
  ] })
  const queries = [password, jwt, testURL, autostart, urlTestDefaults, ruleSetAuto]
  if (queries.some((query) => query.isLoading)) return <Skeleton className="h-64 w-full" />
  const error = queries.find((query) => query.error)?.error
  if (error) {
    return (
      <PageLoadErrorAlert
        error={error}
        scope="settings"
        onRetry={() => {
          for (const item of queries) {
            if (item.error) void item.refetch()
          }
        }}
      />
    )
  }
  return <div className="flex flex-col gap-3 sm:gap-4"><h1 className="text-2xl font-semibold">{t("settings.title")}</h1><div className="grid gap-3 sm:gap-4 lg:grid-cols-2"><AppearanceCard /><AccountSecurityCard defaultPassword={password.data!.defaultPassword} jwt={jwt.data!} /><RuntimeSettingsCard url={testURL.data!.url} enabled={autostart.data!.enabled} /><URLTestDefaultsCard defaults={urlTestDefaults.data!} /><RuleSetAutoUpdateCard defaults={ruleSetAuto.data!} /><SupportBundleCard preferences={{ theme: preferences.theme, language: preferences.language, minimumLogLevel: preferences.minimumLogLevel }} /><BackupExportCard /></div></div>
}
