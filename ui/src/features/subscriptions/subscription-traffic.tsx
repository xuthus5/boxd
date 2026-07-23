import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { formatBytes } from "@/features/dashboard/format"
import type { SubscriptionTraffic } from "@/lib/api/types"

function trafficLabel(traffic: SubscriptionTraffic, t: (key: string, options?: Record<string, string>) => string) {
  const used = formatBytes(traffic.upload + traffic.download)
  if (traffic.total > 0) {
    return t("subscriptions.trafficUsed", { used, total: formatBytes(traffic.total) })
  }
  return t("subscriptions.trafficUnlimited", { used })
}

function expireLabel(traffic: SubscriptionTraffic, t: (key: string, options?: Record<string, string>) => string) {
  if (!traffic.expire) return t("subscriptions.expireNever")
  const date = new Date(traffic.expire)
  if (Number.isNaN(date.getTime())) return t("subscriptions.expireNever")
  return t("subscriptions.expireAt", { date: date.toLocaleString() })
}

export function SubscriptionTrafficBadges({ traffic, now }: { traffic?: SubscriptionTraffic; now?: number }) {
  const { t } = useTranslation()
  // Lazy state init is not treated as impure render evaluation.
  const [mountedAt] = useState(() => Date.now())
  if (!traffic) return null
  const clock = now ?? mountedAt
  const expired = Boolean(traffic.expire && Date.parse(traffic.expire) <= clock)
  return (
    <div className="flex flex-wrap gap-2">
      <Badge variant="outline">{trafficLabel(traffic, t)}</Badge>
      <Badge variant={expired ? "destructive" : "outline"}>{expireLabel(traffic, t)}</Badge>
    </div>
  )
}
