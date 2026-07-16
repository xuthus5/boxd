import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { describe, expect, it, vi } from "vitest"

import { ConfirmAction } from "@/components/confirm-action"
import { Button } from "@/components/ui/button"
import { renderApp } from "@/test/render"

describe("ConfirmAction", () => {
  it("keeps the confirmation pending until an async action completes", async () => {
    let resolveAction: (() => void) | undefined
    const action = vi.fn(() => new Promise<void>((resolve) => { resolveAction = resolve }))
    const user = userEvent.setup()
    renderApp(<ConfirmAction trigger={<Button>Delete</Button>} title="Delete item" description="Cannot undo" confirmLabel="Confirm" onConfirm={action} />)

    await user.click(screen.getByRole("button", { name: "Delete" }))
    await user.click(screen.getByRole("button", { name: "Confirm" }))
    expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled()
    resolveAction?.()
    await vi.waitFor(() => expect(screen.queryByRole("button", { name: "Confirm" })).not.toBeInTheDocument())
    expect(action).toHaveBeenCalledOnce()
  })
})
