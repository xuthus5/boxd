import type { SetupListener } from "@/lib/api/setup"

export function isLoopbackListener(listener: SetupListener) {
  return listener.listen === "::1" || listener.listen === "localhost" || listener.listen.startsWith("127.")
}

export function listenerCommands(listener: SetupListener, container: boolean) {
  const port = listener.listen_port
  if (!["mixed", "http", "socks"].includes(listener.type) || !Number.isInteger(port) || !port || port < 1 || port > 65535) return null
  if (container && isLoopbackListener(listener)) return null
  const address = !container && /^[0-9a-fA-F:.]+$/.test(listener.listen) && !["0.0.0.0", "::"].includes(listener.listen)
    ? listener.listen : "127.0.0.1"
  const host = address.includes(":") ? `[${address}]` : address
  const scheme = listener.type === "http" ? "http" : "socks5h"
  return {
    publication: container ? `-p 127.0.0.1:${port}:${port}` : "",
    check: `curl --proxy ${scheme}://${host}:${port} https://example.com`,
  }
}
