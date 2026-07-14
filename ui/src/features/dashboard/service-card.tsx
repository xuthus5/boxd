import { PauseIcon, PlayIcon, RotateCcwIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { ConfirmAction } from "@/components/confirm-action"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import type { ServiceStatus } from "@/lib/api/types"

interface ServiceCardProps {
  status: ServiceStatus
  pending?: string
  onAction: (action: "start" | "stop" | "restart") => void
}

function ActionButton({ action, pending, onAction }: Omit<ServiceCardProps, "status"> & { action: "start" | "stop" | "restart" }) {
  const { t } = useTranslation()
  const labels = { start: t("dashboard.start"), stop: t("dashboard.stop"), restart: t("dashboard.restart") }
  const icons = { start: PlayIcon, stop: PauseIcon, restart: RotateCcwIcon }
  const Icon = icons[action]
  const button = <Button variant={action === "stop" ? "destructive" : "outline"} size="sm" disabled={Boolean(pending)} onClick={action === "start" ? () => onAction(action) : undefined}>
      {pending === action ? <Spinner data-icon="inline-start" /> : <Icon data-icon="inline-start" />}
      {labels[action]}
    </Button>
  if (action === "start") return button
  return <ConfirmAction trigger={button} title={t("dashboard.confirmActionTitle")} description={t("dashboard.confirmActionDescription")} confirmLabel={t("dashboard.confirmAction")} confirmVariant={action === "stop" ? "destructive" : "default"} onConfirm={() => onAction(action)} />
}

export function ServiceCard({ status, pending, onAction }: ServiceCardProps) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("dashboard.service")}</CardTitle>
        <CardDescription>{t("dashboard.serviceDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="flex items-center gap-3">
        <Badge variant={status.running ? "default" : "secondary"}>{status.running ? t("dashboard.running") : t("dashboard.stopped")}</Badge>
        <span className="text-muted-foreground">{status.uptime || "—"}</span>
      </CardContent>
      <CardFooter className="flex gap-2">
        <ActionButton action="start" pending={pending} onAction={onAction} />
        <ActionButton action="stop" pending={pending} onAction={onAction} />
        <ActionButton action="restart" pending={pending} onAction={onAction} />
      </CardFooter>
    </Card>
  )
}
