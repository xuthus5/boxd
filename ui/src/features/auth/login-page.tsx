import { zodResolver } from "@hookform/resolvers/zod"
import { ActivityIcon, BoxIcon, ShieldCheckIcon, ZapIcon } from "lucide-react"
import { useState } from "react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { Navigate, useNavigate } from "react-router-dom"
import { z } from "zod"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/features/auth/auth-context"
import { LoginErrorAlert } from "@/features/auth/login-error-alert"
import { loginErrorFromError, type LoginErrorState } from "@/features/auth/login-error"

interface LoginValues { username: string; password: string }

const createSchema = (message: (key: string) => string) => z.object({
  username: z.string().trim().min(1, message("auth.usernameRequired")),
  password: z.string().min(1, message("auth.passwordRequired")),
})

export function LoginPage() {
  const { t } = useTranslation()
  const auth = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState<LoginErrorState | null>(null)
  const form = useForm<LoginValues>({ resolver: zodResolver(createSchema((key) => t(key))) })

  if (auth.session) return <Navigate to="/dashboard" replace />

  const submit = form.handleSubmit(async (values) => {
    setError(null)
    try {
      await auth.login(values)
      navigate("/dashboard", { replace: true })
    } catch (reason) {
      setError(loginErrorFromError(reason, t("auth.failed")))
    }
  })

  return (
    <main className="relative flex min-h-svh items-center justify-center overflow-hidden p-4 sm:p-6 lg:p-8">
      <div className="login-aurora pointer-events-none absolute inset-0 opacity-70 dark:opacity-40" aria-hidden="true" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_bottom,transparent,var(--background))] opacity-40" aria-hidden="true" />

      <div className="relative grid w-full max-w-5xl overflow-hidden rounded-3xl border border-border/70 bg-card/80 shadow-[0_32px_100px_-40px_oklch(0.18_0.04_140/0.34)] backdrop-blur-xl lg:grid-cols-[1.05fr_0.95fr]">
        <section className="relative hidden flex-col justify-between overflow-hidden bg-[oklch(0.16_0.04_140)] p-10 text-white lg:flex">
          <div className="login-aurora pointer-events-none absolute inset-0 opacity-80" aria-hidden="true" />
          <div className="relative flex items-center gap-3">
            <span className="flex size-11 items-center justify-center rounded-2xl bg-gradient-to-br from-[oklch(0.84_0.19_132)] to-[oklch(0.64_0.16_166)] text-[oklch(0.15_0.04_140)] shadow-[0_14px_34px_-14px_oklch(0.84_0.19_132)]">
              <BoxIcon className="size-5" />
            </span>
            <div>
              <p className="text-lg font-semibold tracking-tight">boxd</p>
              <p className="text-[0.65rem] tracking-[0.18em] text-white/45 uppercase">sing-box control plane</p>
            </div>
          </div>

          <div className="relative flex flex-col gap-7 py-12">
            <div className="flex flex-col gap-3">
              <span className="flex w-fit items-center gap-2 rounded-full border border-white/10 bg-white/5 px-3 py-1 text-[0.68rem] font-medium tracking-[0.12em] text-white/65 uppercase">
                <span className="size-1.5 rounded-full bg-emerald-400 shadow-[0_0_12px_oklch(0.72_0.15_160)]" />
                {t("auth.badge")}
              </span>
              <h2 className="max-w-sm text-balance text-3xl font-semibold leading-tight tracking-[-0.03em]">{t("auth.tagline")}</h2>
              <p className="max-w-sm text-sm leading-relaxed text-white/55">{t("auth.panelDescription")}</p>
            </div>

            <div className="grid gap-3 text-sm text-white/70">
              {[
                { icon: ActivityIcon, label: t("auth.featureObservability") },
                { icon: ZapIcon, label: t("auth.featureRealtime") },
                { icon: ShieldCheckIcon, label: t("auth.featureSecure") },
              ].map(({ icon: Icon, label }) => (
                <div key={label} className="flex items-center gap-3 rounded-xl border border-white/8 bg-white/[0.035] px-3.5 py-3 backdrop-blur-sm">
                  <span className="flex size-7 items-center justify-center rounded-lg bg-white/8 text-white/80">
                    <Icon className="size-3.5" />
                  </span>
                  <span>{label}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="relative flex items-center justify-between border-t border-white/8 pt-5 text-[0.68rem] text-white/35">
            <span>{t("auth.panelFooter")}</span>
            <span className="flex items-center gap-1.5"><span className="size-1.5 rounded-full bg-emerald-400" /> local</span>
          </div>
        </section>

        <section className="flex items-center justify-center p-5 sm:p-9 lg:p-12">
          <form className="w-full max-w-sm" onSubmit={submit}>
            <Card size="sm" className="gap-0 border-0 bg-transparent py-0 shadow-none ring-0 backdrop-blur-none hover:shadow-none">
              <CardHeader className="gap-2 px-0 pb-6">
                <div className="mb-2 flex size-11 items-center justify-center rounded-2xl bg-primary/10 text-primary ring-1 ring-primary/15 lg:hidden">
                  <BoxIcon className="size-5" />
                </div>
                <CardTitle role="heading" aria-level={1} className="pl-1 text-2xl tracking-[-0.03em]">{t("auth.title")}</CardTitle>
                <CardDescription className="max-w-xs">{t("auth.description")}</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 px-0">
                <LoginErrorAlert error={error} />
                <FieldGroup className="gap-4">
                  <Field data-invalid={Boolean(form.formState.errors.username)}>
                    <FieldLabel htmlFor="username">{t("auth.username")}</FieldLabel>
                    <Input id="username" className="h-10 rounded-xl bg-background/70 px-3" autoComplete="username" aria-invalid={Boolean(form.formState.errors.username)} {...form.register("username")} />
                    {form.formState.errors.username ? <FieldDescription>{form.formState.errors.username.message}</FieldDescription> : null}
                  </Field>
                  <Field data-invalid={Boolean(form.formState.errors.password)}>
                    <FieldLabel htmlFor="password">{t("auth.password")}</FieldLabel>
                    <Input id="password" type="password" className="h-10 rounded-xl bg-background/70 px-3" autoComplete="current-password" aria-invalid={Boolean(form.formState.errors.password)} {...form.register("password")} />
                    {form.formState.errors.password ? <FieldDescription>{form.formState.errors.password.message}</FieldDescription> : null}
                  </Field>
                </FieldGroup>
              </CardContent>
              <CardFooter className="mt-6 border-0 bg-transparent p-0">
                <Button size="lg" className="h-10 w-full rounded-xl shadow-[0_12px_28px_-14px_var(--primary)]" type="submit" disabled={form.formState.isSubmitting}>
                  {form.formState.isSubmitting ? <Spinner aria-hidden="true" data-icon="inline-start" /> : null}
                  {t("auth.submit")}
                </Button>
              </CardFooter>
            </Card>
          </form>
        </section>
      </div>
    </main>
  )
}
