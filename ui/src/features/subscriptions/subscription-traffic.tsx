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

export function SubscriptionTrafficBadges({ traffic }: { traffic?: SubscriptionTraffic }) {
  const { t } = useTranslation()
  if (!traffic) return null
  const expired = Boolean(traffic.expire && Date.parse(traffic.expire) <= Date.now())
  return (
    <div className="flex flex-wrap gap-2">
      <Badge variant="outline">{trafficLabel(traffic, t)}</Badge>
      <Badge variant={expired ? "destructive" : "outline"}>{expireLabel(traffic, t)}</Badge>
    </div>
  )
}
