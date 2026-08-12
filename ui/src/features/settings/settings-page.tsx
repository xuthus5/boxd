import { useMutation, useQueries } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
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
import { isDesktop } from "@/lib/api/desktop"
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
            <FieldContent className="w-full sm:w-auto sm:min-w-40">
              <Select
                items={[
                  { value: "light", label: t("settings.light") },
                  { value: "dark", label: t("settings.dark") },
                  { value: "system", label: t("settings.system") },
                ]}
                value={preferences.theme}
                onValueChange={(value) => preferences.setTheme(String(value) as Theme)}
              >
                <SelectTrigger aria-label={t("settings.theme")} className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="light">{t("settings.light")}</SelectItem>
                    <SelectItem value="dark">{t("settings.dark")}</SelectItem>
                    <SelectItem value="system">{t("settings.system")}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FieldContent>
          </Field>
          <Field orientation="responsive" className="gap-2 sm:justify-between">
            <FieldTitle id="language-label" className="shrink-0">{t("settings.language")}</FieldTitle>
            <FieldContent className="w-full sm:w-auto sm:min-w-40">
              <Select
                items={[
                  { value: "zh", label: "中文" },
                  { value: "en", label: "English" },
                ]}
                value={preferences.language}
                onValueChange={(value) => preferences.setLanguage(String(value) as Language)}
              >
                <SelectTrigger aria-label={t("settings.language")} className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="zh">中文</SelectItem>
                    <SelectItem value="en">English</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FieldContent>
          </Field>
          <Field orientation="responsive" className="gap-2 sm:justify-between">
            <FieldContent className="min-w-0 gap-1">
              <FieldTitle id="minimum-log-level-label" className="shrink-0">{t("settings.minimumLogLevel")}</FieldTitle>
              <FieldDescription>{t("settings.minimumLogLevelDescription")}</FieldDescription>
            </FieldContent>
            <FieldContent className="w-full sm:w-auto sm:min-w-40">
              <Select
                items={[
                  { value: "all", label: t("observability.allLevels") },
                  { value: "debug", label: "Debug" },
                  { value: "info", label: "Info" },
                  { value: "warn", label: "Warn" },
                  { value: "error", label: "Error" },
                ]}
                value={preferences.minimumLogLevel}
                onValueChange={(value) => preferences.setMinimumLogLevel(String(value) as LogThreshold)}
              >
                <SelectTrigger aria-label={t("settings.minimumLogLevel")} className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="all">{t("observability.allLevels")}</SelectItem>
                    <SelectItem value="debug">Debug</SelectItem>
                    <SelectItem value="info">Info</SelectItem>
                    <SelectItem value="warn">Warn</SelectItem>
                    <SelectItem value="error">Error</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </FieldContent>
          </Field>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

export function RuntimeSettingsCard({ url, enabled, appAutostart }: { url: string; enabled: boolean; appAutostart: boolean }) {
  const { t } = useTranslation()
  const [savedURL, setSavedURL] = useState(url)
  const [testURL, setTestURL] = useState(() => resolveInitialSpeedTestURL(url))
  const [autostart, setAutostart] = useState(enabled)
  const [desktopAutostart, setDesktopAutostart] = useState(appAutostart)
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
  const saveDesktopAutostart = (checked: boolean) => {
    const previous = desktopAutostart
    setDesktopAutostart(checked)
    api.desktop.setAutostart(checked).then(() => toast.success(t("settings.appAutostartSaved"))).catch((error: Error) => {
      setDesktopAutostart(previous)
      reportSettingsRequestError(error, t, {
        scope: "app-autostart",
        fallback: t("settings.appAutostartFailed"),
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
          {isDesktop() ? (
            <Field orientation="horizontal">
              <div className="min-w-0">
                <FieldLabel htmlFor="app-autostart">{t("settings.appAutostart")}</FieldLabel>
                <FieldDescription>{t("settings.appAutostartDescription")}</FieldDescription>
              </div>
              <Switch id="app-autostart" checked={desktopAutostart} onCheckedChange={saveDesktopAutostart} />
            </Field>
          ) : null}
        </FieldGroup>
      </CardContent>
    </Card>
  )
}

export function SettingsPage() {
  const { t } = useTranslation()
  const preferences = usePreferences()
  const [password, jwt, testURL, autostart, urlTestDefaults, ruleSetAuto, appAutostart] = useQueries({ queries: [
    { queryKey: ["settings", "password"], queryFn: api.settings.password },
    { queryKey: ["settings", "jwt"], queryFn: api.settings.jwt },
    { queryKey: ["settings", "url"], queryFn: api.settings.testURL },
    { queryKey: ["settings", "autostart"], queryFn: api.settings.autostart },
    { queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults },
    { queryKey: ["settings", "ruleset-auto-update"], queryFn: api.config.ruleSetsAutoUpdate },
    { queryKey: ["desktop", "autostart"], queryFn: api.desktop.autostart, enabled: isDesktop() },
  ] })
  const queries = [password, jwt, testURL, autostart, urlTestDefaults, ruleSetAuto, appAutostart]
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
  return <div className="flex flex-col gap-3 sm:gap-4"><h1 className="text-2xl font-semibold">{t("settings.title")}</h1><div className="grid gap-3 sm:gap-4 lg:grid-cols-2"><AppearanceCard /><AccountSecurityCard defaultPassword={password.data!.defaultPassword} jwt={jwt.data!} /><RuntimeSettingsCard url={testURL.data!.url} enabled={autostart.data!.enabled} appAutostart={appAutostart.data?.enabled ?? false} /><URLTestDefaultsCard defaults={urlTestDefaults.data!} /><RuleSetAutoUpdateCard defaults={ruleSetAuto.data!} /><SupportBundleCard preferences={{ theme: preferences.theme, language: preferences.language, minimumLogLevel: preferences.minimumLogLevel }} /><BackupExportCard /></div></div>
}
