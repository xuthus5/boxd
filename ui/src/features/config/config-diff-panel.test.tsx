import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"

import { ConfigDiffPanel } from "@/features/config/config-diff-panel"
import { i18n } from "@/i18n"

describe("ConfigDiffPanel", () => {
  it("renders none state", () => {
    render(<I18nextProvider i18n={i18n}><ConfigDiffPanel items={[]} /></I18nextProvider>)
    expect(screen.getByTestId("config-diff-panel")).toHaveTextContent(/无变化|No changes/i)
  })

  it("lists path diffs and invokes path select", async () => {
    const onSelectPath = vi.fn()
    render(
      <I18nextProvider i18n={i18n}>
        <ConfigDiffPanel
          items={[
            { path: "log.level", kind: "changed", before: "info", after: "debug" },
            { path: "dns", kind: "removed", before: { final: "local" } },
            { path: "experimental", kind: "added", after: { clash_api: true } },
          ]}
          onSelectPath={onSelectPath}
        />
      </I18nextProvider>,
    )
    expect(screen.getByText("log.level")).toBeInTheDocument()
    await userEvent.click(screen.getByRole("button", { name: "log.level" }))
    expect(onSelectPath).toHaveBeenCalledWith("log.level")
    expect(screen.getByText(/info/)).toBeInTheDocument()
    expect(screen.getByText(/debug/)).toBeInTheDocument()
  })
})
