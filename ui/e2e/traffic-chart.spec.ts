import { expect, test, type Page, type Route } from "@playwright/test"

interface TrafficSample {
  upload_bytes: number
  download_bytes: number
  timestamp: string
}

const trafficStartedAt = Date.parse("2026-01-01T00:00:00Z")

function trafficSample(index: number): TrafficSample {
  const uploadBytes = ((index * (index + 1)) / 2) * 1024
  return {
    upload_bytes: uploadBytes,
    download_bytes: uploadBytes * 2,
    timestamp: new Date(trafficStartedAt + (index * 1000)).toISOString(),
  }
}

const initialPoints = Array.from({ length: 60 }, (_, index) => trafficSample(index))
const livePoints = Array.from({ length: 62 }, (_, index) => trafficSample(index + 60))

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

test("live traffic curve rolls the full window without redrawing history", async ({ page }) => {
  await page.addInitScript(() => {
    sessionStorage.setItem("boxd.session.v1", JSON.stringify({ token: "token", expiresAt: "2099-01-01T00:00:00Z" }))
  })
  await installTrafficStream(page)
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await page.goto("/dashboard")

  const uploadSegments = page.locator('path.traffic-chart-curve[data-series="upload_rate"]')
  const firstSegment = uploadSegments.first()
  await expect(firstSegment).toBeVisible()
  await expect(firstSegment.locator("..")).toHaveClass(/transition-transform/)
  const chart = page.locator('[data-slot="chart"]').first()
  await chart.scrollIntoViewIfNeeded()
  const chartBox = await chart.boundingBox()
  if (!chartBox) throw new Error("traffic chart bounds unavailable")
  await page.mouse.move(chartBox.x + (chartBox.width * 0.15), chartBox.y + (chartBox.height * 0.5))
  await expect(page.locator(".recharts-tooltip-wrapper")).toContainText("/s")
  await page.evaluate((samples) => {
    const scope = window as typeof window & { __pushTraffic?: (next: TrafficSample) => void }
    for (const sample of samples) scope.__pushTraffic?.(sample)
  }, livePoints.slice(0, 61))
  await page.waitForFunction(() => {
    const segments = document.querySelectorAll<SVGPathElement>('path.traffic-chart-curve[data-series="upload_rate"]')
    return segments.item(segments.length - 1)?.getAttribute("d")?.startsWith("M119000,")
  })
  await page.waitForFunction(() => {
    const path = document.querySelector('path.traffic-chart-curve[data-series="upload_rate"]')
    const motion = path?.parentElement
    if (!motion) return false
    const targetX = new DOMMatrix(motion.style.transform).m41
    const computedX = new DOMMatrix(getComputedStyle(motion).transform).m41
    return Math.abs(targetX - computedX) < 0.5
  })
  const initial = await firstSegment.evaluate((path) => {
    const motion = path.parentElement as SVGGElement
    return {
      count: document.querySelectorAll('path.traffic-chart-curve[data-series="upload_rate"]').length,
      opacity: getComputedStyle(path).opacity,
      targetX: new DOMMatrix(motion.style.transform).m41,
      transform: motion.style.transform,
    }
  })
  await page.evaluate(() => {
    const selector = 'path.traffic-chart-curve[data-series="upload_rate"]'
    const segments = Array.from(document.querySelectorAll<SVGPathElement>(selector)).slice(1)
    const state = {
      d: segments.map((segment) => segment.getAttribute("d")),
      dMutations: 0,
      rechartsLineReplacements: 0,
      segments,
    }
    const observer = new MutationObserver((records) => {
      state.dMutations += records.filter((record) => record.attributeName === "d").length
    })
    for (const segment of segments) observer.observe(segment, { attributes: true, attributeFilter: ["d"] })
    const chartObserver = new MutationObserver((records) => {
      state.rechartsLineReplacements += records.filter((record) => (
        record.type === "childList"
        && record.target instanceof Element
        && record.target.classList.contains("recharts-line")
      )).length
    })
    const chart = document.querySelector('[data-slot="chart"]')
    if (chart) chartObserver.observe(chart, { childList: true, subtree: true })
    const scope = window as typeof window & { __trafficSegments?: typeof state }
    scope.__trafficSegments = state
  })

  await page.evaluate((sample) => {
    const scope = window as typeof window & { __pushTraffic?: (next: TrafficSample) => void }
    scope.__pushTraffic?.(sample)
  }, livePoints[61])
  await page.waitForFunction((transform) => {
    const path = document.querySelector('path.traffic-chart-curve[data-series="upload_rate"]')
    return path?.parentElement?.style.transform !== transform
  }, initial.transform)
  await expect(uploadSegments).toHaveCount(initial.count)
  await page.waitForTimeout(100)

  const updated = await firstSegment.evaluate((path) => {
    const motion = path.parentElement as SVGGElement
    const style = getComputedStyle(motion)
    const scope = window as typeof window & {
      __trafficSegments?: {
        d: Array<string | null>
        dMutations: number
        rechartsLineReplacements: number
        segments: SVGPathElement[]
      }
    }
    const state = scope.__trafficSegments
    return {
      computedX: new DOMMatrix(style.transform).m41,
      dMutations: state?.dMutations,
      duration: style.transitionDuration,
      historicalGeometryStable: state?.segments.every((segment, index) => (
        segment.isConnected && segment.getAttribute("d") === state.d[index]
      )),
      opacity: getComputedStyle(path).opacity,
      rechartsLineReplacements: state?.rechartsLineReplacements,
      targetX: new DOMMatrix(motion.style.transform).m41,
      transitionProperty: style.transitionProperty,
    }
  })
  const lowerBound = Math.min(initial.targetX, updated.targetX)
  const upperBound = Math.max(initial.targetX, updated.targetX)

  expect(updated.historicalGeometryStable).toBe(true)
  expect(updated.dMutations).toBe(0)
  expect(updated.opacity).toBe(initial.opacity)
  expect(updated.rechartsLineReplacements).toBe(0)
  expect(updated.transitionProperty).toContain("transform")
  expect(updated.duration).toBe("1s")
  expect(updated.targetX).not.toBe(initial.targetX)
  expect(updated.computedX).toBeGreaterThan(lowerBound)
  expect(updated.computedX).toBeLessThan(upperBound)
})
