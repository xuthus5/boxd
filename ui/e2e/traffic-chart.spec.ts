import { expect, test, type Page, type Route } from "@playwright/test"

interface TrafficSample {
  upload_bytes: number
  download_bytes: number
  timestamp: string
}

const initialPoints: TrafficSample[] = [
  { upload_bytes: 0, download_bytes: 0, timestamp: "2026-01-01T00:00:00Z" },
  { upload_bytes: 1024, download_bytes: 2048, timestamp: "2026-01-01T00:00:01Z" },
]

function apiBody(path: string) {
  if (path === "/readyz") return { status: "ready" }
  if (path === "/api/service/status") return { running: true, uptime: "1m" }
  if (path === "/api/stats/traffic/history") return { points: initialPoints }
  if (path === "/api/runtime/memory") return { alloc: 1024, total: 2048, sys: 4096, num_gc: 1, heap_inuse: 512, stack_inuse: 128 }
  if (path === "/api/runtime/version") return { version: "dev", kernel_version: "1.13.14" }
  if (path === "/api/config/diagnostics") return {
    status: "healthy",
    checked_at: "2026-01-01T00:00:00Z",
    summary: { errors: 0, warnings: 0 },
    counts: { inbounds: 0, outbounds: 0, endpoints: 0, route_rules: 0, rule_sets: 0, dns_servers: 0, dns_rules: 0 },
    features: { tun: false, clash_api: false, cache_file: false, fakeip: false, selector: false, urltest: false, wireguard: false, remote_rule_set: false },
    issues: [],
  }
  if (path === "/api/config/rule-sets/status") return []
  if (path === "/api/config/rule-sets/auto-update") return { enabled: false, interval: "24h" }
  if (path === "/api/config/apply-history") return { events: [] }
  if (path === "/api/nodes/groups") return { groups: [] }
  if (path === "/api/config/" || path === "/api/config") return { inbounds: [], outbounds: [], route: { rules: [] }, experimental: {} }
  if (path === "/api/subscriptions/" || path === "/api/subscriptions") return []
  if (path === "/api/runtime/clash-mode") return { mode: "Rule", mode_list: ["Rule", "Global", "Direct"] }
  return {}
}

async function fulfillAPI(route: Route) {
  const path = new URL(route.request().url()).pathname
  if (["/api/stats/logs", "/api/stats/app-logs", "/api/stats/connections"].includes(path)) {
    const event = path === "/api/stats/connections" ? { active_connections: 0, list: [] } : { level: "info", message: "ready" }
    await route.fulfill({ contentType: "text/event-stream", body: `data: ${JSON.stringify(event)}\n\n` })
    return
  }
  await route.fulfill({ contentType: "application/json", body: JSON.stringify(apiBody(path)) })
}

async function installTrafficStream(page: Page) {
  await page.addInitScript(() => {
    const originalFetch = window.fetch.bind(window)
    let controller: ReadableStreamDefaultController<Uint8Array> | undefined
    const encoder = new TextEncoder()
    const scope = window as typeof window & { __pushTraffic?: (sample: TrafficSample) => void }
    scope.__pushTraffic = (sample) => {
      controller?.enqueue(encoder.encode(`data: ${JSON.stringify(sample)}\n\n`))
    }
    window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      const requestURL = typeof input === "string"
        ? new URL(input, window.location.origin)
        : input instanceof URL ? input : new URL(input.url)
      if (requestURL.pathname !== "/api/stats/traffic") return originalFetch(input, init)
      return new Response(new ReadableStream<Uint8Array>({ start: (next) => { controller = next } }), {
        headers: { "Content-Type": "text/event-stream" },
      })
    }
  })
}

test("live traffic curve slides without replacing the SVG path", async ({ page }) => {
  await page.addInitScript(() => {
    sessionStorage.setItem("boxd.session.v1", JSON.stringify({ token: "token", expiresAt: "2099-01-01T00:00:00Z" }))
  })
  await installTrafficStream(page)
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await page.goto("/dashboard")

  const uploadCurve = page.locator('path.traffic-chart-curve[data-series="upload_rate"]')
  await expect(uploadCurve).toBeVisible()
  await expect(uploadCurve.locator("..")).toHaveClass(/transition-transform/)
  await page.waitForFunction(() => {
    const path = document.querySelector('path.traffic-chart-curve[data-series="upload_rate"]')
    const motion = path?.parentElement
    if (!motion) return false
    const targetX = new DOMMatrix(motion.style.transform).m41
    const computedX = new DOMMatrix(getComputedStyle(motion).transform).m41
    return Math.abs(targetX - computedX) < 0.5
  })
  const initial = await uploadCurve.evaluate((path) => {
    const motion = path.parentElement as SVGGElement
    const scope = window as typeof window & { __trafficCurve?: Element }
    scope.__trafficCurve = path
    return {
      opacity: getComputedStyle(path).opacity,
      targetX: new DOMMatrix(motion.style.transform).m41,
      transform: motion.style.transform,
    }
  })

  await page.evaluate((sample) => {
    const scope = window as typeof window & { __pushTraffic?: (next: TrafficSample) => void }
    scope.__pushTraffic?.(sample)
  }, { upload_bytes: 3072, download_bytes: 6144, timestamp: "2026-01-01T00:00:02Z" } satisfies TrafficSample)
  await page.waitForFunction((transform) => {
    const path = document.querySelector('path.traffic-chart-curve[data-series="upload_rate"]')
    return path?.parentElement?.style.transform !== transform
  }, initial.transform)
  await page.waitForTimeout(100)

  const updated = await uploadCurve.evaluate((path) => {
    const motion = path.parentElement as SVGGElement
    const style = getComputedStyle(motion)
    const scope = window as typeof window & { __trafficCurve?: Element }
    return {
      computedX: new DOMMatrix(style.transform).m41,
      duration: style.transitionDuration,
      opacity: getComputedStyle(path).opacity,
      sameNode: scope.__trafficCurve === path,
      targetX: new DOMMatrix(motion.style.transform).m41,
      transitionProperty: style.transitionProperty,
    }
  })
  const lowerBound = Math.min(initial.targetX, updated.targetX)
  const upperBound = Math.max(initial.targetX, updated.targetX)

  expect(updated.sameNode).toBe(true)
  expect(updated.opacity).toBe(initial.opacity)
  expect(updated.transitionProperty).toContain("transform")
  expect(updated.duration).toBe("0.8s")
  expect(updated.targetX).not.toBe(initial.targetX)
  expect(updated.computedX).toBeGreaterThan(lowerBound)
  expect(updated.computedX).toBeLessThan(upperBound)
})
