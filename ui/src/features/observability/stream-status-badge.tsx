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
  error,
  className,
}: {
  status: StreamConnectionStatus
  paused: boolean
  error?: string
  className?: string
}) {
  const { t } = useTranslation()
  const hasError = Boolean(error?.trim())
  if (!shouldShowStreamStatus(status, paused, hasError)) return null
  return (
    <Badge
      variant={streamStatusVariant(status, paused, hasError)}
      className={cn(className)}
      data-stream-status={paused ? "paused" : hasError && status === "closed" ? "error" : status}
    >
      {t(streamStatusLabelKey(status, paused, hasError))}
    </Badge>
  )
}
