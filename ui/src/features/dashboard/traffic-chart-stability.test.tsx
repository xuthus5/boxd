import { useState, type ReactNode } from "react"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const recharts = vi.hoisted(() => ({
  line: vi.fn(),
  chart: vi.fn(),
  grid: vi.fn(),
  plotArea: undefined as { x: number; y: number; width: number; height: number } | undefined,
  tooltip: vi.fn(),
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
  Tooltip: (props: Record<string, unknown>) => {
    recharts.tooltip(props)
    return null
  },
  Legend: () => null,
}))

import { TrafficChart } from "@/features/dashboard/traffic-chart"
import type { TrafficHistoryPoint } from "@/lib/api/types"
import { renderApp } from "@/test/render"

const initialPoints: TrafficHistoryPoint[] = [
  { timestamp: "2026-01-01T00:00:00Z", upload_bytes: 0, download_bytes: 0 },
  { timestamp: "2026-01-01T00:00:01Z", upload_bytes: 1024, download_bytes: 2048 },
]

function rollingPoint(index: number): TrafficHistoryPoint {
  const uploadBytes = ((index * (index + 1)) / 2) * 1024
  return {
    timestamp: new Date(Date.parse("2026-01-01T00:00:00Z") + (index * 1000)).toISOString(),
    upload_bytes: uploadBytes,
    download_bytes: uploadBytes * 2,
  }
}

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

function OriginBoundaryHarness() {
  const [points, setPoints] = useState<TrafficHistoryPoint[]>([
    { timestamp: "2026-01-01T00:00:58Z", upload_bytes: 0, download_bytes: 0 },
    { timestamp: "2026-01-01T00:00:59Z", upload_bytes: 1024, download_bytes: 2048 },
  ])
  return <>
    <button onClick={() => setPoints((current) => [...current, {
      timestamp: "2026-01-01T00:01:00Z", upload_bytes: 2048, download_bytes: 4096,
    }])}>cross origin boundary</button>
    <TrafficChart points={points} />
  </>
}

function ScaleHarness() {
  const [points, setPoints] = useState(initialPoints)
  return <>
    <button onClick={() => setPoints((current) => [...current, {
      timestamp: "2026-01-01T00:00:02Z", upload_bytes: 3072, download_bytes: 6144,
    }])}>increase scale</button>
    <TrafficChart points={points} />
  </>
}

function RollingWindowHarness() {
  const [points, setPoints] = useState(() => Array.from({ length: 61 }, (_, index) => rollingPoint(index)))
  return <>
    <button onClick={() => setPoints((current) => [...current.slice(1), rollingPoint(61)])}>roll window</button>
    <TrafficChart points={points} />
  </>
}

describe("TrafficChart stability", () => {
  beforeEach(() => {
    recharts.line.mockClear()
    recharts.chart.mockClear()
    recharts.grid.mockClear()
    recharts.plotArea = { x: 72, y: 4, width: 720, height: 220 }
    recharts.tooltip.mockClear()
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
    const uploadSeries = document.querySelector<SVGGElement>('g[data-traffic-series="upload_rate"]')
    expect(uploadSeries).not.toBeNull()
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
        hide: true,
        isAnimationActive: false,
        animateNewValues: false,
      })
      expect(props).not.toHaveProperty("shape")
    }
    expect(recharts.tooltip.mock.calls.at(-1)?.[0]).toMatchObject({ includeHidden: true })

    await user.click(screen.getByRole("button", { name: "rerender parent" }))
    expect(screen.getByLabelText("unrelated state")).toHaveTextContent("1")
    expect(recharts.line).toHaveBeenCalledTimes(initialLineCount)

    await user.click(screen.getByRole("button", { name: "append sample" }))
    expect(recharts.line.mock.calls.length).toBeGreaterThan(initialLineCount)
    expect(screen.getByTestId("traffic-chart-instance")).toBe(chartInstance)
    expect(recharts.line.mock.calls.at(-1)?.[0]).toMatchObject({
      activeDot: false,
      hide: true,
      isAnimationActive: false,
      animateNewValues: false,
    })
    expect(document.querySelector('g[data-traffic-series="upload_rate"]')).toBe(uploadSeries)
    const updatedTick = document.querySelector(`[data-traffic-time-tick="${stableTickValue}"]`)
    expect(updatedTick).toBe(stableTick)
    expect(updatedTick).toHaveClass("transition-transform")
    expect((updatedTick as SVGGElement).style.transform).not.toBe(initialTickTransform)
  })

  it("keeps the last curve while chart layout is temporarily unavailable", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)
    const uploadSeries = document.querySelector<SVGGElement>('g[data-traffic-series="upload_rate"]')
    expect(uploadSeries).not.toBeNull()
    const initialSegment = uploadSeries.querySelector("path")

    recharts.plotArea = undefined
    await user.click(screen.getByRole("button", { name: "append sample" }))

    expect(document.querySelector('g[data-traffic-series="upload_rate"]')).toBe(uploadSeries)
    expect(initialSegment?.isConnected).toBe(true)
    expect(uploadSeries.querySelectorAll("path").length).toBeGreaterThan(0)
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

  it("appends one segment without mutating historical geometry", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)
    const selector = 'path.traffic-chart-curve[data-series="upload_rate"]'
    const initialSegments = Array.from(document.querySelectorAll<SVGPathElement>(selector))
    const initialGeometry = initialSegments.map((segment) => segment.getAttribute("d"))

    await user.click(screen.getByRole("button", { name: "append sample" }))

    const updatedSegments = Array.from(document.querySelectorAll<SVGPathElement>(selector))
    expect(updatedSegments).toHaveLength(initialSegments.length + 1)
    for (const [index, segment] of initialSegments.entries()) {
      expect(updatedSegments[index]).toBe(segment)
      expect(segment.getAttribute("d")).toBe(initialGeometry[index])
    }
  })

  it("keeps historical segment coordinates stable across an origin boundary", async () => {
    const user = userEvent.setup()
    renderApp(<OriginBoundaryHarness />)
    const selector = 'path.traffic-chart-curve[data-series="upload_rate"]'
    const initialSegment = document.querySelector<SVGPathElement>(selector)
    const initialGeometry = initialSegment?.getAttribute("d")

    await user.click(screen.getByRole("button", { name: "cross origin boundary" }))

    expect(document.querySelector<SVGPathElement>(selector)).toBe(initialSegment)
    expect(initialSegment?.getAttribute("d")).toBe(initialGeometry)
  })

  it("rescales through the series transform without rewriting segments", async () => {
    const user = userEvent.setup()
    renderApp(<ScaleHarness />)
    const selector = 'path.traffic-chart-curve[data-series="upload_rate"]'
    const initialSegment = document.querySelector<SVGPathElement>(selector)
    const initialGeometry = initialSegment?.getAttribute("d")
    const motion = initialSegment?.parentElement
    const initialTransform = motion?.style.transform

    await user.click(screen.getByRole("button", { name: "increase scale" }))

    expect(document.querySelector<SVGPathElement>(selector)).toBe(initialSegment)
    expect(initialSegment?.getAttribute("d")).toBe(initialGeometry)
    expect(motion?.style.transform).not.toBe(initialTransform)
  })

  it("keeps retained rate segments stable when the source seed rolls", async () => {
    const user = userEvent.setup()
    renderApp(<RollingWindowHarness />)
    const selector = 'path.traffic-chart-curve[data-series="upload_rate"]'
    const retained = Array.from(document.querySelectorAll<SVGPathElement>(selector)).slice(1)
    const geometry = retained.map((segment) => segment.getAttribute("d"))

    await user.click(screen.getByRole("button", { name: "roll window" }))

    for (const [index, segment] of retained.entries()) {
      expect(segment.isConnected).toBe(true)
      expect(segment.getAttribute("d")).toBe(geometry[index])
    }
  })
})
