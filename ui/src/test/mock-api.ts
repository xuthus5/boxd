import { vi } from "vitest"

const config = {
  inbounds: [{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }],
  outbounds: [{ tag: "proxy", type: "selector", outbounds: ["direct"] }],
  route: { final: "proxy", rules: [] },
  dns: { servers: [] },
  endpoints: [],
  experimental: {},
}

function stream(data: unknown) {
  const encoder = new TextEncoder()
  return new Response(new ReadableStream({
    start(controller) {
      controller.enqueue(encoder.encode(`data: ${JSON.stringify(data)}\n\n`))
      controller.close()
    },
  }))
}

const payloads: Record<string, unknown> = {
  "/api/auth/login": { token: "token", expires_at: "2099-01-01T00:00:00Z" },
  "/api/config/": config,
  "/api/config/raw": config,
  "/api/service/status": { running: true, uptime: "1m" },
  "/api/stats/traffic/history": { points: [{ upload_bytes: 10, download_bytes: 20, timestamp: "2026-01-01T00:00:00Z" }] },
  "/api/runtime/memory": { alloc: 1024, total: 2048, sys: 4096, num_gc: 2, heap_inuse: 512, stack_inuse: 128 },
  "/api/runtime/version": { version: "dev", kernel_version: "1.13.14" },
  "/api/nodes/": [{ tag: "hk-01", type: "vless", server: "example.com", port: 443, source: "import", raw: {} }],
  "/api/nodes/test": { tag: "hk-01", test_type: "http", success: true, latency_ms: 25 },
  "/api/import/link": { tag: "new-node", type: "vless", server: "new.example.com", port: 443, config: {} },
  "/api/subscriptions/": [{ id: "sub-1", name: "主订阅", url: "https://example.com/sub", interval_min: 60, last_updated: "2026-01-01T00:00:00Z", outbounds: [] }],
  "/api/settings/password": { defaultPassword: false },
  "/api/settings/jwt-secret": { masked: "ab********cd", present: true, length: 32 },
  "/api/settings/url-test": { url: "https://cp.cloudflare.com/" },
  "/api/settings/urltest-defaults": { enabled: true, url: "https://www.gstatic.com/generate_204", interval: "3m", tolerance: 50 },
  "/api/settings/kernel-autostart": { enabled: true },
}

function payload(path: string) { return payloads[path] ?? {} }

export function installMockAPI() {
  const fetchMock = vi.fn((input: string | URL | Request) => {
    const path = typeof input === "string"
      ? input
      : input instanceof URL
        ? input.pathname
        : new URL(input.url).pathname
    if (path === "/api/stats/logs" || path === "/api/stats/app-logs") {
      return Promise.resolve(stream({ level: "info", message: "ready", timestamp: "2026-01-01T00:00:00Z" }))
    }
    if (path === "/api/stats/connections") {
      return Promise.resolve(stream({ active_connections: 1, list: [{ id: "1", target: "example.com:443", outbound: "proxy", upload: 10, download: 20, duration: "1s" }] }))
    }
    return Promise.resolve(new Response(JSON.stringify(payload(path))))
  })
  vi.stubGlobal("fetch", fetchMock)
  return fetchMock
}
