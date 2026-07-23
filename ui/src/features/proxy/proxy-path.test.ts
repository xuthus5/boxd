import { describe, expect, it } from "vitest"

import { parseProxyItemPath } from "@/features/proxy/proxy-path"

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
  })

  it("rejects mismatched or incomplete paths", () => {
    expect(parseProxyItemPath("outbounds[0].tag", "inbounds")).toBeNull()
    expect(parseProxyItemPath("inbounds", "inbounds")).toBeNull()
    expect(parseProxyItemPath("", "inbounds")).toBeNull()
  })
})
