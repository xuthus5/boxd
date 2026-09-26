import { describe, expect, it } from "vitest"

import { buildTerminalCommands, resolveProxyEndpoint } from "@/features/terminal/terminal-commands"
import { setupStatus } from "@/test/setup-fixtures"

describe("resolveProxyEndpoint", () => {
  it("returns null when there is no usable proxy listener", () => {
    expect(resolveProxyEndpoint(undefined)).toBeNull()
    expect(resolveProxyEndpoint(setupStatus({ listeners: [{ tag: "tun-in", type: "tun", listen: "" }] }))).toBeNull()
  })

  it("uses the mixed inbound and rewrites wildcard listens to loopback", () => {
    const endpoint = resolveProxyEndpoint(setupStatus({
      listeners: [{ tag: "mixed-in", type: "mixed", listen: "::", listen_port: 1080 }],
    }))
    expect(endpoint).toEqual({ host: "127.0.0.1", port: 1080, scheme: "http", url: "http://127.0.0.1:1080" })
  })

  it("keeps explicit IPv4/IPv6 listens and switches the scheme for socks", () => {
    const ipv4 = resolveProxyEndpoint(setupStatus({
      listeners: [{ tag: "http-in", type: "http", listen: "192.168.1.10", listen_port: 2080 }],
    }))
    expect(ipv4?.url).toBe("http://192.168.1.10:2080")

    const socks = resolveProxyEndpoint(setupStatus({
      listeners: [{ tag: "socks-in", type: "socks", listen: "::1", listen_port: 1081 }],
    }))
    expect(socks?.url).toBe("socks5h://[::1]:1081")
  })

  it("skips listeners with an invalid port and falls back to the next type", () => {
    const endpoint = resolveProxyEndpoint(setupStatus({
      listeners: [
        { tag: "mixed-in", type: "mixed", listen: "127.0.0.1", listen_port: 0 },
        { tag: "http-in", type: "http", listen: "127.0.0.1", listen_port: 8080 },
      ],
    }))
    expect(endpoint?.url).toBe("http://127.0.0.1:8080")
  })
})

describe("buildTerminalCommands", () => {
  const endpoint = { host: "127.0.0.1", port: 1080, scheme: "http" as const, url: "http://127.0.0.1:1080" }

  it("emits one command per platform that exports both proxy variables", () => {
    const commands = buildTerminalCommands(endpoint)
    expect(commands.map((item) => item.id)).toEqual(["bash", "fish", "powershell", "cmd"])
    for (const item of commands) {
      expect(item.command).toContain("http://127.0.0.1:1080")
    }
    const byId = Object.fromEntries(commands.map((item) => [item.id, item.command]))
    expect(byId.bash).toContain("http_proxy=")
    expect(byId.bash).toContain("https_proxy=")
    expect(byId.fish).toContain("set -x http_proxy")
    expect(byId.powershell).toContain("$env:HTTP_PROXY=")
    expect(byId.powershell).toContain("$env:HTTPS_PROXY=")
    expect(byId.cmd).toContain('set "http_proxy=')
    expect(byId.cmd).toContain('set "https_proxy=')
  })
})
