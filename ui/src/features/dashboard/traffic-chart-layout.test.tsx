import { renderHook } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const recharts = vi.hoisted(() => ({
  plotArea: undefined as { x: number; y: number; width: number; height: number } | undefined,
}))

vi.mock("recharts", () => ({ usePlotArea: () => recharts.plotArea }))

import {
  useStableTrafficPlotArea,
  useTrafficTimeOrigin,
} from "@/features/dashboard/traffic-chart-layout"
import type { TrafficChartPoint } from "@/features/dashboard/traffic-chart-model"

describe("traffic chart layout", () => {
  beforeEach(() => {
    recharts.plotArea = undefined
  })

  it("retains the last usable plot area", () => {
    recharts.plotArea = { x: 10, y: 20, width: 300, height: 160 }
    const hook = renderHook(() => useStableTrafficPlotArea())
    expect(hook.result.current).toEqual(recharts.plotArea)

    recharts.plotArea = { x: 10, y: 20, width: 300, height: 160 }
    hook.rerender()
    expect(hook.result.current).toEqual(recharts.plotArea)

    recharts.plotArea = { x: 12, y: 24, width: 640, height: 220 }
    hook.rerender()
    expect(hook.result.current).toEqual(recharts.plotArea)

    recharts.plotArea = undefined
    hook.rerender()
    expect(hook.result.current).toEqual({ x: 12, y: 24, width: 640, height: 220 })
  })

  it("anchors a late traffic stream to its first hour bucket", () => {
    const hook = renderHook(
      ({ data }: { data: TrafficChartPoint[] }) => useTrafficTimeOrigin(data),
      { initialProps: { data: [] } },
    )
    expect(hook.result.current).toBe(0)

    hook.rerender({ data: [{ timestamp: "2026-01-01T00:59:59Z" }] })
    expect(hook.result.current).toBe(Date.parse("2026-01-01T00:00:00Z"))

    hook.rerender({ data: [{ timestamp: "2026-01-01T01:00:01Z" }] })
    expect(hook.result.current).toBe(Date.parse("2026-01-01T00:00:00Z"))
  })
})
