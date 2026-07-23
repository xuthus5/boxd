export function formatRelativeTime(value: string, now = Date.now(), locale = "zh-CN") {
  const time = Date.parse(value)
  if (Number.isNaN(time)) return ""
  const deltaSec = Math.round((time - now) / 1000)
  const abs = Math.abs(deltaSec)
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" })
  if (abs < 60) return rtf.format(deltaSec, "second")
  if (abs < 3600) return rtf.format(Math.round(deltaSec / 60), "minute")
  if (abs < 86400) return rtf.format(Math.round(deltaSec / 3600), "hour")
  if (abs < 86400 * 30) return rtf.format(Math.round(deltaSec / 86400), "day")
  if (abs < 86400 * 365) return rtf.format(Math.round(deltaSec / (86400 * 30)), "month")
  return rtf.format(Math.round(deltaSec / (86400 * 365)), "year")
}
