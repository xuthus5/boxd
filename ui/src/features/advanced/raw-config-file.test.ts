import { describe, expect, it } from "vitest"

import {
  formatRawConfig,
  MAX_RAW_CONFIG_FILE_BYTES,
  parseRawConfigText,
  RawConfigFileError,
  rawConfigFileErrorMessageKey,
  readRawConfigFile,
} from "@/features/advanced/raw-config-file"

describe("raw config file helpers", () => {
  it("parses object configs and formats them deterministically", async () => {
    const config = parseRawConfigText('{"log":{"level":"info"}}')
    expect(config).toEqual({ log: { level: "info" } })
    expect(formatRawConfig(config)).toBe('{\n  "log": {\n    "level": "info"\n  }\n}\n')

    const file = { size: 25, text: async () => '{"log":{"level":"info"}}' }
    await expect(readRawConfigFile(file)).resolves.toEqual(config)
  })

  it.each([
    ["{", "invalid_json"],
    ["[]", "invalid_root"],
    ["null", "invalid_root"],
  ] as const)("rejects %s as %s", (text, code) => {
    expect(() => parseRawConfigText(text)).toThrowError(
      expect.objectContaining({ code }),
    )
  })

  it("rejects oversized files before reading their contents", async () => {
    let readCount = 0
    const file = {
      size: MAX_RAW_CONFIG_FILE_BYTES + 1,
      text: async () => {
        readCount += 1
        return "{}"
      },
    }
    await expect(readRawConfigFile(file)).rejects.toMatchObject({ code: "too_large" })
    expect(readCount).toBe(0)
  })

  it("normalizes file read failures and exposes translation keys", async () => {
    const file = { size: 1, text: async () => { throw new Error("offline") } }
    await expect(readRawConfigFile(file)).rejects.toBeInstanceOf(RawConfigFileError)
    expect(rawConfigFileErrorMessageKey("read_failed")).toBe("advanced.importFailed")
    expect(rawConfigFileErrorMessageKey("too_large")).toBe("advanced.importTooLarge")
    expect(rawConfigFileErrorMessageKey("invalid_json")).toBe("advanced.importInvalidJSON")
    expect(rawConfigFileErrorMessageKey("invalid_root")).toBe("advanced.importInvalidRoot")
  })
})
