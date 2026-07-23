import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { describe, expect, it } from "vitest"

import { facetHref } from "@/features/observability/connection-facets"
import { FacetLink, MetaChip } from "@/features/observability/connection-facet-links"

describe("connection-facet-links", () => {
  it("builds facet deep-links and skips empty values", () => {
    expect(facetHref("process", "/usr/bin/curl")).toBe("/observability/connections?process=%2Fusr%2Fbin%2Fcurl")
    expect(facetHref("network", "—")).toBe("")
    expect(facetHref("rule", "  ")).toBe("")
  })

  it("renders clickable facet links and chips", () => {
    render(
      <MemoryRouter>
        <FacetLink field="outbound" value="proxy" label="出站" />
        <MetaChip field="process" label="进程" value="/usr/bin/curl" />
        <MetaChip label="来源" value="10.0.0.1:1" />
      </MemoryRouter>,
    )
    expect(screen.getByRole("link", { name: "出站: proxy" })).toHaveAttribute(
      "href",
      "/observability/connections?outbound=proxy",
    )
    expect(screen.getByRole("link", { name: "进程: /usr/bin/curl" })).toHaveAttribute(
      "href",
      "/observability/connections?process=%2Fusr%2Fbin%2Fcurl",
    )
    expect(screen.getByText("来源: 10.0.0.1:1")).toBeInTheDocument()
    expect(screen.queryByRole("link", { name: "来源: 10.0.0.1:1" })).not.toBeInTheDocument()
  })
})
