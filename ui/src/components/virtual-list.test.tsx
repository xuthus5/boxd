import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { VirtualList } from "@/components/virtual-list"

describe("VirtualList", () => {
  it("renders only a window of items", () => {
    const items = Array.from({ length: 200 }, (_, index) => `item-${index}`)
    render(
      <VirtualList
        className="h-40"
        items={items}
        itemHeight={40}
        overscan={1}
        getKey={(item) => item}
        renderItem={(item) => <div>{item}</div>}
      />,
    )
    // ResizeObserver may be absent in jsdom; force viewport via scroll container style height
    const region = screen.getByRole("list")
    Object.defineProperty(region, "clientHeight", { configurable: true, value: 160 })
    fireEvent.scroll(region, { target: { scrollTop: 0 } })
    expect(screen.getByText("item-0")).toBeInTheDocument()
    expect(screen.queryByText("item-50")).not.toBeInTheDocument()
  })
})
