import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import {
  shouldShowStreamStatus,
  streamStatusLabelKey,
  streamStatusVariant,
} from "@/features/observability/stream-status"
import type { StreamConnectionStatus } from "@/features/observability/use-stream-buffer"
import { cn } from "@/lib/utils"

export function StreamStatusBadge({
  status,
  paused,
  className,
}: {
  status: StreamConnectionStatus
  paused: boolean
  className?: string
}) {
  const { t } = useTranslation()
  if (!shouldShowStreamStatus(status, paused)) return null
  return (
    <Badge
      variant={streamStatusVariant(status, paused)}
      className={cn(className)}
      data-stream-status={paused ? "paused" : status}
    >
      {t(streamStatusLabelKey(status, paused))}
    </Badge>
  )
}
