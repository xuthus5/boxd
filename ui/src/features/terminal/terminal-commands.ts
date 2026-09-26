import type { SetupListener, SetupStatus } from "@/lib/api/setup"

/** Proxy inbound types that expose a local HTTP/SOCKS endpoint. */
const PROXY_TYPES = ["mixed", "http", "socks"] as const

export interface ProxyEndpoint {
  /** Host as it should appear in the copied command (already bracketed for IPv6). */
  host: string
  port: number
  /** URL scheme for both http_proxy and https_proxy. */
  scheme: "http" | "socks5h"
  /** Full proxy URL, e.g. http://127.0.0.1:1080 */
  url: string
}

export interface TerminalCommand {
  id: "bash" | "fish" | "powershell" | "cmd"
  labelKey: string
  command: string
}

function isValidPort(port: number | undefined): port is number {
  return Number.isInteger(port) && (port as number) >= 1 && (port as number) <= 65535
}

function formatHost(host: string): string {
  return host.includes(":") ? `[${host}]` : host
}

/** Wildcard and loopback hostnames resolve to a client-usable loopback address. */
function endpointHost(listen: string): string {
  if (!listen || listen === "0.0.0.0" || listen === "::" || listen === "[::]" || listen === "localhost") {
    return "127.0.0.1"
  }
  return listen
}

/** Pick the first usable mixed/HTTP/SOCKS listener and turn it into copy-ready proxy URL parts. */
export function resolveProxyEndpoint(status: SetupStatus | undefined): ProxyEndpoint | null {
  const listeners = status?.listeners ?? []
  let match: SetupListener | undefined
  for (const type of PROXY_TYPES) {
    const candidate = listeners.find((listener) => listener.type === type && isValidPort(listener.listen_port))
    if (candidate) {
      match = candidate
      break
    }
  }
  if (!match) return null
  const host = formatHost(endpointHost(match.listen))
  const port = match.listen_port as number
  const scheme = match.type === "socks" ? "socks5h" : "http"
  return { host, port, scheme, url: `${scheme}://${host}:${port}` }
}

/** Ready-to-copy shell snippets that export http_proxy/https_proxy for each platform. */
export function buildTerminalCommands(endpoint: ProxyEndpoint): TerminalCommand[] {
  const { url } = endpoint
  return [
    {
      id: "bash",
      labelKey: "terminal.platformBash",
      command: `export http_proxy="${url}" https_proxy="${url}" HTTP_PROXY="${url}" HTTPS_PROXY="${url}"`,
    },
    {
      id: "fish",
      labelKey: "terminal.platformFish",
      command: `set -x http_proxy "${url}"; set -x https_proxy "${url}"; set -x HTTP_PROXY "${url}"; set -x HTTPS_PROXY "${url}"`,
    },
    {
      id: "powershell",
      labelKey: "terminal.platformPowerShell",
      command: `$env:HTTP_PROXY="${url}"; $env:HTTPS_PROXY="${url}"`,
    },
    {
      id: "cmd",
      labelKey: "terminal.platformCmd",
      command: `set "http_proxy=${url}" && set "https_proxy=${url}"`,
    },
  ]
}
