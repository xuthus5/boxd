import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { ApiError } from "@/lib/api/client"
import { api } from "@/lib/api/endpoints"

export function ClashModeCard({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ["clash-mode"],
    queryFn: api.runtime.clashMode,
    enabled,
    retry: false,
    refetchInterval: enabled ? 10000 : false,
  })
  const mutation = useMutation({
    mutationFn: (mode: string) => api.runtime.setClashMode(mode),
    onSuccess: async (status) => {
      queryClient.setQueryData(["clash-mode"], status)
      toast.success(t("dashboard.clashModeUpdated", { mode: status.mode }))
    },
    onError: (error: Error) => toast.error(error.message),
  })

  if (!enabled) {
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("dashboard.clashMode")}</CardTitle>
          <CardDescription>{t("dashboard.clashModeNeedRunning")}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (query.isLoading) return <Skeleton className="h-32 w-full" />

  if (query.error) {
    const message = query.error instanceof ApiError && query.error.code === "invalid_request"
      ? t("dashboard.clashModeDisabled")
      : query.error.message
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("dashboard.clashMode")}</CardTitle>
          <CardDescription>{message}</CardDescription>
        </CardHeader>
        <CardContent>
          <Badge variant="secondary">{t("dashboard.clashModeUnavailable")}</Badge>
        </CardContent>
      </Card>
    )
  }

  const status = query.data
  if (!status) return null
  const modes = status.mode_list?.length ? status.mode_list : [status.mode]

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("dashboard.clashMode")}</CardTitle>
        <CardDescription>{t("dashboard.clashModeDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">{t("dashboard.clashModeCurrent")}</span>
          <Badge>{status.mode}</Badge>
        </div>
        <ToggleGroup
          className="w-full max-w-full flex-wrap justify-start"
          value={[status.mode]}
          disabled={mutation.isPending}
          onValueChange={(value) => {
            const next = value[0]
            if (next && next !== status.mode) mutation.mutate(next)
          }}
        >
          {modes.map((mode) => (
            <ToggleGroupItem key={mode} value={mode} aria-label={mode}>
              {mode}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      </CardContent>
    </Card>
  )
}
