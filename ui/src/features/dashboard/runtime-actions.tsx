import { DatabaseZapIcon, RotateCcwIcon, Trash2Icon } from "lucide-react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"

interface RuntimeActionsProps {
  onGC: () => void
  onFlushDNS: () => void
  onFlushFakeIP: () => void
  pending: boolean
}

export function RuntimeActions({ onGC, onFlushDNS, onFlushFakeIP, pending }: RuntimeActionsProps) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("dashboard.maintenance")}</CardTitle>
        <CardDescription>{t("dashboard.maintenanceDescription")}</CardDescription>
      </CardHeader>
      <CardContent />
      <CardFooter className="flex flex-wrap gap-2">
        <Button variant="outline" size="sm" disabled={pending} onClick={onGC}><RotateCcwIcon data-icon="inline-start" />GC</Button>
        <Button variant="outline" size="sm" disabled={pending} onClick={onFlushDNS}><DatabaseZapIcon data-icon="inline-start" />{t("dashboard.flushDNS")}</Button>
        <Button variant="outline" size="sm" disabled={pending} onClick={onFlushFakeIP}><Trash2Icon data-icon="inline-start" />{t("dashboard.flushFakeIP")}</Button>
      </CardFooter>
    </Card>
  )
}
