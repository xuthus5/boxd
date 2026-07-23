import { describe, expect, it } from "vitest"

import { indexToLineColumn, locateJsonPath, parseJsonPath } from "@/features/config/json-path"

describe("json-path", () => {
  const sample = `{
  "log": {
    "level": "info"
  },
  "inbounds": [
    {
      "tag": "mixed-in",
      "listen_port": 1080
    }
  ]
}`

  it("parses dotted and bracket paths", () => {
    expect(parseJsonPath("log.level")).toEqual(["log", "level"])
    expect(parseJsonPath("inbounds[0].tag")).toEqual(["inbounds", 0, "tag"])
    expect(parseJsonPath("$.dns.final")).toEqual(["dns", "final"])
  })

  it("locates nested keys and reports line/column", () => {
    const location = locateJsonPath(sample, "log.level")
    expect(location).not.toBeNull()
    expect(location?.line).toBe(3)
    expect(sample.slice(location!.index, location!.index + 7)).toBe('"level"')
    const port = locateJsonPath(sample, "inbounds[0].listen_port")
    expect(port).not.toBeNull()
    expect(sample.includes("listen_port")).toBe(true)
    expect(indexToLineColumn(sample, 0)).toEqual({ line: 1, column: 1 })
  })

  it("returns null for missing paths", () => {
    expect(locateJsonPath(sample, "missing.path")).toBeNull()
  })
})
