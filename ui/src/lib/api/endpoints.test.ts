import { afterEach, describe, expect, it, vi } from "vitest"

import { api } from "@/lib/api/endpoints"

afterEach(() => vi.unstubAllGlobals())

describe("api endpoints", () => {
  it("posts login credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      token: "token",
      expires_at: "2099-01-01T00:00:00Z",
    })))
    vi.stubGlobal("fetch", fetchMock)

    await api.auth.login({ username: "admin", password: "secret" })

    expect(fetchMock).toHaveBeenCalledWith("/api/auth/login", expect.objectContaining({
      method: "POST",
      body: JSON.stringify({ username: "admin", password: "secret" }),
    }))
  })

  it("encodes dynamic path segments", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response("{}")))
    vi.stubGlobal("fetch", fetchMock)

    await api.nodes.get("node/one")

    expect(fetchMock).toHaveBeenCalledWith("/api/nodes/node%2Fone", expect.any(Object))
  })

  it("preserves partial subscription refresh status", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      status: "partial",
      data: { refreshed: 1, failed: 1 },
      error: { code: "partial_failure", message: "one subscription failed" },
      meta: null,
    }))))

    const response = await api.subscriptions.refreshAll()

    expect(response.status).toBe("partial")
    expect(response.error?.message).toBe("one subscription failed")
  })

  it("adds optional runtime delay query parameters", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response("{}")))
    vi.stubGlobal("fetch", fetchMock)

    await api.nodes.delay("node/one", { url: "https://example.com/ping", timeout: 2500 })

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/nodes/node%2Fone/delay?url=https%3A%2F%2Fexample.com%2Fping&timeout=2500",
      expect.any(Object),
    )
  })

  it("sends URLTest defaults and subscription overrides", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response("{}")))
    vi.stubGlobal("fetch", fetchMock)
    const defaults = {
      enabled: true,
      url: "https://example.com/generate_204",
      interval: "3m",
      tolerance: 50,
    }

    await api.settings.urlTestDefaults()
    await api.settings.setURLTestDefaults(defaults)
    await api.subscriptions.create({
      name: "sub",
      url: "https://example.com/sub",
      interval_min: 60,
      urltest: { enabled: false, tolerance: 0 },
    })

    expect(fetchMock).toHaveBeenCalledWith("/api/settings/urltest-defaults", expect.any(Object))
    expect(fetchMock).toHaveBeenCalledWith("/api/settings/urltest-defaults", expect.objectContaining({
      method: "PUT",
      body: JSON.stringify(defaults),
    }))
    expect(fetchMock).toHaveBeenCalledWith("/api/subscriptions/", expect.objectContaining({
      body: JSON.stringify({
        name: "sub",
        url: "https://example.com/sub",
        interval_min: 60,
        urltest: { enabled: false, tolerance: 0 },
      }),
    }))
  })
})

describe("api endpoint coverage", () => {
  it("maps every backend operation to an HTTP request", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(new Response("{}")))
    vi.stubGlobal("fetch", fetchMock)
    const node = { tag: "node", type: "vless", server: "example.com", port: 443, config: {} as const }
    const subscription = { name: "sub", url: "https://example.com/sub", interval_min: 60 }
    const test = { tag: "node", test_type: "http" as const, server: "example.com", port: 443 }

    await Promise.all([
      api.auth.logout(), api.config.get(), api.config.update({}), api.config.getRaw(), api.config.updateRaw({}),
      api.config.installDNS(), api.config.installRuleSets(), api.config.installOutbounds(), api.config.installRoute(),
      api.service.status(), api.service.start(), api.service.stop(), api.service.restart(), api.stats.history(),
      api.stats.closeAll(), api.stats.closeConnection("1"), api.import.link("vless://node"), api.import.save(node),
      api.nodes.list(), api.nodes.get("node"), api.nodes.update("node", node), api.nodes.delete("node"),
      api.nodes.groups(), api.nodes.delay("node"), api.nodes.test(test), api.nodes.testBatch([test]),
      api.nodes.results(), api.nodes.select("proxy", "node"), api.nodes.urlTest("proxy"), api.nodes.sync(),
      api.subscriptions.list(), api.subscriptions.create(subscription), api.subscriptions.get("sub"),
      api.subscriptions.update("sub", subscription), api.subscriptions.delete("sub"),
      api.subscriptions.refresh("sub"), api.subscriptions.refreshAll(), api.settings.testURL(),
      api.settings.setTestURL("https://example.com"), api.settings.autostart(), api.settings.setAutostart(true),
      api.settings.jwt(), api.settings.setJWT("secret"), api.settings.password(),
      api.settings.changePassword("current", "new-password"), api.network.interfaces(), api.runtime.version(),
      api.runtime.memory(), api.runtime.gc(), api.runtime.flushDNS(), api.runtime.flushFakeIP(),
    ])

    const paths = fetchMock.mock.calls.map(([path]) => path)
    expect(paths).toContain("/api/config/raw")
    expect(paths).toContain("/api/subscriptions/refresh-all")
    expect(paths).toContain("/api/runtime/fakeip/flush")
  })
})
