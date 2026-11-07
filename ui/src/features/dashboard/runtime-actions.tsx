import { DatabaseZapIcon, RotateCcwIcon, Trash2Icon, WrenchIcon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

interface RuntimeActionsProps {
  onGC: () => void
  onFlushDNS: () => void
  onFlushFakeIP: () => void
  pending: boolean
}

export function RuntimeActions({ onGC, onFlushDNS, onFlushFakeIP, pending }: RuntimeActionsProps) {
  const { t } = useTranslation()
  return (
    <Card size="sm">
      <CardHeader className="gap-1.5">
        <CardTitle className="flex min-w-0 items-center gap-2 truncate"><span className="flex size-8 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground ring-1 ring-border"><WrenchIcon className="size-4" /></span><span className="truncate">{t("dashboard.maintenance")}</span></CardTitle>
        <CardDescription>{t("dashboard.maintenanceDescription")}</CardDescription>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Button variant="outline" size="sm" className="h-8" disabled={pending} onClick={onGC}>
          <RotateCcwIcon data-icon="inline-start" />GC
        </Button>
        <Button variant="outline" size="sm" className="h-8" disabled={pending} onClick={onFlushDNS}>
          <DatabaseZapIcon data-icon="inline-start" />{t("dashboard.flushDNS")}
        </Button>
        <Button variant="outline" size="sm" className="h-8" disabled={pending} onClick={onFlushFakeIP}>
          <Trash2Icon data-icon="inline-start" />{t("dashboard.flushFakeIP")}
        </Button>
      </CardContent>
    </Card>
  )
}
