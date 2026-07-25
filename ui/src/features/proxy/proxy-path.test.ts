import { describe, expect, it } from "vitest"

import { parseProxyItemPath, proxyItemRelativePaths } from "@/features/proxy/proxy-path"

describe("parseProxyItemPath", () => {
  it("parses full section paths", () => {
    expect(parseProxyItemPath("inbounds[0].listen_port", "inbounds")).toEqual({
      section: "inbounds",
      index: 0,
      relativePath: "listen_port",
    })
    expect(parseProxyItemPath("outbounds[2].tls.server_name", "outbounds")).toEqual({
      section: "outbounds",
      index: 2,
      relativePath: "tls.server_name",
    })
  })

  it("parses bare index paths and empty relative", () => {
    expect(parseProxyItemPath("[1].server", "outbounds")).toEqual({
      section: "outbounds",
      index: 1,
      relativePath: "server",
    })
    expect(parseProxyItemPath("inbounds[3]", "inbounds")).toEqual({
      section: "inbounds",
      index: 3,
      relativePath: "",
    })
    expect(parseProxyItemPath("  [4]  ", "inbounds")).toEqual({
      section: "inbounds",
      index: 4,
      relativePath: "",
    })
  })

  it("rejects mismatched or incomplete paths", () => {
    expect(parseProxyItemPath("outbounds[0].tag", "inbounds")).toBeNull()
    expect(parseProxyItemPath("inbounds", "inbounds")).toBeNull()
    expect(parseProxyItemPath("", "inbounds")).toBeNull()
  })
})

describe("proxyItemRelativePaths", () => {
  it("maps full paths for the open item index", () => {
    expect(proxyItemRelativePaths("inbounds[0].listen_port", "inbounds", 0)).toEqual(["listen_port"])
    expect(proxyItemRelativePaths("inbounds[1].listen_port", "inbounds", 0)).toEqual([])
  })

  it("accepts any index when drafting a new item", () => {
    expect(proxyItemRelativePaths("outbounds[3].server", "outbounds", -1)).toEqual(["server"])
  })

  it("accepts bare relative paths and section-dot prefixes", () => {
    expect(proxyItemRelativePaths("tls.server_name", "outbounds", 0)).toEqual(["tls.server_name"])
    expect(proxyItemRelativePaths("inbounds.listen_port", "inbounds", 0)).toEqual(["listen_port"])
    expect(proxyItemRelativePaths("inbounds.", "inbounds", 0)).toEqual([])
    expect(proxyItemRelativePaths("inbounds[0]", "inbounds", 0)).toEqual([])
    expect(proxyItemRelativePaths("", "inbounds", 0)).toEqual([])
    expect(proxyItemRelativePaths("route.final", "inbounds", 0)).toEqual([])
  })
})
