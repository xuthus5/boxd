import type { ReactElement } from "react"
import { I18nextProvider } from "react-i18next"
import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { SetupAccessGuide } from "@/features/setup/setup-access-guide"
import { SetupCapabilities } from "@/features/setup/setup-capabilities"
import { SetupModules } from "@/features/setup/setup-modules"
import { SetupPreview } from "@/features/setup/setup-preview"
import { i18n } from "@/i18n"
import { renderApp } from "@/test/render"
import { setupStatus } from "@/test/setup-fixtures"

function renderLanguage(element: ReactElement, language: string) {
  return renderApp(<I18nextProvider i18n={i18n.cloneInstance({ lng: language })}>{element}</I18nextProvider>)
}

describe.each(["zh", "en"])("setup messages in %s", (language) => {
  it("shows unknown capabilities as unconfirmed", () => {
    const capabilities = { ...setupStatus().capabilities, tun_reason: "tun_unknown", ipv6_reason: "ipv6_unknown" }
    renderLanguage(<SetupCapabilities capabilities={capabilities} />, language)
    expect(screen.getByText(language === "zh" ? "TUN 未确认" : "TUN unconfirmed")).toBeInTheDocument()
    expect(screen.getByText(language === "zh" ? "IPv6 未确认" : "IPv6 unconfirmed")).toBeInTheDocument()
    expect(screen.queryByText(language === "zh" ? "TUN 不可用" : "TUN unavailable")).not.toBeInTheDocument()
    expect(screen.queryByText("ipv6_unknown")).not.toBeInTheDocument()
  })

  it.each([
    ["tun_permission_required", "CAP_NET_ADMIN"],
    ["tun_device_unavailable", "--device /dev/net/tun"],
    ["tun_unsupported", "TUN"],
    ["ipv6_disabled", "IPv6"],
    ["ipv6_unsupported", "IPv6"],
  ])("explains capability reason %s", (reason, detail) => {
    const capabilities = { ...setupStatus().capabilities,
      tun_reason: reason.startsWith("tun") ? reason : undefined,
      ipv6_reason: reason.startsWith("ipv6") ? reason : undefined }
    const view = renderLanguage(<SetupCapabilities capabilities={capabilities} />, language)
    expect(view.container).toHaveTextContent(detail)
    expect(screen.queryByText(reason)).not.toBeInTheDocument()
  })

  it("translates all stable preview warnings", () => {
    const warnings = ["container_port_mapping", "proxy_nodes_required", "tun_ipv4_only", "config_reset"]
    const view = renderLanguage(<SetupPreview preview={{ source_hash: "hash", current_config: {}, config: {},
      modules: [], warnings, will_restart: false }} />, language)
    for (const warning of warnings) expect(screen.queryByText(warning)).not.toBeInTheDocument()
    expect(view.container).toHaveTextContent(language === "zh" ? "宿主代理端口" : "host proxy port")
    expect(view.container).toHaveTextContent(language === "zh" ? "导入节点或订阅" : "Import nodes or a subscription")
    expect(view.container).toHaveTextContent("IPv4")
    expect(view.container).toHaveTextContent(language === "zh" ? "保留原文件" : "copy of the original file")
  })

  it("translates module messages while retaining safe diagnostic text", () => {
    const modules = setupStatus().modules.slice(0, 2)
    modules[0]!.message = "ipv6_disabled"
    modules[1]!.message = 'route.final: <img src=x onerror="alert(1)">'
    const view = renderLanguage(<SetupModules modules={modules} selected={[]} disabled={false} onChange={vi.fn()} />, language)
    expect(screen.queryByText("ipv6_disabled")).not.toBeInTheDocument()
    expect(screen.getByText(modules[1]!.message!)).toBeInTheDocument()
    expect(view.container.querySelector("img")).toBeNull()
  })

  it("states that loopback verification runs on the service host", () => {
    const status = setupStatus({ listeners: [{ type: "mixed", tag: "proxy", listen: "127.0.0.1", listen_port: 1080 }] })
    status.capabilities.container = false
    const view = renderLanguage(<SetupAccessGuide status={status} />, language)
    expect(view.container).toHaveTextContent(language === "zh" ? "在运行 boxd 的主机执行" : "Run on the boxd host")
    expect(view.container).toHaveTextContent(language === "zh" ? "远端浏览器需要自行配置代理" : "A remote browser needs its own proxy settings")
  })
})

it("keeps unknown capability and warning text without interpreting HTML or object properties", () => {
  const capabilities = { ...setupStatus().capabilities, tun_reason: "vendor-specific check failed" }
  const view = renderLanguage(<><SetupCapabilities capabilities={capabilities} />
    <SetupPreview preview={{ source_hash: "hash", current_config: {}, config: {}, modules: [],
      warnings: ["toString", "future_warning", "<img src=x>"], will_restart: false }} /></>, "zh")
  for (const message of ["vendor-specific check failed", "toString", "future_warning", "<img src=x>"]) {
    expect(screen.getByText(message)).toBeInTheDocument()
  }
  expect(view.container.querySelector("img")).toBeNull()
})
