import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { ProblemNodesPreview } from "@/features/dashboard/problem-nodes-preview"
import type { ProblemNodePreview } from "@/features/nodes/nodes-filter"
import { i18n } from "@/i18n"

const items: ProblemNodePreview[] = [
  {
    tag: "hk-bad",
    type: "vless",
    stability: "failed",
    percent: 0,
    success: 0,
    count: 3,
  },
  {
    tag: "us-flaky",
    type: "trojan",
    stability: "unstable",
    percent: 25,
    success: 1,
    latest: 180,
    count: 4,
  },
]

function renderPreview(preview = items.slice(0, 1), total = 2) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <ProblemNodesPreview items={preview} total={total} />
      </MemoryRouter>
    </I18nextProvider>,
  )
}

describe("ProblemNodesPreview", () => {
  it("renders problem rows with deep links and remaining count", () => {
    renderPreview()
    expect(screen.getByText("hk-bad")).toBeInTheDocument()
    expect(screen.getByText("全失败")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "查看: hk-bad" })).toHaveAttribute(
      "href",
      "/nodes?q=hk-bad&stability=failed",
    )
    expect(screen.getByText(/还有 1 个/)).toBeInTheDocument()
  })

  it("renders nothing when empty", () => {
    const { container } = renderPreview([], 0)
    expect(container.querySelector('[data-slot="problem-nodes-preview"]')).toBeNull()
  })
})
