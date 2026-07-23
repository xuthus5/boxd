/** Dirty-state helpers for settings forms. */

import type { RuleSetAutoUpdate, URLTestDefaults } from "@/lib/api/types"
import { isHTTPURL, isPositiveDuration, isTolerance } from "@/lib/urltest"

export function isTestURLDirty(current: string, saved: string) {
  return current.trim() !== saved.trim()
}

export function isTestURLReady(current: string, saved: string) {
  const next = current.trim()
  return isTestURLDirty(current, saved) && next.length > 0 && isHTTPURL(next)
}

export function isURLTestDefaultsDirty(
  current: Pick<URLTestDefaults, "enabled" | "url" | "interval" | "tolerance">,
  saved: URLTestDefaults,
) {
  return (
    current.enabled !== saved.enabled
    || current.url.trim() !== saved.url.trim()
    || current.interval.trim() !== saved.interval.trim()
    || current.tolerance !== saved.tolerance
  )
}

export function isURLTestDefaultsReady(
  current: Pick<URLTestDefaults, "enabled" | "url" | "interval" | "tolerance"> & { toleranceInput: string },
  saved: URLTestDefaults,
) {
  if (!isHTTPURL(current.url)) return false
  if (!isPositiveDuration(current.interval)) return false
  if (!isTolerance(current.toleranceInput)) return false
  return isURLTestDefaultsDirty(
    {
      enabled: current.enabled,
      url: current.url,
      interval: current.interval,
      tolerance: current.tolerance,
    },
    saved,
  )
}

export function isRuleSetAutoUpdateDirty(
  current: RuleSetAutoUpdate,
  saved: RuleSetAutoUpdate,
) {
  return (
    current.enabled !== saved.enabled
    || current.interval.trim() !== (saved.interval || "").trim()
  )
}

export function isRuleSetAutoUpdateReady(
  current: RuleSetAutoUpdate,
  saved: RuleSetAutoUpdate,
  intervalValid: boolean,
) {
  return intervalValid && isRuleSetAutoUpdateDirty(current, saved)
}
