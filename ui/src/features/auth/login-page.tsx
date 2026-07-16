import { zodResolver } from "@hookform/resolvers/zod"
import { useState } from "react"
import { useForm } from "react-hook-form"
import { useTranslation } from "react-i18next"
import { Navigate, useNavigate } from "react-router-dom"
import { z } from "zod"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Spinner } from "@/components/ui/spinner"
import { useAuth } from "@/features/auth/auth-context"

interface LoginValues { username: string; password: string }

const createSchema = (message: (key: string) => string) => z.object({
  username: z.string().trim().min(1, message("auth.usernameRequired")),
  password: z.string().min(1, message("auth.passwordRequired")),
})

export function LoginPage() {
  const { t } = useTranslation()
  const auth = useAuth()
  const navigate = useNavigate()
  const [error, setError] = useState("")
  const form = useForm<LoginValues>({ resolver: zodResolver(createSchema((key) => t(key))) })

  if (auth.session) return <Navigate to="/dashboard" replace />

  const submit = form.handleSubmit(async (values) => {
    setError("")
    try {
      await auth.login(values)
      navigate("/dashboard", { replace: true })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : t("auth.failed"))
    }
  })

  return (
    <main className="flex min-h-svh items-center justify-center p-6">
      <form className="w-full max-w-sm" onSubmit={submit}>
        <Card>
          <CardHeader><CardTitle role="heading" aria-level={1}>{t("auth.title")}</CardTitle><CardDescription>{t("auth.description")}</CardDescription></CardHeader>
          <CardContent className="flex flex-col gap-4">
            {error ? <Alert variant="destructive"><AlertTitle>{t("auth.failed")}</AlertTitle><AlertDescription>{error}</AlertDescription></Alert> : null}
            <FieldGroup>
              <Field data-invalid={Boolean(form.formState.errors.username)}><FieldLabel htmlFor="username">{t("auth.username")}</FieldLabel><Input id="username" autoComplete="username" aria-invalid={Boolean(form.formState.errors.username)} {...form.register("username")} />{form.formState.errors.username ? <FieldDescription>{form.formState.errors.username.message}</FieldDescription> : null}</Field>
              <Field data-invalid={Boolean(form.formState.errors.password)}><FieldLabel htmlFor="password">{t("auth.password")}</FieldLabel><Input id="password" type="password" autoComplete="current-password" aria-invalid={Boolean(form.formState.errors.password)} {...form.register("password")} />{form.formState.errors.password ? <FieldDescription>{form.formState.errors.password.message}</FieldDescription> : null}</Field>
            </FieldGroup>
          </CardContent>
          <CardFooter>
            <Button className="w-full" type="submit" disabled={form.formState.isSubmitting}>
              {form.formState.isSubmitting ? <Spinner aria-hidden="true" data-icon="inline-start" /> : null}
              {t("auth.submit")}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </main>
  )
}
