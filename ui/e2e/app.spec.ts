import { expect, test, type Page, type Route } from "@playwright/test"

const config = { inbounds: [], outbounds: [], route: {}, dns: {}, endpoints: [], experimental: {} }

function bodyFor(path: string) {
  if (path === "/api/auth/login") return { token: "token", expires_at: "2099-01-01T00:00:00Z" }
  if (path === "/api/service/status") return { running: true, uptime: "1m" }
  if (path === "/api/stats/traffic/history") return { points: [] }
  if (path === "/api/runtime/memory") return { alloc: 1024, total: 2048, sys: 4096, num_gc: 1, heap_inuse: 512, stack_inuse: 128 }
  if (path === "/api/runtime/version") return { version: "dev", kernel_version: "1.13.14" }
  if (path === "/api/config/" || path === "/api/config/raw") return config
  return {}
}

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
