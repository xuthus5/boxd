import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { toast } from "sonner"

import { SupportBundleCard } from "@/features/settings/support-bundle-card"
import * as client from "@/lib/api/client"
import { i18n } from "@/i18n"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const preferences = { theme: "system" as const, language: "zh" as const, minimumLogLevel: "all" as const }

const payloads: Record<string, unknown> = {
  "/api/runtime/version": { version: "dev", kernel_version: "1.13.14" },
  "/api/service/status": { running: true, uptime: "1h" },
  "/readyz": { status: "ready" },
  "/api/runtime/memory": { alloc: 10, total: 20, sys: 30, num_gc: 1, heap_inuse: 2, stack_inuse: 3 },
  "/api/config/diagnostics": {
    status: "healthy", checked_at: "now", summary: { errors: 0, warnings: 0 },
    counts: { inbounds: 1, outbounds: 1, endpoints: 0, route_rules: 0, rule_sets: 0, dns_servers: 0, dns_rules: 0 },
    features: { tun: false, clash_api: false, cache_file: false, fakeip: false, selector: false, urltest: false, wireguard: false, remote_rule_set: false }, issues: [],
  },
  "/api/config/rule-sets/status": [],
  "/api/config/rule-sets/auto-update": { enabled: false, interval: "24h" },
  "/api/subscriptions/": [],
  "/api/nodes/": [],
  "/api/nodes/test-history": { history: {} },
  "/api/config/apply-history": { events: [] },
  "/api/network/interfaces": { interfaces: [] },
}

function pathOf(input: string | URL | Request): string {
  if (typeof input === "string") return input
  if (input instanceof URL) return input.pathname
  return new URL(input.url).pathname
}

function renderCard() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={queryClient}>
        <SupportBundleCard preferences={preferences} />
      </QueryClientProvider>
    </I18nextProvider>,
  )
}

function installFetch(failingPath?: string) {
  vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
    const path = pathOf(input)
    if (path === failingPath) {
      return Promise.resolve(new Response(JSON.stringify({ status: "error", data: null, error: { code: "unavailable", message: "not ready" } }), { status: 503 }))
    }
    return Promise.resolve(new Response(JSON.stringify(payloads[path] ?? {}), { status: 200 }))
  }))
}

function installPendingFetch() {
  const releases: Array<() => void> = []
  vi.stubGlobal("fetch", vi.fn((input: string | URL | Request) => {
    const path = pathOf(input)
    return new Promise<Response>((resolve) => {
      releases.push(() => resolve(new Response(JSON.stringify(payloads[path] ?? {}), { status: 200 })))
    })
  }))
  return () => releases.splice(0).forEach((release) => release())
}

beforeEach(async () => {
  await i18n.changeLanguage("zh")
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  vi.clearAllMocks()
})

describe("SupportBundleCard", () => {
  it("downloads a redacted JSON bundle", async () => {
    installFetch()
    const trigger = vi.spyOn(client, "triggerBrowserDownload").mockImplementation(() => undefined)

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "生成支持包" }))

    await waitFor(() => expect(trigger).toHaveBeenCalledTimes(1))
    const [blob, filename] = trigger.mock.calls[0]
    expect(filename).toMatch(/^boxd-support-bundle-.*\.json$/)
    const content = await (blob as Blob).text()
    expect(JSON.parse(content)).toMatchObject({ product: "boxd", redaction: { strategy: "allowlist" } })
    expect(toast.success).toHaveBeenCalledWith(expect.stringContaining("支持包已下载"))
  })

  it("reports partial collection without blocking the download", async () => {
    installFetch("/api/nodes/")
    const trigger = vi.spyOn(client, "triggerBrowserDownload").mockImplementation(() => undefined)

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "生成支持包" }))

    await waitFor(() => expect(trigger).toHaveBeenCalledTimes(1))
    expect(toast.success).toHaveBeenCalledWith(expect.stringMatching(/1.*暂不可用|unavailable/i))
  })

  it("shows collection progress while sources are pending", async () => {
    const release = installPendingFetch()
    const trigger = vi.spyOn(client, "triggerBrowserDownload").mockImplementation(() => undefined)

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "生成支持包" }))
    await waitFor(() => expect(screen.getByRole("button", { name: "正在收集…" })).toBeDisabled())
    release()
    await waitFor(() => expect(trigger).toHaveBeenCalledTimes(1))
  })

  it("surfaces browser download failures", async () => {
    installFetch()
    vi.spyOn(client, "triggerBrowserDownload").mockImplementation(() => { throw new Error("download blocked") })

    renderCard()
    await userEvent.setup().click(screen.getByRole("button", { name: "生成支持包" }))

    await waitFor(() => expect(toast.error).toHaveBeenCalled())
    expect(String(toast.error.mock.calls.at(-1)?.[0])).toMatch(/download blocked|生成诊断支持包失败/i)
  })
})
