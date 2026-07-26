import { afterEach, describe, expect, it } from "vitest"

import {
  CONNECTION_COLUMN_STORAGE_KEY,
  connectionColumnVisible,
  defaultConnectionColumns,
  loadConnectionColumns,
  normalizeConnectionColumns,
  saveConnectionColumns,
  toggleConnectionColumn,
} from "@/features/observability/connection-columns"

afterEach(() => {
  localStorage.removeItem(CONNECTION_COLUMN_STORAGE_KEY)
})

describe("connection-columns", () => {
  it("defaults to core columns and keeps required ones", () => {
    const defaults = defaultConnectionColumns()
    expect(defaults).toContain("target")
    expect(defaults).toContain("outbound")
    expect(defaults).toContain("actions")
    expect(defaults).toContain("process")
    expect(defaults).toContain("rate")
    expect(normalizeConnectionColumns(["process"])).toEqual(
      expect.arrayContaining(["target", "outbound", "actions", "process"]),
    )
  })

  it("loads and saves visibility in localStorage", () => {
    expect(loadConnectionColumns()).toEqual(defaultConnectionColumns())
    const saved = saveConnectionColumns(["target", "outbound", "process", "actions"])
    expect(saved).toContain("process")
    expect(loadConnectionColumns()).toEqual(saved)
  })

  it("ignores corrupt storage and keeps required columns when toggling", () => {
    localStorage.setItem(CONNECTION_COLUMN_STORAGE_KEY, "{not-json")
    expect(loadConnectionColumns()).toEqual(defaultConnectionColumns())
    const current = defaultConnectionColumns()
    expect(toggleConnectionColumn(current, "target", false)).toEqual(normalizeConnectionColumns(current))
    const withSource = toggleConnectionColumn(current, "source", true)
    expect(connectionColumnVisible(withSource, "source")).toBe(true)
    expect(connectionColumnVisible(toggleConnectionColumn(withSource, "source", false), "source")).toBe(false)
  })
})
