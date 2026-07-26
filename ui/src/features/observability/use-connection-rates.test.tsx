import { renderHook, waitFor } from "@testing-library/react"
import { StrictMode } from "react"
import { describe, expect, it } from "vitest"

import { useConnectionRates } from "@/features/observability/use-connection-rates"
import type { Connection } from "@/lib/api/types"

function connection(upload: number, download: number): Connection {
  return {
    id: 1,
    target: "api.example.com:443",
    outbound: "proxy",
    upload,
    download,
    start: "2026-07-26T00:00:00Z",
  }
}

describe("useConnectionRates", () => {
  it("ignores StrictMode effect replays for the same snapshot", async () => {
    let now = 0
    const clock = () => {
      now += 1000
      return now
    }
    const snapshot = [connection(100, 200)]
    const { result } = renderHook(
      () => useConnectionRates(snapshot, clock),
      { wrapper: StrictMode },
    )

    await waitFor(() => expect(result.current[0]).not.toHaveProperty("uploadRate"))
  })

  it("tracks elapsed time between snapshots and recovers after resets", async () => {
    let now = 1000
    const clock = () => now
    const { result, rerender } = renderHook(
      ({ connections }) => useConnectionRates(connections, clock),
      { initialProps: { connections: [connection(100, 200)] } },
    )
    expect(result.current[0]).not.toHaveProperty("uploadRate")

    now = 3000
    rerender({ connections: [connection(300, 600)] })
    await waitFor(() => expect(result.current[0]).toMatchObject({ uploadRate: 100, downloadRate: 200 }))

    now = 4000
    rerender({ connections: [connection(10, 20)] })
    await waitFor(() => expect(result.current[0]).not.toHaveProperty("uploadRate"))

    now = 5000
    rerender({ connections: [connection(30, 60)] })
    await waitFor(() => expect(result.current[0]).toMatchObject({ uploadRate: 20, downloadRate: 40 }))
  })
})
