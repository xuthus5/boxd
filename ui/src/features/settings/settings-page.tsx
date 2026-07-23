import { useMutation, useQueries } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldDescription, FieldGroup, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { useAuth } from "@/features/auth/auth-context"
import { usePreferences } from "@/features/preferences/preferences-provider"
import { RuleSetAutoUpdateCard } from "@/features/settings/ruleset-auto-update-card"
import {
  isJWTSecretReady,
  isPasswordFormReady,
  MIN_ADMIN_PASSWORD_LENGTH,
  MIN_JWT_SECRET_LENGTH,
  validateAdminPassword,
  validateJWTSecret,
  validatePasswordConfirmation,
} from "@/features/settings/security-validation"
import { URLTestDefaultsCard } from "@/features/settings/urltest-defaults-card"
import { api } from "@/lib/api/endpoints"
import { isTestURLReady } from "@/features/settings/settings-dirty"
import { resolveInitialSpeedTestURL } from "@/lib/speed-test-urls"
import { isHTTPURL } from "@/lib/urltest"
import type { Language, LogThreshold, Theme } from "@/lib/storage"

function AppearanceCard() {
  const preferences = usePreferences()
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.appearanceTitle")}</CardTitle>
        <CardDescription>{t("settings.appearanceDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup className="gap-4">
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

function passwordIssueMessage(issue: ReturnType<typeof validateAdminPassword> | ReturnType<typeof validatePasswordConfirmation>) {
  if (issue === "too_short") return "settings.passwordTooShort"
  if (issue === "matches_username") return "settings.passwordMatchesUsername"
  if (issue === "weak_common") return "settings.passwordWeakCommon"
  if (issue === "mismatch") return "settings.passwordMismatch"
  return ""
}

function jwtIssueMessage(issue: ReturnType<typeof validateJWTSecret>) {
  if (issue === "empty") return "settings.jwtSecretRequired"
  if (issue === "too_short") return "settings.jwtSecretTooShort"
  return ""
}

function AccountCard({ defaultPassword, jwt }: { defaultPassword: boolean; jwt: { masked: string; present: boolean; length: number } }) {
  const auth = useAuth()
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [secret, setSecret] = useState("")
  const passwordIssue = newPassword ? validateAdminPassword(newPassword) : null
  const confirmIssue = confirmPassword ? validatePasswordConfirmation(newPassword, confirmPassword) : null
  const jwtIssue = secret ? validateJWTSecret(secret) : null
  const passwordReady = isPasswordFormReady(currentPassword, newPassword, confirmPassword)
  const jwtReady = isJWTSecretReady(secret)
  const rotate = useMutation({
    mutationFn: () => api.settings.changePassword(currentPassword, newPassword),
    onSuccess: () => { toast.success(t("settings.passwordRotated")); auth.clear() },
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => { setCurrentPassword(""); setNewPassword(""); setConfirmPassword("") },
  })
  const rotateJWT = useMutation({
    mutationFn: () => api.settings.setJWT(secret),
    onSuccess: () => { toast.success(t("settings.jwtRotated")); auth.clear() },
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => setSecret(""),
  })
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.accountTitle")}</CardTitle>
        <CardDescription>{t("settings.accountDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        {defaultPassword ? (
          <Alert variant="destructive">
            <AlertTitle>{t("settings.defaultPasswordTitle")}</AlertTitle>
            <AlertDescription>{t("settings.defaultPasswordDescription")}</AlertDescription>
          </Alert>
        ) : null}
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="current-password">{t("settings.currentPassword")}</FieldLabel>
            <Input id="current-password" type="password" autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
          </Field>
          <Field data-invalid={passwordIssue ? true : undefined}>
            <FieldLabel htmlFor="new-password">{t("settings.newPassword")}</FieldLabel>
            <Input id="new-password" type="password" autoComplete="new-password" aria-invalid={passwordIssue ? true : undefined} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} />
            <FieldDescription>{passwordIssue ? t(passwordIssueMessage(passwordIssue), { count: MIN_ADMIN_PASSWORD_LENGTH }) : t("settings.passwordHint", { count: MIN_ADMIN_PASSWORD_LENGTH })}</FieldDescription>
          </Field>
          <Field data-invalid={confirmIssue ? true : undefined}>
            <FieldLabel htmlFor="confirm-password">{t("settings.confirmPassword")}</FieldLabel>
            <Input id="confirm-password" type="password" autoComplete="new-password" aria-invalid={confirmIssue ? true : undefined} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
            <FieldDescription>{confirmIssue ? t(passwordIssueMessage(confirmIssue)) : t("settings.confirmPasswordHint")}</FieldDescription>
          </Field>
          <Field>
            <Button disabled={!passwordReady || rotate.isPending} onClick={() => rotate.mutate()}>{t("settings.rotatePassword")}</Button>
          </Field>
        </FieldGroup>
        <FieldGroup>
          <Field data-invalid={jwtIssue ? true : undefined}>
            <FieldLabel htmlFor="jwt-secret">{t("settings.jwtSecret")}</FieldLabel>
            <Input id="jwt-secret" type="password" autoComplete="off" aria-invalid={jwtIssue ? true : undefined} placeholder={`${jwt.masked} (${jwt.length})`} value={secret} onChange={(event) => setSecret(event.target.value)} />
            <FieldDescription>{jwtIssue ? t(jwtIssueMessage(jwtIssue), { count: MIN_JWT_SECRET_LENGTH }) : t("settings.jwtSecretHint", { count: MIN_JWT_SECRET_LENGTH })}</FieldDescription>
          </Field>
          <Field>
            <AlertDialog>
              <AlertDialogTrigger render={<Button variant="destructive" disabled={!jwtReady || rotateJWT.isPending} />}>{t("settings.rotateJWT")}</AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>{t("settings.rotateJWTTitle")}</AlertDialogTitle>
                  <AlertDialogDescription>{t("settings.rotateJWTDescription")}</AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>{t("settings.cancel")}</AlertDialogCancel>
                  <AlertDialogAction variant="destructive" onClick={() => rotateJWT.mutate()}>{t("settings.confirmRotate")}</AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
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
    onError: (error: Error) => toast.error(error.message),
  })
  const saveAutostart = (checked: boolean) => {
    const previous = autostart
    setAutostart(checked)
    api.settings.setAutostart(checked).then(() => toast.success(t("settings.autostartSaved"))).catch((error: Error) => {
      setAutostart(previous)
      toast.error(error.message)
    })
  }
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.runtimeTitle")}</CardTitle>
        <CardDescription>{t("settings.runtimeDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup>
          <div className="grid gap-2">
            <ProbeURLField
              id="test-url"
              label={t("settings.testURL")}
              value={testURL}
              onChange={setTestURL}
              invalid={urlInvalid}
              description={urlInvalid ? t("settings.urlTestURLInvalid") : t("settings.testURLDescription")}
            />
            <Button onClick={() => saveURL.mutate()} disabled={!urlReady || saveURL.isPending}>
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
  if (error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{error.message}</AlertDescription></Alert>
  return <div className="flex flex-col gap-3 sm:gap-4"><h1 className="text-2xl font-semibold">{t("settings.title")}</h1><div className="grid gap-3 sm:gap-4 lg:grid-cols-2"><AppearanceCard /><AccountCard defaultPassword={password.data!.defaultPassword} jwt={jwt.data!} /><RuntimeSettingsCard url={testURL.data!.url} enabled={autostart.data!.enabled} /><URLTestDefaultsCard defaults={urlTestDefaults.data!} /><RuleSetAutoUpdateCard defaults={ruleSetAuto.data!} /></div></div>
}
