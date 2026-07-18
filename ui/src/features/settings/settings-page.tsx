import { useMutation, useQueries } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ProbeURLField } from "@/components/probe-url-field"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle, AlertDialogTrigger } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { useAuth } from "@/features/auth/auth-context"
import { usePreferences } from "@/features/preferences/preferences-provider"
import { URLTestDefaultsCard } from "@/features/settings/urltest-defaults-card"
import { api } from "@/lib/api/endpoints"
import { resolveInitialSpeedTestURL } from "@/lib/speed-test-urls"
import type { Language, LogThreshold, Theme } from "@/lib/storage"

function AppearanceCard() {
  const preferences = usePreferences()
  const { t } = useTranslation()
  return <Card><CardHeader><CardTitle>{t("settings.appearanceTitle")}</CardTitle><CardDescription>{t("settings.appearanceDescription")}</CardDescription></CardHeader><CardContent><FieldGroup>
    <Field orientation="horizontal"><FieldTitle id="theme-label">{t("settings.theme")}</FieldTitle><ToggleGroup aria-labelledby="theme-label" value={[preferences.theme]} onValueChange={(value) => { if (value[0]) preferences.setTheme(value[0] as Theme) }}><ToggleGroupItem value="light">{t("settings.light")}</ToggleGroupItem><ToggleGroupItem value="dark">{t("settings.dark")}</ToggleGroupItem><ToggleGroupItem value="system">{t("settings.system")}</ToggleGroupItem></ToggleGroup></Field>
    <Field orientation="horizontal"><FieldTitle id="language-label">{t("settings.language")}</FieldTitle><ToggleGroup aria-labelledby="language-label" value={[preferences.language]} onValueChange={(value) => { if (value[0]) preferences.setLanguage(value[0] as Language) }}><ToggleGroupItem value="zh">中文</ToggleGroupItem><ToggleGroupItem value="en">English</ToggleGroupItem></ToggleGroup></Field>
    <Field orientation="horizontal"><FieldTitle id="minimum-log-level-label">{t("settings.minimumLogLevel")}</FieldTitle><ToggleGroup aria-labelledby="minimum-log-level-label" value={[preferences.minimumLogLevel]} onValueChange={(value) => { if (value[0]) preferences.setMinimumLogLevel(value[0] as LogThreshold) }}><ToggleGroupItem value="all">{t("observability.allLevels")}</ToggleGroupItem><ToggleGroupItem value="debug">Debug</ToggleGroupItem><ToggleGroupItem value="info">Info</ToggleGroupItem><ToggleGroupItem value="warn">Warn</ToggleGroupItem><ToggleGroupItem value="error">Error</ToggleGroupItem></ToggleGroup></Field>
    <FieldDescription>{t("settings.minimumLogLevelDescription")}</FieldDescription>
  </FieldGroup></CardContent></Card>
}

function AccountCard({ defaultPassword, jwt }: { defaultPassword: boolean; jwt: { masked: string; present: boolean; length: number } }) {
  const auth = useAuth()
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [secret, setSecret] = useState("")
  const rotate = useMutation({ mutationFn: () => api.settings.changePassword(currentPassword, newPassword), onSuccess: () => { toast.success(t("settings.passwordRotated")); auth.clear() }, onError: (error: Error) => toast.error(error.message), onSettled: () => { setCurrentPassword(""); setNewPassword("") } })
  const rotateJWT = useMutation({ mutationFn: () => api.settings.setJWT(secret), onSuccess: () => { toast.success(t("settings.jwtRotated")); auth.clear() }, onError: (error: Error) => toast.error(error.message), onSettled: () => setSecret("") })
  return <Card><CardHeader><CardTitle>{t("settings.accountTitle")}</CardTitle><CardDescription>{t("settings.accountDescription")}</CardDescription></CardHeader><CardContent className="flex flex-col gap-4">
    {defaultPassword ? <Alert variant="destructive"><AlertTitle>{t("settings.defaultPasswordTitle")}</AlertTitle><AlertDescription>{t("settings.defaultPasswordDescription")}</AlertDescription></Alert> : null}
    <FieldGroup><Field><FieldLabel htmlFor="current-password">{t("settings.currentPassword")}</FieldLabel><Input id="current-password" type="password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} /></Field><Field><FieldLabel htmlFor="new-password">{t("settings.newPassword")}</FieldLabel><Input id="new-password" type="password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} /><FieldDescription>{t("settings.passwordHint")}</FieldDescription></Field><Field><Button disabled={!currentPassword || newPassword.length < 8} onClick={() => rotate.mutate()}>{t("settings.rotatePassword")}</Button></Field></FieldGroup>
    <FieldGroup><Field><FieldLabel htmlFor="jwt-secret">{t("settings.jwtSecret")}</FieldLabel><Input id="jwt-secret" type="password" placeholder={`${jwt.masked} (${jwt.length})`} value={secret} onChange={(event) => setSecret(event.target.value)} /></Field><Field><AlertDialog><AlertDialogTrigger render={<Button variant="destructive" disabled={!secret} />}>{t("settings.rotateJWT")}</AlertDialogTrigger><AlertDialogContent><AlertDialogHeader><AlertDialogTitle>{t("settings.rotateJWTTitle")}</AlertDialogTitle><AlertDialogDescription>{t("settings.rotateJWTDescription")}</AlertDialogDescription></AlertDialogHeader><AlertDialogFooter><AlertDialogCancel>{t("settings.cancel")}</AlertDialogCancel><AlertDialogAction variant="destructive" onClick={() => rotateJWT.mutate()}>{t("settings.confirmRotate")}</AlertDialogAction></AlertDialogFooter></AlertDialogContent></AlertDialog></Field></FieldGroup>
  </CardContent></Card>
}

function RuntimeSettingsCard({ url, enabled }: { url: string; enabled: boolean }) {
  const { t } = useTranslation()
  const [testURL, setTestURL] = useState(() => resolveInitialSpeedTestURL(url))
  const [autostart, setAutostart] = useState(enabled)
  const saveURL = useMutation({ mutationFn: () => api.settings.setTestURL(testURL), onSuccess: () => toast.success(t("settings.testURLSaved")), onError: (error: Error) => toast.error(error.message) })
  const saveAutostart = (checked: boolean) => {
    const previous = autostart
    setAutostart(checked)
    api.settings.setAutostart(checked).then(() => toast.success(t("settings.autostartSaved"))).catch((error: Error) => { setAutostart(previous); toast.error(error.message) })
  }
  return <Card><CardHeader><CardTitle>{t("settings.runtimeTitle")}</CardTitle><CardDescription>{t("settings.runtimeDescription")}</CardDescription></CardHeader><CardContent><FieldGroup>
    <div className="grid gap-2">
      <ProbeURLField
        id="test-url"
        label={t("settings.testURL")}
        value={testURL}
        onChange={setTestURL}
        description={t("settings.testURLDescription")}
      />
      <Button onClick={() => saveURL.mutate()} disabled={!testURL.trim()}>{t("settings.saveTestURL")}</Button>
    </div>
    <Field orientation="horizontal"><FieldLabel htmlFor="autostart">{t("settings.autostart")}</FieldLabel><Switch id="autostart" checked={autostart} onCheckedChange={saveAutostart} /></Field>
  </FieldGroup></CardContent></Card>
}

export function SettingsPage() {
  const { t } = useTranslation()
  const [password, jwt, testURL, autostart, urlTestDefaults] = useQueries({ queries: [
    { queryKey: ["settings", "password"], queryFn: api.settings.password },
    { queryKey: ["settings", "jwt"], queryFn: api.settings.jwt },
    { queryKey: ["settings", "url"], queryFn: api.settings.testURL },
    { queryKey: ["settings", "autostart"], queryFn: api.settings.autostart },
    { queryKey: ["settings", "urltest-defaults"], queryFn: api.settings.urlTestDefaults },
  ] })
  const queries = [password, jwt, testURL, autostart, urlTestDefaults]
  if (queries.some((query) => query.isLoading)) return <Skeleton className="h-64 w-full" />
  const error = queries.find((query) => query.error)?.error
  if (error) return <Alert variant="destructive"><AlertTitle>{t("common.loadFailed")}</AlertTitle><AlertDescription>{error.message}</AlertDescription></Alert>
  return <div className="flex flex-col gap-4"><h1 className="text-2xl font-semibold">{t("settings.title")}</h1><div className="grid gap-4 lg:grid-cols-2"><AppearanceCard /><AccountCard defaultPassword={password.data!.defaultPassword} jwt={jwt.data!} /><RuntimeSettingsCard url={testURL.data!.url} enabled={autostart.data!.enabled} /><URLTestDefaultsCard defaults={urlTestDefaults.data!} /></div></div>
}
