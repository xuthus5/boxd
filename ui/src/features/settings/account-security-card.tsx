import { useMutation } from "@tanstack/react-query"
import { useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { ConfirmAction } from "@/components/confirm-action"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useAuth } from "@/features/auth/auth-context"
import {
  isJWTSecretReady,
  isPasswordFormReady,
  MIN_ADMIN_PASSWORD_LENGTH,
  MIN_JWT_SECRET_LENGTH,
  validateAdminPassword,
  validateJWTSecret,
  validatePasswordConfirmation,
} from "@/features/settings/security-validation"
import { api } from "@/lib/api/endpoints"

function passwordIssueMessage(
  issue: ReturnType<typeof validateAdminPassword> | ReturnType<typeof validatePasswordConfirmation>,
) {
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

export function AccountSecurityCard({
  defaultPassword,
  jwt,
}: {
  defaultPassword: boolean
  jwt: { masked: string; present: boolean; length: number }
}) {
  const auth = useAuth()
  const { t } = useTranslation()
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [secret, setSecret] = useState("")
  const passwordIssue = newPassword ? validateAdminPassword(newPassword) : null
  const confirmIssue = confirmPassword
    ? validatePasswordConfirmation(newPassword, confirmPassword)
    : null
  const jwtIssue = secret ? validateJWTSecret(secret) : null
  const passwordReady = isPasswordFormReady(currentPassword, newPassword, confirmPassword)
  const jwtReady = isJWTSecretReady(secret)
  const passwordDirty = Boolean(currentPassword || newPassword || confirmPassword)
  const jwtDirty = Boolean(secret)

  const rotate = useMutation({
    mutationFn: () => api.settings.changePassword(currentPassword, newPassword),
    onSuccess: () => {
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
      toast.success(t("settings.passwordRotated"))
      auth.clear()
    },
    onError: (error: Error) => toast.error(error.message),
  })
  const rotateJWT = useMutation({
    mutationFn: () => api.settings.setJWT(secret),
    onSuccess: () => {
      setSecret("")
      toast.success(t("settings.jwtRotated"))
      auth.clear()
    },
    onError: (error: Error) => toast.error(error.message),
  })

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <CardTitle className="truncate">{t("settings.accountTitle")}</CardTitle>
          {defaultPassword ? (
            <Badge variant="destructive">{t("settings.defaultPasswordBadge")}</Badge>
          ) : (
            <Badge variant="secondary">{t("settings.customPasswordBadge")}</Badge>
          )}
          <Badge variant={jwt.present ? "outline" : "destructive"} className="tabular-nums">
            {jwt.present
              ? t("settings.jwtPresentBadge", { length: jwt.length })
              : t("settings.jwtMissingBadge")}
          </Badge>
        </div>
        <CardDescription>{t("settings.accountDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        {defaultPassword ? (
          <Alert variant="destructive">
            <AlertTitle>{t("settings.defaultPasswordTitle")}</AlertTitle>
            <AlertDescription>{t("settings.defaultPasswordDescription")}</AlertDescription>
          </Alert>
        ) : null}

        <FieldGroup className="gap-3">
          <p className="text-sm font-medium">{t("settings.passwordSectionTitle")}</p>
          <p className="text-xs text-muted-foreground">{t("settings.passwordSectionHint")}</p>
          <Field>
            <FieldLabel htmlFor="current-password">{t("settings.currentPassword")}</FieldLabel>
            <Input
              id="current-password"
              type="password"
              className="h-8"
              autoComplete="current-password"
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
            />
          </Field>
          <Field data-invalid={passwordIssue ? true : undefined}>
            <FieldLabel htmlFor="new-password">{t("settings.newPassword")}</FieldLabel>
            <Input
              id="new-password"
              type="password"
              className="h-8"
              autoComplete="new-password"
              aria-invalid={passwordIssue ? true : undefined}
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
            />
            <FieldDescription>
              {passwordIssue
                ? t(passwordIssueMessage(passwordIssue), { count: MIN_ADMIN_PASSWORD_LENGTH })
                : t("settings.passwordHint", { count: MIN_ADMIN_PASSWORD_LENGTH })}
            </FieldDescription>
          </Field>
          <Field data-invalid={confirmIssue ? true : undefined}>
            <FieldLabel htmlFor="confirm-password">{t("settings.confirmPassword")}</FieldLabel>
            <Input
              id="confirm-password"
              type="password"
              className="h-8"
              autoComplete="new-password"
              aria-invalid={confirmIssue ? true : undefined}
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
            />
            <FieldDescription>
              {confirmIssue
                ? t(passwordIssueMessage(confirmIssue))
                : t("settings.confirmPasswordHint")}
            </FieldDescription>
          </Field>
          <div className="flex flex-wrap gap-2">
            <ConfirmAction
              trigger={
                <Button size="sm" className="h-8" disabled={!passwordReady || rotate.isPending}>
                  {t("settings.rotatePassword")}
                </Button>
              }
              title={t("settings.rotatePasswordTitle")}
              description={t("settings.rotatePasswordDescription")}
              confirmLabel={t("settings.confirmRotate")}
              confirmVariant="destructive"
              onConfirm={async () => {
                try {
                  await rotate.mutateAsync()
                } catch {
                  // mutation onError already toasts; keep dialog open for retry/clear
                }
              }}
            />
            {passwordDirty ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8"
                disabled={rotate.isPending}
                onClick={() => {
                  setCurrentPassword("")
                  setNewPassword("")
                  setConfirmPassword("")
                }}
              >
                {t("settings.clearForm")}
              </Button>
            ) : null}
          </div>
        </FieldGroup>

        <FieldGroup className="gap-3">
          <p className="text-sm font-medium">{t("settings.jwtSectionTitle")}</p>
          <p className="text-xs text-muted-foreground">{t("settings.jwtSectionHint")}</p>
          <Field data-invalid={jwtIssue ? true : undefined}>
            <FieldLabel htmlFor="jwt-secret">{t("settings.jwtSecret")}</FieldLabel>
            <Input
              id="jwt-secret"
              type="password"
              className="h-8"
              autoComplete="off"
              aria-invalid={jwtIssue ? true : undefined}
              placeholder={
                jwt.present
                  ? `${jwt.masked} (${jwt.length})`
                  : t("settings.jwtSecretPlaceholder")
              }
              value={secret}
              onChange={(event) => setSecret(event.target.value)}
            />
            <FieldDescription>
              {jwtIssue
                ? t(jwtIssueMessage(jwtIssue), { count: MIN_JWT_SECRET_LENGTH })
                : t("settings.jwtSecretHint", { count: MIN_JWT_SECRET_LENGTH })}
            </FieldDescription>
          </Field>
          <div className="flex flex-wrap gap-2">
            <ConfirmAction
              trigger={
                <Button
                  variant="destructive"
                  size="sm"
                  className="h-8"
                  disabled={!jwtReady || rotateJWT.isPending}
                >
                  {t("settings.rotateJWT")}
                </Button>
              }
              title={t("settings.rotateJWTTitle")}
              description={t("settings.rotateJWTDescription")}
              confirmLabel={t("settings.confirmRotate")}
              confirmVariant="destructive"
              onConfirm={async () => {
                try {
                  await rotateJWT.mutateAsync()
                } catch {
                  // mutation onError already toasts; keep dialog open for retry/clear
                }
              }}
            />
            {jwtDirty ? (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="h-8"
                disabled={rotateJWT.isPending}
                onClick={() => setSecret("")}
              >
                {t("settings.clearForm")}
              </Button>
            ) : null}
          </div>
        </FieldGroup>
      </CardContent>
    </Card>
  )
}
