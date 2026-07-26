import { useState, type ReactNode } from "react"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

const recharts = vi.hoisted(() => ({ line: vi.fn(), chart: vi.fn() }))

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  LineChart: ({ children, ...props }: { children: ReactNode }) => {
    recharts.chart(props)
    return <div>{children}</div>
  },
  Line: (props: Record<string, unknown>) => {
    recharts.line(props)
    return <span data-testid="traffic-line" />
  },
  CartesianGrid: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  Legend: () => null,
}))

import { TrafficChart } from "@/features/dashboard/traffic-chart"
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
  })

  it("skips unchanged parent renders and disables line replay animations", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)

    const initialLineCount = recharts.line.mock.calls.length
    expect(initialLineCount).toBeGreaterThan(0)
    for (const [props] of recharts.line.mock.calls) {
      expect(props).toMatchObject({ isAnimationActive: false })
    }

    await user.click(screen.getByRole("button", { name: "rerender parent" }))
    expect(screen.getByLabelText("unrelated state")).toHaveTextContent("1")
    expect(recharts.line).toHaveBeenCalledTimes(initialLineCount)

    await user.click(screen.getByRole("button", { name: "append sample" }))
    expect(recharts.line.mock.calls.length).toBeGreaterThan(initialLineCount)
  })
})
