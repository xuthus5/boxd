import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { RouteVisualEditor } from "@/features/policy/route-visual-editor"
import { ApiError } from "@/lib/api/client"
import { renderApp } from "@/test/render"

describe("RouteVisualEditor metadata densify", () => {
  it("renders densified metadata load failure with retry", async () => {
    const onRetry = vi.fn()
    const user = userEvent.setup()
    renderApp(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <RouteVisualEditor
          object={{ rules: [] }}
          revision={0}
          onChange={vi.fn()}
          onFieldValidityChange={vi.fn()}
          metadata={[]}
          metadataError={new ApiError("metadata unavailable", 503, "unavailable")}
          onMetadataRetry={onRetry}
          onMetadataChange={vi.fn()}
        />
      </QueryClientProvider>,
    )
    expect(await screen.findByTestId("page-load-error")).toHaveAttribute("data-error-code", "unavailable")
    expect(screen.getByText("metadata unavailable")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "重试" }))
    expect(onRetry).toHaveBeenCalled()
  })
})
