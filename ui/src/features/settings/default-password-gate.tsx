import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Navigate, useLocation } from "react-router-dom"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { api } from "@/lib/api/endpoints"

export function useDefaultPasswordStatus() {
  return useQuery({
    queryKey: ["settings", "password"],
    queryFn: api.settings.password,
    staleTime: 15_000,
    retry: false,
    refetchOnWindowFocus: false,
  })
}

export function DefaultPasswordBanner() {
  const { t } = useTranslation()
  return (
    <Alert variant="destructive">
      <AlertTitle>{t("settings.defaultPasswordTitle")}</AlertTitle>
      <AlertDescription>{t("settings.defaultPasswordForced")}</AlertDescription>
    </Alert>
  )
}

export function DefaultPasswordGate({ children }: { children: React.ReactNode }) {
  const location = useLocation()
  const query = useDefaultPasswordStatus()
  const forced = query.data?.defaultPassword === true
  const onSettings = location.pathname === "/settings"

  // Fail open while loading/error so normal pages and tests keep working.
  if (forced && !onSettings) return <Navigate to="/settings" replace />
  return (
    <div className="flex min-w-0 flex-col gap-4">
      {forced ? <DefaultPasswordBanner /> : null}
      {children}
    </div>
  )
}
