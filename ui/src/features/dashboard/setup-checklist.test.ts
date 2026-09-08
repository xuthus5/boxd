import { describe, expect, it } from "vitest"

import {
  buildSetupSteps,
  hasClashAPI,
  hasLocalInbound,
  hasProxyOutbound,
  hasRouteRules,
  hasSubscriptions,
  setupProgress,
} from "@/features/dashboard/setup-checklist"
import type { SingBoxConfig, Subscription } from "@/lib/api/types"
import { setupStatus } from "@/test/setup-fixtures"

const emptyConfig = {} as SingBoxConfig

describe("setup-checklist", () => {
  it("detects inbound/outbound/route/clash readiness", () => {
    expect(hasLocalInbound(emptyConfig)).toBe(false)
    expect(hasLocalInbound({ inbounds: [{ type: "mixed", tag: "mixed-in" }] } as SingBoxConfig)).toBe(true)
    expect(hasProxyOutbound({ outbounds: [{ type: "direct", tag: "direct" }] } as SingBoxConfig)).toBe(false)
    expect(hasProxyOutbound({
      outbounds: [
        { type: "selector", tag: "proxy", outbounds: ["hk"] },
        { type: "vless", tag: "hk" },
      ],
    } as SingBoxConfig)).toBe(true)
    expect(hasRouteRules({ route: { rules: [{ outbound: "proxy" }] } } as SingBoxConfig)).toBe(true)
    expect(hasClashAPI({ experimental: { clash_api: { external_controller: "127.0.0.1:9090" } } } as SingBoxConfig)).toBe(true)
  })

  it("recognizes internal Clash control without opening a listener", () => {
    expect(hasClashAPI({ experimental: { clash_api: { default_mode: "rule" } } } as SingBoxConfig)).toBe(true)
    expect(hasClashAPI({ experimental: { clash_api: {} } } as SingBoxConfig)).toBe(true)
    expect(hasClashAPI({ experimental: { clash_api: null } } as SingBoxConfig)).toBe(false)
    expect(hasClashAPI(emptyConfig)).toBe(false)
  })

  it("keeps proxy setup pending until a real node is added", () => {
    expect(hasProxyOutbound({ outbounds: [
      { type: "direct", tag: "direct" },
      { type: "block", tag: "block" },
      { type: "selector", tag: "proxy", outbounds: ["block"] },
    ] } as SingBoxConfig)).toBe(false)
    expect(hasProxyOutbound({ outbounds: [
      { type: "selector", tag: "proxy", outbounds: ["missing"] },
    ] } as SingBoxConfig)).toBe(false)
    expect(hasProxyOutbound({ outbounds: [{ tag: "empty" }, { type: "vless" }] } as SingBoxConfig)).toBe(false)
  })

  it("builds progress across steps", () => {
    const steps = buildSetupSteps({
      status: { running: true, uptime: "1m" },
      config: {
        inbounds: [{ type: "tun", tag: "tun-in" }],
        outbounds: [{ type: "vless", tag: "node" }],
        route: { rules: [] },
        experimental: {},
      } as SingBoxConfig,
      subscriptions: [],
    })
    const progress = setupProgress(steps)
    expect(progress.done).toBe(3)
    expect(progress.total).toBe(4)
    expect(progress.complete).toBe(false)
    expect(steps.map((step) => step.id)).toEqual(["modules", "nodes", "access", "kernel"])
    expect(steps.find((step) => step.id === "nodes")?.done).toBe(true)
  })

  it("does not accept failed or empty subscriptions as imported nodes", () => {
    const subscription = { id: "s", name: "test", url: "https://example.com/sub", interval_min: 60,
      last_updated: "2026-01-01T00:00:00Z", outbounds: [] }
    expect(hasSubscriptions([subscription])).toBe(false)
    expect(hasSubscriptions([{ ...subscription, error: "timeout" }])).toBe(false)
    expect(hasSubscriptions(undefined)).toBe(false)
    expect(hasSubscriptions([{ ...subscription, outbounds: [{ type: "vless", tag: "node" }] } as Subscription])).toBe(true)
  })

  it("uses server module and proxy status and detects unavailable TUN", () => {
    const setup = setupStatus({ listeners: [{ type: "tun", tag: "tun", listen: "" }] })
    expect(buildSetupSteps({ setup }).map((step) => step.done)).toEqual([false, false, false, false])
    setup.modules = setup.modules.map((item) => ({ ...item, state: "ready" }))
    setup.capabilities.tun_available = true
    setup.proxy_ready = true
    setup.kernel_running = true
    expect(setupProgress(buildSetupSteps({ setup }))).toEqual({ done: 4, total: 4, complete: true })
  })

  it("does not confuse unrelated listeners with client access", () => {
    const setup = setupStatus({ listeners: [{ type: "vless", tag: "server", listen: "0.0.0.0", listen_port: 443 }] })
    expect(buildSetupSteps({ setup }).find((step) => step.id === "access")?.done).toBe(false)
    expect(hasLocalInbound({ inbounds: [null, 1, { type: "vless" }] } as SingBoxConfig)).toBe(false)
    expect(hasProxyOutbound()).toBe(false)
    expect(hasRouteRules()).toBe(false)
  })

  it("does not complete base modules while configuration has a semantic error", () => {
    const setup = setupStatus({ config_error: "ntp.server: invalid server", kernel_running: true, proxy_ready: true,
      listeners: [{ type: "mixed", tag: "mixed", listen: "127.0.0.1", listen_port: 1080 }] })
    setup.modules = setup.modules.map((module) => ({ ...module, state: "ready" }))
    const steps = buildSetupSteps({ setup })
    expect(steps.find((step) => step.id === "modules")?.done).toBe(false)
    expect(setupProgress(steps).complete).toBe(false)
  })

  it("does not complete access for invalid IPv6 TUN despite a running kernel", () => {
    const setup = setupStatus({ kernel_running: true, proxy_ready: true,
      listeners: [{ type: "tun", tag: "tun", listen: "", address: ["172.19.0.1/30", "fdfe:dcba:9876::1/126"] }] })
    setup.capabilities.tun_available = true
    setup.capabilities.ipv6_reason = "ipv6_unsupported"
    setup.modules = setup.modules.map((module) => ({ ...module, state: module.id === "inbounds" ? "invalid" : "ready" }))
    const steps = buildSetupSteps({ setup })
    expect(steps.find((step) => step.id === "access")?.done).toBe(false)
    expect(setupProgress(steps).complete).toBe(false)
  })
})
