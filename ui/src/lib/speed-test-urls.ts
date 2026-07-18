/** Prefetched HTTP probe endpoints for node speed tests. Cloudflare first. */
export const SPEED_TEST_URL_PRESETS = [
  "https://cp.cloudflare.com/",
  "https://www.gstatic.com/generate_204",
  "https://www.google-analytics.com/generate_204",
  "https://www.google.com/generate_204",
  "https://connectivitycheck.gstatic.com/generate_204",
  "https://spectrum.s3.amazonaws.com/kindle-wifi/wifistub.html",
  "https://captive.apple.com",
  "https://www.apple.com/library/test/success.html",
  "https://detectportal.firefox.com/success.txt",
  "https://www.v2ex.com/generate_204",
] as const

export const DEFAULT_SPEED_TEST_URL = SPEED_TEST_URL_PRESETS[0]

export function isSpeedTestURLPreset(url: string): boolean {
  return (SPEED_TEST_URL_PRESETS as readonly string[]).includes(url)
}

export function resolveSpeedTestURLMode(url: string): string {
  if (!url) return DEFAULT_SPEED_TEST_URL
  return isSpeedTestURLPreset(url) ? url : "manual"
}

export function resolveInitialSpeedTestURL(url: string): string {
  return url || DEFAULT_SPEED_TEST_URL
}
