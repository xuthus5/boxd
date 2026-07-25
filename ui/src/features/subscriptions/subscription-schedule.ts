import type { Subscription } from "@/lib/api/types"

type SubscriptionScheduleInput = Pick<Subscription, "interval_min"> & Partial<Pick<Subscription, "last_updated" | "error_at">>

export interface SubscriptionRefreshSchedule {
  intervalMinutes: number | null
  nextAt: number | null
  due: boolean
}

function parseTimestamp(value?: string): number | null {
  if (!value) return null
  const timestamp = Date.parse(value)
  return Number.isNaN(timestamp) ? null : timestamp
}

export function subscriptionRefreshSchedule(
  item: SubscriptionScheduleInput,
  now = Date.now(),
): SubscriptionRefreshSchedule {
  const intervalMinutes = Number.isFinite(item.interval_min) && item.interval_min > 0
    ? item.interval_min
    : null
  if (intervalMinutes === null) return { intervalMinutes: null, nextAt: null, due: false }

  const attempts = [parseTimestamp(item.last_updated), parseTimestamp(item.error_at)]
    .filter((value): value is number => value !== null)
  if (attempts.length === 0) return { intervalMinutes, nextAt: null, due: true }

  const nextAt = Math.max(...attempts) + intervalMinutes * 60_000
  return { intervalMinutes, nextAt, due: nextAt <= now }
}
