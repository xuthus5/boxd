import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Link } from "react-router-dom"

import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { api } from "@/lib/api/endpoints"
import { cn } from "@/lib/utils"

export function KernelStatusBadge() {
  const { t } = useTranslation()
  const status = useQuery({
    queryKey: ["service"],
    queryFn: api.service.status,
    refetchInterval: 5000,
    staleTime: 4000,
    retry: false,
  })

  if (status.isLoading) {
    return (
      <Badge variant="outline" className="max-w-[12rem] truncate tabular-nums" data-kernel-status="loading">
        {t("nav.kernelChecking")}
      </Badge>
    )
  }

  if (status.error || !status.data) {
    return (
      <Link
        to="/dashboard"
        className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7 px-2")}
        aria-label={t("nav.kernelUnknown")}
      >
        <Badge variant="outline" className="max-w-[12rem] truncate" data-kernel-status="unknown">
          {t("nav.kernelUnknown")}
        </Badge>
      </Link>
    )
  }

  const running = Boolean(status.data.running)
  const hasLastError = Boolean(status.data.last_error?.trim())
  const label = running
    ? status.data.uptime
      ? t("nav.kernelRunningWithUptime", { uptime: status.data.uptime })
      : t("dashboard.running")
    : hasLastError
      ? t("nav.kernelFailed")
      : t("dashboard.stopped")
  const title = hasLastError && !running
    ? `${label}: ${status.data.last_error}`
    : label

  return (
    <Link
      to="/dashboard"
      className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "h-7 max-w-[14rem] px-2")}
      aria-label={title}
      title={title}
    >
      <Badge
        variant={running ? "default" : "destructive"}
        className="max-w-full truncate tabular-nums"
        data-kernel-status={running ? "running" : hasLastError ? "failed" : "stopped"}
      >
        <span
          className={cn(
            "mr-1.5 inline-block size-1.5 shrink-0 rounded-full",
            running ? "bg-primary-foreground" : "bg-destructive-foreground",
          )}
          aria-hidden="true"
        />
        {label}
      </Badge>
    </Link>
  )
}
