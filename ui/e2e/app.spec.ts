import { expect, test, type Locator, type Page, type Route } from "@playwright/test"

const config = {
  inbounds: [],
  outbounds: [],
  route: {
    final: "proxy",
    rules: [
      { domain_suffix: ["example.com"], action: "route", outbound: "proxy" },
      { action: "reject" },
    ],
    rule_set: [{ type: "remote", tag: "geo", url: "https://example.com/geo.srs" }],
  },
  dns: {
    final: "legacy",
    fakeip: { enabled: true, inet4_range: "198.18.0.0/15" },
    servers: [
      { tag: "legacy", address: "local" },
      { type: "https", tag: "remote", server: "dns.example", server_port: 443 },
    ],
    rules: [
      { domain_suffix: ["example.com"], action: "route", server: "remote" },
      { action: "reject" },
    ],
  },
  endpoints: [],
  experimental: {},
}
const nodes = [
  { tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import", raw: {} },
  { tag: "us-01", type: "trojan", server: "us.example.com", port: 443, source: "subscription", source_name: "主订阅", raw: {} },
]
const apiBodies: Record<string, unknown> = {
  "/api/auth/login": { token: "token", expires_at: "2099-01-01T00:00:00Z" },
  "/api/service/status": { running: true, uptime: "1m" },
  "/api/stats/traffic/history": { points: [] },
  "/api/runtime/memory": { alloc: 1024, total: 2048, sys: 4096, num_gc: 1, heap_inuse: 512, stack_inuse: 128 },
  "/api/runtime/version": { version: "dev", kernel_version: "1.13.14" },
  "/api/config/": config,
  "/api/config/raw": config,
  "/api/nodes/": nodes,
  "/api/nodes/test-results": {
    "hk-01": { tcp: { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 } },
  },
  "/api/nodes/groups": { groups: [{ type: "selector", tag: "proxy", now: "hk-01", all: ["hk-01", "us-01"] }] },
  "/api/subscriptions/": [{
    id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60,
    last_updated: "2026-01-01T00:00:00Z", outbounds: nodes.filter((node) => node.source === "subscription"),
  }],
  "/api/settings/urltest-defaults": {
    enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50,
  },
  "/api/nodes/test": { tag: "hk-01", test_type: "tcp", success: true, latency_ms: 18 },
  "/api/nodes/test-batch": { results: [] },
}

function bodyFor(path: string) { return apiBodies[path] ?? {} }

async function fulfillAPI(route: Route) {
  const request = route.request()
  const path = new URL(request.url()).pathname
  if (["/api/stats/traffic", "/api/stats/logs", "/api/stats/app-logs", "/api/stats/connections"].includes(path)) {
    const event = path.includes("logs") ? { level: "info", message: "ready" } : {}
    await route.fulfill({ contentType: "text/event-stream", body: `data: ${JSON.stringify(event)}\n\n` })
    return
  }
  if (request.method() === "PUT") {
    await route.fulfill({ contentType: "application/json", body: JSON.stringify({ status: "ok", data: null, error: null, meta: null }) })
    return
  }
  await route.fulfill({ contentType: "application/json", body: JSON.stringify(bodyFor(path)) })
}

async function login(page: Page) {
  await page.goto("/login")
  await page.getByLabel(/用户名|Username/).fill("admin")
  await page.getByLabel(/密码|Password/).fill("secret")
  await page.getByRole("button", { name: /登录|Sign in/ }).click()
  await expect(page.getByRole("heading", { name: /仪表盘|Dashboard/ })).toBeVisible()
}

async function expectPageFitsViewport(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
  const cards = page.locator("main [data-slot=card]")
  const boxes = await cards.evaluateAll((elements) => elements.map((element) => element.getBoundingClientRect().toJSON()))
  expect(boxes.length).toBeGreaterThan(0)
  for (const box of boxes) {
    expect(box.x).toBeGreaterThanOrEqual(0)
    expect(box.x + box.width).toBeLessThanOrEqual(320)
  }
}

async function openMenuWithKeyboard(page: Page, name: string) {
  const trigger = page.getByRole("button", { name })
  await trigger.focus()
  await page.keyboard.press("Shift+Tab")
  await page.keyboard.press("Tab")
  await expect(trigger).toBeFocused()
  await page.keyboard.press("Enter")
  await expect(page.getByRole("menu")).toBeVisible()
  await page.keyboard.press("Escape")
}

async function expectKeyboardTabs(page: Page, dialog: Locator, first: string, next: string) {
  const firstTab = dialog.getByRole("tab", { name: first })
  await firstTab.focus()
  await page.keyboard.press("Tab")
  await page.keyboard.press("Shift+Tab")
  await expect(firstTab).toBeFocused()
  await firstTab.press("ArrowRight")
  const nextTab = dialog.getByRole("tab", { name: next })
  await expect(nextTab).toBeFocused()
  await expect(nextTab).toHaveAttribute("aria-selected", "true")
  await expect(dialog.getByRole("tabpanel")).toBeVisible()
}

async function expectDialogFitsViewport(dialog: Locator) {
  const viewport = dialog.page().viewportSize()
  const box = await dialog.boundingBox()
  expect(viewport).not.toBeNull()
  expect(box).not.toBeNull()
  expect(box!.x).toBeGreaterThanOrEqual(0)
  expect(box!.y).toBeGreaterThanOrEqual(0)
  expect(box!.x + box!.width).toBeLessThanOrEqual(viewport!.width)
  expect(box!.y + box!.height).toBeLessThanOrEqual(viewport!.height)
  expect(await dialog.evaluate((element) => element.scrollWidth <= element.clientWidth)).toBe(true)
}

async function checkRoutePolicy(page: Page) {
  await page.goto("/policy/route")
  await expect(page.getByRole("heading", { name: "Route" })).toBeVisible()
  await expectPageFitsViewport(page)
  await openMenuWithKeyboard(page, "More actions for route rule 1")
  await page.getByRole("button", { name: "Edit route rule 1" }).click()
  const dialog = page.getByRole("dialog", { name: "Edit route rule 1" })
  await expectDialogFitsViewport(dialog)
  await expectKeyboardTabs(page, dialog, "Basics and network", "Domains and addresses")
  await dialog.getByRole("button", { name: "Cancel" }).click()
  await expectPageFitsViewport(page)
}

async function checkDNSPolicy(page: Page) {
  await page.goto("/policy/dns")
  await expect(page.getByRole("heading", { name: "DNS" })).toBeVisible()
  await expectPageFitsViewport(page)
  await openMenuWithKeyboard(page, "More actions for DNS server legacy")
  await page.getByRole("button", { name: "Edit DNS server legacy" }).click()
  const dialog = page.getByRole("dialog", { name: "Edit DNS server" })
  await expectDialogFitsViewport(dialog)
  await expectKeyboardTabs(page, dialog, "Basics", "Dialing and resolution")
  await dialog.getByRole("button", { name: "Cancel" }).click()
  await expectPageFitsViewport(page)
}

test("smoke: login, navigation, log tabs, and raw save", async ({ page }) => {
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await login(page)

  await page.getByRole("link", { name: "日志" }).click()
  await expect(page.getByRole("tab", { name: "内核日志" })).toBeVisible()
  await page.getByRole("tab", { name: "应用日志" }).click()
  await expect(page.getByRole("tabpanel").getByText("ready")).toBeVisible()

  await page.getByRole("link", { name: "完整配置" }).click()
  await page.getByRole("button", { name: "保存完整配置" }).click()
  const response = page.waitForResponse((item) => item.url().endsWith("/api/config/raw") && item.request().method() === "PUT")
  await page.getByRole("button", { name: "确认覆盖" }).click()
  await expect((await response).ok()).toBe(true)
})

test("mobile node and subscription cards stay within the viewport", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await login(page)

  await page.goto("/nodes")
  const nodesRegion = page.getByRole("region", { name: "所有节点" })
  const nodeCard = nodesRegion.getByRole("article", { name: "hk-01" })
  await expect(nodeCard).toBeVisible()
  await expect(nodesRegion.getByRole("button", { name: "批量测速" })).toBeVisible()
  await expect(nodeCard.getByText("18 ms")).toBeVisible()
  await nodeCard.getByRole("button", { name: "测速" }).click()
  await expect(page.getByRole("menuitem", { name: "全部" })).toBeVisible()
  await expect(page.getByRole("menuitem", { name: "TCP" })).toBeVisible()
  await expect(page.getByRole("menuitem", { name: "HTTP" })).toBeVisible()
  await expect(page.getByRole("menuitem", { name: "ICMP" })).toBeVisible()
  await expect(page.locator("[data-slot=card] [data-slot=card]")).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(0)

  await page.goto("/subscriptions")
  await expect(page.getByRole("button", { name: "导入节点" })).toBeVisible()
  await expect(page.getByRole("article", { name: "主订阅" })).toBeVisible()
  await expect(page.locator("[data-slot=card] [data-slot=card]")).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(0)
})

test("subscription URLTest policy fits a 320px viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 568 })
  await page.addInitScript(() => {
    localStorage.setItem("boxui.preferences.v1", JSON.stringify({ theme: "system", language: "en" }))
  })
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await login(page)

  await page.goto("/subscriptions")
  await page.getByRole("button", { name: "Add subscription" }).click()
  const dialog = page.getByRole("dialog")
  await expect(dialog.getByRole("button", { name: "Inherit global" })).toBeVisible()
  await expect(dialog.getByRole("button", { name: "Enable" })).toBeVisible()
  await expect(dialog.getByRole("button", { name: "Disable" })).toBeVisible()
  expect(await dialog.evaluate((element) => element.scrollWidth - element.clientWidth)).toBe(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBe(0)
})

test("route and DNS policy editors fit 320px and remain keyboard reachable", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 568 })
  await page.addInitScript(() => {
    localStorage.setItem("boxui.preferences.v1", JSON.stringify({ theme: "system", language: "en" }))
  })
  await page.route("http://127.0.0.1:4173/api/**", fulfillAPI)
  await login(page)

  await checkRoutePolicy(page)
  await checkDNSPolicy(page)
})
