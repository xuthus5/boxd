import { useState } from "react"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it } from "vitest"

import { ProbeURLField } from "@/components/probe-url-field"
import { renderApp } from "@/test/render"

function Harness({ initial = "https://cp.cloudflare.com/", allowEmpty = false }: { initial?: string; allowEmpty?: boolean }) {
  const [value, setValue] = useState(initial)
  return <ProbeURLField id="probe" label="探测地址" value={value} onChange={setValue} allowEmpty={allowEmpty} emptyLabel="留空" />
}

describe("ProbeURLField", () => {
  it("selects presets and supports manual input", async () => {
    const user = userEvent.setup()
    renderApp(<Harness />)
    await user.click(screen.getByRole("combobox", { name: "探测地址" }))
    await user.click(await screen.findByRole("option", { name: "https://captive.apple.com" }))
    expect(screen.getByRole("combobox", { name: "探测地址" })).toHaveTextContent("https://captive.apple.com")
    expect(screen.queryByLabelText("自定义测速地址")).not.toBeInTheDocument()

    await user.click(screen.getByRole("combobox", { name: "探测地址" }))
    await user.click(await screen.findByRole("option", { name: "手动输入" }))
    await user.type(screen.getByLabelText("自定义测速地址"), "https://example.com/probe")
    expect(screen.getByLabelText("自定义测速地址")).toHaveValue("https://example.com/probe")
  })

  it("supports an empty inherit option", async () => {
    const user = userEvent.setup()
    renderApp(<Harness initial="" allowEmpty />)
    expect(screen.getByRole("combobox", { name: "探测地址" })).toHaveTextContent("留空")
    await user.click(screen.getByRole("combobox", { name: "探测地址" }))
    await user.click(await screen.findByRole("option", { name: "https://cp.cloudflare.com/" }))
    expect(screen.getByRole("combobox", { name: "探测地址" })).toHaveTextContent("https://cp.cloudflare.com/")
  })
})
