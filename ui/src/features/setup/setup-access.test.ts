import { describe, expect, it } from "vitest"

import { listenerCommands } from "@/features/setup/setup-access"

const mixed = { type: "mixed", tag: "mixed-in", listen: "0.0.0.0", listen_port: 1080 }

describe("client connection commands", () => {
  it("publishes only on host loopback and sends proxy DNS through SOCKS", () => {
    expect(listenerCommands(mixed, true)).toEqual({ publication: "-p 127.0.0.1:1080:1080",
      check: "curl --proxy socks5h://127.0.0.1:1080 https://example.com" })
  })
  it.each(["127.0.0.1", "127.0.0.2", "::1", "localhost"])("does not offer a broken mapping to container loopback %s", (listen) => {
    expect(listenerCommands({ ...mixed, listen }, true)).toBeNull()
  })
  it("uses explicit HTTP and IPv6 listeners for local commands", () => {
    expect(listenerCommands({ ...mixed, type: "http", listen: "::1" }, false)?.check).toBe("curl --proxy http://[::1]:1080 https://example.com")
    expect(listenerCommands({ ...mixed, listen: "192.168.1.2" }, false)?.publication).toBe("")
    expect(listenerCommands({ ...mixed, listen: "192.168.1.2" }, false)?.check).toContain("192.168.1.2:1080")
  })
  it.each([0, -1, 65536, 1.5, undefined])("rejects invalid ports %s", (listen_port) => {
    expect(listenerCommands({ ...mixed, listen_port }, true)).toBeNull()
  })
  it("does not interpolate configuration text into shell commands", () => {
    expect(listenerCommands({ ...mixed, listen: "$(touch /tmp/injected)" }, false)?.check).toContain("127.0.0.1")
    expect(listenerCommands({ ...mixed, type: "tun" }, false)).toBeNull()
  })
})
