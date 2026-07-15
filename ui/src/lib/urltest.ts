const durationPattern = /(\d+(?:\.\d+)?)(ns|us|µs|ms|s|m|h)/gy
const durationScales: Record<string, number> = {
  ns: 0.000001,
  us: 0.001,
  µs: 0.001,
  ms: 1,
  s: 1000,
  m: 60000,
  h: 3600000,
}

export function isHTTPURL(value: string) {
  try {
    const parsed = new URL(value)
    return (parsed.protocol === "http:" || parsed.protocol === "https:") && Boolean(parsed.host)
  } catch {
    return false
  }
}

export function isPositiveDuration(value: string) {
  if (!value) return false
  durationPattern.lastIndex = 0
  let total = 0
  let consumed = 0
  let match = durationPattern.exec(value)
  while (match) {
    consumed = durationPattern.lastIndex
    total += Number(match[1]) * durationScales[match[2]]
    match = durationPattern.exec(value)
  }
  durationPattern.lastIndex = 0
  return consumed === value.length && total > 0
}

export function isTolerance(value: string) {
  const parsed = Number(value)
  return value.trim() !== "" && Number.isInteger(parsed) && parsed >= 0 && parsed <= 65535
}
