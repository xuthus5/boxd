import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import { CardQueryError } from "@/features/common/card-query-error"
import { reportSettingsRequestError } from "@/features/settings/settings-request-error-actions"
import { ApiError } from "@/lib/api/client"
import { api } from "@/lib/api/endpoints"

const modeLabels: Record<string, string> = {
  Rule: "policy.routeModeRule",
  Direct: "policy.routeModeDirect",
  Global: "policy.routeModeGlobal",
}

const modeHints: Record<string, string> = {
  Rule: "policy.routeModeRuleHint",
  Direct: "policy.routeModeDirectHint",
  Global: "policy.routeModeGlobalHint",
}

/**
 * 路由模式：在「按配置规则匹配」「全局直连」「全局代理」之间一键切换。
 * 复用内核的 clash 模式（配置文件里的 clash_mode 规则），切换立即作用于运行中的内核。
 */
export function RouteModeCard({ enabled }: { enabled: boolean }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const modeLabel = (mode: string) => {
    const key = modeLabels[mode]
    return key ? t(key) : mode
  }
  const query = useQuery({
    queryKey: ["clash-mode"],
    queryFn: api.runtime.clashMode,
    enabled,
    retry: false,
    refetchInterval: enabled ? 10000 : false,
  })
  const mutation = useMutation({
    mutationFn: (mode: string) => api.runtime.setClashMode(mode),
    onSuccess: (status) => {
      queryClient.setQueryData(["clash-mode"], status)
      toast.success(t("policy.routeModeUpdated", { mode: modeLabel(status.mode) }))
    },
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "route-mode",
      fallback: t("policy.routeModeFailed"),
    }),
  })
  // 配置里缺少 clash_mode 规则时，安装默认路由会按既有优先级补齐规则，从而启用全局模式。
  const installDefaults = useMutation({
    mutationFn: () => api.config.installRoute(),
    onSuccess: async (response) => {
      if (response.status === "rolled_back") {
        toast.error(t("policy.routeModeEnableFailed"))
        return
      }
      await queryClient.invalidateQueries({ queryKey: ["config"] })
      await query.refetch()
      toast.success(t("policy.routeModeEnabled"))
    },
    onError: (error: Error) => reportSettingsRequestError(error, t, {
      scope: "route-mode-enable",
      fallback: t("policy.routeModeEnableFailed"),
    }),
  })

  if (!enabled) {
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("policy.routeModeTitle")}</CardTitle>
          <CardDescription>{t("policy.routeModeNeedRunning")}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (query.isLoading) return <Skeleton className="h-28 w-full" />

  if (query.error) {
    const unavailable = query.error instanceof ApiError && query.error.code === "invalid_request"
    return (
      <Card size="sm">
        <CardHeader className="gap-1.5">
          <CardTitle className="truncate">{t("policy.routeModeTitle")}</CardTitle>
          {unavailable ? <CardDescription>{t("policy.routeModeUnavailable")}</CardDescription> : null}
        </CardHeader>
        {unavailable ? null : (
          <CardContent>
            <CardQueryError error={query.error} scope="route-mode" onRetry={() => { void query.refetch() }} />
          </CardContent>
        )}
      </Card>
    )
  }

  const status = query.data
  if (!status) return null
  const modes = status.mode_list?.length ? status.mode_list : [status.mode]
  // clash_mode 规则缺失时内核只会报告规则模式，此时提示安装默认路由补齐。
  const missingModes = !modes.includes("Direct") && !modes.includes("Global")
  const hint = modeHints[status.mode]

  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="truncate">{t("policy.routeModeTitle")}</CardTitle>
        <CardDescription>{t("policy.routeModeDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-2 sm:gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-muted-foreground">{t("policy.routeModeCurrent")}</span>
          <Badge>{modeLabel(status.mode)}</Badge>
          {hint ? <span className="text-xs text-muted-foreground">{t(hint)}</span> : null}
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
            <ToggleGroupItem key={mode} value={mode} aria-label={modeLabel(mode)}>
              {modeLabel(mode)}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
        {missingModes ? (
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-xs text-muted-foreground">{t("policy.routeModeNeedsDefaultRoute")}</p>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="h-7"
              disabled={installDefaults.isPending}
              onClick={() => installDefaults.mutate()}
            >
              {installDefaults.isPending ? t("policy.routeModeEnabling") : t("policy.routeModeEnable")}
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
