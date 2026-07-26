import { useState, type ReactNode } from "react"
import { act, render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const recharts = vi.hoisted(() => ({
  line: vi.fn(),
  chart: vi.fn(),
  curve: vi.fn(),
  grid: vi.fn(),
  plotArea: undefined as { x: number; y: number; width: number; height: number } | undefined,
  xAxis: vi.fn(),
  yAxis: vi.fn(),
}))

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  LineChart: ({ children, ...props }: { children: ReactNode }) => {
    recharts.chart(props)
    return <svg data-testid="traffic-chart-instance">{children}</svg>
  },
  Line: (props: Record<string, unknown>) => {
    recharts.line(props)
    return <g data-testid="traffic-line" />
  },
  Curve: (props: Record<string, unknown>) => {
    recharts.curve(props)
    const series = props["data-series"]
    return <path data-testid={typeof series === "string" ? `traffic-curve-${series}` : "traffic-curve"} />
  },
  usePlotArea: () => recharts.plotArea,
  useXAxisScale: () => (value: unknown) => Number(value),
  useYAxisScale: () => (value: unknown) => Number(value),
  CartesianGrid: (props: Record<string, unknown>) => {
    recharts.grid(props)
    return null
  },
  XAxis: (props: Record<string, unknown>) => {
    recharts.xAxis(props)
    return null
  },
  YAxis: (props: Record<string, unknown>) => {
    recharts.yAxis(props)
    return null
  },
  Tooltip: () => null,
  Legend: () => null,
}))

import { TrafficChart } from "@/features/dashboard/traffic-chart"
import { SmoothTrafficCurve } from "@/features/dashboard/traffic-chart-plot"
import type { TrafficHistoryPoint } from "@/lib/api/types"
import { renderApp } from "@/test/render"

const initialPoints: TrafficHistoryPoint[] = [
  { timestamp: "2026-01-01T00:00:00Z", upload_bytes: 0, download_bytes: 0 },
  { timestamp: "2026-01-01T00:00:01Z", upload_bytes: 1024, download_bytes: 2048 },
]

function Harness() {
  const [unrelated, setUnrelated] = useState(0)
  const [points, setPoints] = useState(initialPoints)
  return <>
    <button onClick={() => setUnrelated((value) => value + 1)}>rerender parent</button>
    <button onClick={() => setPoints((current) => [...current, {
      timestamp: "2026-01-01T00:00:02Z", upload_bytes: 2048, download_bytes: 4096,
    }])}>append sample</button>
    <output aria-label="unrelated state">{unrelated}</output>
    <TrafficChart points={points} />
  </>
}

describe("TrafficChart stability", () => {
  beforeEach(() => {
    recharts.line.mockClear()
    recharts.chart.mockClear()
    recharts.curve.mockClear()
    recharts.grid.mockClear()
    recharts.plotArea = { x: 72, y: 4, width: 720, height: 220 }
    recharts.xAxis.mockClear()
    recharts.yAxis.mockClear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("keeps the chart mounted without replaying line animations", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)

    const initialLineCount = recharts.line.mock.calls.length
    expect(initialLineCount).toBeGreaterThan(0)
    const chartInstance = screen.getByTestId("traffic-chart-instance")
    const uploadCurve = screen.getByTestId("traffic-curve-upload_rate")
    const stableTickValue = Date.parse("2026-01-01T00:00:00Z")
    const stableTick = document.querySelector(`[data-traffic-time-tick="${stableTickValue}"]`)
    expect(stableTick).not.toBeNull()
    expect(stableTick).not.toHaveClass("transition-transform")
    const initialTickTransform = (stableTick as SVGGElement).style.transform
    expect(recharts.xAxis).toHaveBeenCalled()
    const initialXAxis = recharts.xAxis.mock.calls.at(-1)?.[0] as Record<string, unknown>
    expect(initialXAxis).toMatchObject({
      type: "number",
      allowDataOverflow: true,
      domain: [expect.any(Number), expect.any(Number)],
      tick: false,
    })
    expect(recharts.yAxis.mock.calls.at(-1)?.[0]).toMatchObject({ tick: false })
    expect(recharts.grid).not.toHaveBeenCalled()
    const timestampValue = initialXAxis.dataKey as (point: { timestamp: string }) => number
    const tickFormatter = initialXAxis.tickFormatter as (value: unknown) => string
    expect(timestampValue({ timestamp: "2026-01-01T00:00:01Z" })).toBe(Date.parse("2026-01-01T00:00:01Z"))
    expect(timestampValue({ timestamp: "invalid" })).toBe(0)
    expect(timestampValue({ timestamp: 1 as unknown as string })).toBe(0)
    expect(tickFormatter(Date.parse("2026-01-01T00:00:01Z"))).not.toBe("")
    expect(tickFormatter(Date.parse("2026-01-01T00:00:01Z"))).not.toMatch(/AM|PM/)
    expect(tickFormatter("invalid")).toBe("")
    for (const [props] of recharts.line.mock.calls) {
      expect(props).toMatchObject({
        activeDot: false,
        opacity: 0,
        isAnimationActive: false,
        animateNewValues: false,
      })
      expect(props).not.toHaveProperty("shape")
    }

    await user.click(screen.getByRole("button", { name: "rerender parent" }))
    expect(screen.getByLabelText("unrelated state")).toHaveTextContent("1")
    expect(recharts.line).toHaveBeenCalledTimes(initialLineCount)

    await user.click(screen.getByRole("button", { name: "append sample" }))
    expect(recharts.line.mock.calls.length).toBeGreaterThan(initialLineCount)
    expect(screen.getByTestId("traffic-chart-instance")).toBe(chartInstance)
    expect(recharts.line.mock.calls.at(-1)?.[0]).toMatchObject({
      activeDot: false,
      opacity: 0,
      isAnimationActive: false,
      animateNewValues: false,
    })
    expect(screen.getByTestId("traffic-curve-upload_rate")).toBe(uploadCurve)
    const updatedTick = document.querySelector(`[data-traffic-time-tick="${stableTickValue}"]`)
    expect(updatedTick).toBe(stableTick)
    expect(updatedTick).toHaveClass("transition-transform")
    expect((updatedTick as SVGGElement).style.transform).not.toBe(initialTickTransform)
  })

  it("keeps the last curve while chart layout is temporarily unavailable", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)
    const uploadCurve = screen.getByTestId("traffic-curve-upload_rate")

    recharts.plotArea = undefined
    await user.click(screen.getByRole("button", { name: "append sample" }))

    expect(screen.getByTestId("traffic-curve-upload_rate")).toBe(uploadCurve)
    const updated = recharts.curve.mock.calls.at(-2)?.[0] as { points?: unknown[] }
    expect(updated.points?.length).toBeGreaterThan(0)
  })

  it("reduces time tick density for narrow traffic cards", () => {
    recharts.plotArea = { x: 72, y: 4, width: 184, height: 220 }
    renderApp(<TrafficChart points={initialPoints} />)

    expect(document.querySelectorAll("[data-traffic-time-tick]")).toHaveLength(2)
  })

  it("does not render synthetic epoch ticks without traffic samples", () => {
    renderApp(<TrafficChart points={[]} />)

    expect(document.querySelectorAll("[data-traffic-time-tick], [data-traffic-value-tick]")).toHaveLength(0)
  })

  it("interpolates matching samples without replaying the full curve", () => {
    const frames: FrameRequestCallback[] = []
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      frames.push(callback)
      return frames.length
    })
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined)
    vi.spyOn(performance, "now").mockReturnValue(0)

    const initial = [
      { x: 0, y: 20, payload: { timestamp: "2026-01-01T00:00:00Z" } },
      { x: 100, y: 10, payload: { timestamp: "2026-01-01T00:00:01Z" } },
    ]
    const next = [
      { x: 0, y: 10, payload: { timestamp: "2026-01-01T00:00:01Z" } },
      { x: 100, y: 5, payload: { timestamp: "2026-01-01T00:00:02Z" } },
    ]
    const view = render(<svg><SmoothTrafficCurve points={initial} transitionDuration={800} /></svg>)
    const curve = screen.getByTestId("traffic-curve")

    view.rerender(<svg><SmoothTrafficCurve points={next} transitionDuration={800} /></svg>)
    expect(frames).toHaveLength(1)

    act(() => frames.shift()?.(400))
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({
      points: [
        expect.objectContaining({ x: 50, y: 10 }),
        expect.objectContaining({ x: 100, y: 7.5 }),
      ],
    })
    expect(screen.getByTestId("traffic-curve")).toBe(curve)

    act(() => frames.shift()?.(800))
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({ points: next })
    expect(screen.getByTestId("traffic-curve")).toBe(curve)
  })

  it("commits discontinuous and reduced-motion updates immediately", () => {
    let reduceMotion = false
    const requestFrame = vi.spyOn(window, "requestAnimationFrame").mockImplementation(() => 1)
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined)
    vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
      matches: reduceMotion,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    const initial = [
      { x: 0, y: 20, payload: { timestamp: "2026-01-01T00:00:00Z" } },
      { x: 100, y: 10, payload: { timestamp: "2026-01-01T00:00:01Z" } },
    ]
    const disconnected = [
      { x: 0, y: 8, payload: { timestamp: "2026-01-01T00:01:00Z" } },
      { x: 100, y: 4, payload: { timestamp: "2026-01-01T00:01:01Z" } },
    ]
    const reduced = [
      { x: 0, y: 4, payload: { timestamp: "2026-01-01T00:01:01Z" } },
      { x: 100, y: 2, payload: { timestamp: "2026-01-01T00:01:02Z" } },
    ]
    const view = render(<svg><SmoothTrafficCurve points={initial} /></svg>)

    view.rerender(<svg><SmoothTrafficCurve points={disconnected} /></svg>)
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({ points: disconnected })
    expect(requestFrame).not.toHaveBeenCalled()

    reduceMotion = true
    view.rerender(<svg><SmoothTrafficCurve points={reduced} /></svg>)
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({ points: reduced })
    expect(requestFrame).not.toHaveBeenCalled()
  })

  it("distinguishes unavailable points from an explicit empty series", () => {
    const initial = [
      { x: 0, y: 20, payload: { timestamp: "2026-01-01T00:00:00Z" } },
      { x: 100, y: 10, payload: { timestamp: "2026-01-01T00:00:01Z" } },
    ]
    const view = render(<svg><SmoothTrafficCurve points={initial} /></svg>)

    view.rerender(<svg><SmoothTrafficCurve points={undefined} /></svg>)
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({ points: initial })

    view.rerender(<svg><SmoothTrafficCurve points={[]} /></svg>)
    expect(recharts.curve.mock.calls.at(-1)?.[0]).toMatchObject({ points: [] })
  })
})
