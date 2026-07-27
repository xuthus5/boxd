import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { I18nextProvider } from "react-i18next"

import { CopyTagButton } from "@/features/proxy/copy-tag-button"
import { i18n } from "@/i18n"

describe("CopyTagButton", () => {
  it("renders accessible copy control", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <CopyTagButton tag="mixed-in" />
      </I18nextProvider>,
    )
    expect(screen.getByRole("button", { name: "复制 tag" })).toBeInTheDocument()
  })
})
