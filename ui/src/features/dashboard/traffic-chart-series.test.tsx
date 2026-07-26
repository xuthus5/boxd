import { render } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const recharts = vi.hoisted(() => ({
  plotArea: undefined as { x: number; y: number; width: number; height: number } | undefined,
}))

vi.mock("recharts", () => ({ usePlotArea: () => recharts.plotArea }))

import { TrafficSeries } from "@/features/dashboard/traffic-chart-series"

const origin = Date.parse("2026-01-01T00:00:00Z")

describe("TrafficSeries", () => {
  beforeEach(() => {
    recharts.plotArea = { x: 72, y: 4, width: 720, height: 220 }
  })

  it("deduplicates timestamps and ignores invalid or out-of-order samples", () => {
    const data = [
      { timestamp: "2026-01-01T00:00:00Z", upload_rate: 1 },
      { timestamp: "2026-01-01T00:00:01Z", upload_rate: 2 },
      { timestamp: "2026-01-01T00:00:01Z", upload_rate: 3 },
      { timestamp: "2026-01-01T00:00:00.500Z", upload_rate: 4 },
      { timestamp: "invalid", upload_rate: 5 },
      { timestamp: "2026-01-01T00:00:02Z", upload_rate: Number.POSITIVE_INFINITY },
    ]
    const view = render(<svg><TrafficSeries
      data={data}
      dataKey="upload_rate"
      timeOrigin={origin}
      timeDomain={[origin - 60_000, origin]}
      valueDomain={[0, 4]}
    /></svg>)

    const segments = view.container.querySelectorAll('path[data-series="upload_rate"]')
    expect(segments).toHaveLength(1)
    expect(segments[0]).toHaveAttribute("d", "M0,-1L1000,-3")
  })

  it("handles unavailable layout, empty data, and degenerate domains", () => {
    const data = [
      { timestamp: "2026-01-01T00:00:00Z", upload_rate: 1 },
      { timestamp: "2026-01-01T00:00:01Z", upload_rate: 2 },
    ]
    recharts.plotArea = undefined
    const view = render(<svg><TrafficSeries
      data={data}
      dataKey="upload_rate"
      timeOrigin={origin}
      timeDomain={[origin, origin]}
      valueDomain={[0, 0]}
    /></svg>)
    expect(view.container.querySelector("[data-traffic-series]")).toBeNull()

    recharts.plotArea = { x: 72, y: 4, width: 720, height: 220 }
    view.rerender(<svg><TrafficSeries
      data={data}
      dataKey="upload_rate"
      timeOrigin={origin}
      timeDomain={[origin, origin]}
      valueDomain={[0, 0]}
    /></svg>)
    expect(view.container.querySelector("[data-traffic-series]")).toHaveStyle({
      transform: "matrix(1, 0, 0, 1, 72, 224)",
    })

    view.rerender(<svg><TrafficSeries
      data={[]}
      dataKey="upload_rate"
      timeOrigin={origin}
      timeDomain={[origin, origin + 60_000]}
      valueDomain={[0, 1]}
    /></svg>)
    expect(view.container.querySelectorAll('path[data-series="upload_rate"]')).toHaveLength(0)
  })
})
