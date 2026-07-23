import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"

import { LatencyHistoryDialog } from "@/features/nodes/latency-history-dialog"
import {
  latencyHistoryChartRows,
  latencyHistoryTypes,
  summarizeLatencyHistory,
} from "@/features/nodes/latency-history-model"
import { i18n } from "@/i18n"

describe("latency history helpers", () => {
  it("summarizes and charts successful samples", () => {
    const points = [
      { timestamp: "2026-07-23T00:00:00Z", success: true, latency_ms: 10 },
      { timestamp: "2026-07-23T00:01:00Z", success: false, error: "timeout" },
      { timestamp: "2026-07-23T00:02:00Z", success: true, latency_ms: 30 },
    ]
    expect(summarizeLatencyHistory(points)).toMatchObject({ count: 3, success: 2, latest: 30, min: 10, max: 30 })
    expect(latencyHistoryChartRows(points)[1].latency).toBeNull()
    expect(latencyHistoryTypes({ tcp: points, http: [] })).toEqual(["tcp"])
  })
})

describe("LatencyHistoryDialog", () => {
  it("opens detail dialog with chart and samples", async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <LatencyHistoryDialog
          tag="hk-01"
          history={{
            tcp: [
              { timestamp: "2026-07-23T00:00:00Z", success: true, latency_ms: 12 },
              { timestamp: "2026-07-23T00:01:00Z", success: true, latency_ms: 28 },
              { timestamp: "2026-07-23T00:02:00Z", success: false, error: "timeout" },
            ],
          }}
        />
      </I18nextProvider>,
    )
    await userEvent.click(screen.getByRole("button", { name: "延迟历史" }))
    expect(await screen.findByTestId("latency-history-dialog")).toBeInTheDocument()
    expect(screen.getByText(/hk-01/)).toBeInTheDocument()
    expect(screen.getByText("TCP")).toBeInTheDocument()
  })
})
