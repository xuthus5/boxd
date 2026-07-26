import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { I18nextProvider } from "react-i18next"
import { MemoryRouter } from "react-router-dom"

import { ConnectionGroupTable } from "@/features/observability/connection-group-table"
import type { ConnectionGroupStat } from "@/features/observability/connection-stats"
import { i18n } from "@/i18n"

const state = vi.hoisted(() => ({ mobile: false }))

vi.mock("@/hooks/use-mobile", () => ({
  useIsMobile: () => state.mobile,
}))

const group: ConnectionGroupStat = {
  key: "proxy",
  count: 2,
  upload: 100,
  download: 200,
  uploadRate: 1024,
  downloadRate: 2048,
  rateSamples: 2,
}

function renderTable(
  groups: ConnectionGroupStat[],
  onEmptyAction?: () => void,
  onCloseGroup: (field: "outbound", key: string) => void = vi.fn(),
) {
  return render(
    <I18nextProvider i18n={i18n}>
      <MemoryRouter>
        <ConnectionGroupTable
          groups={groups}
          field="outbound"
          closingId={null}
          onCloseGroup={onCloseGroup}
          emptyTitle="暂无分组"
          emptyDescription="等待连接数据"
          emptyActionLabel={onEmptyAction ? "清除筛选" : undefined}
          onEmptyAction={onEmptyAction}
        />
      </MemoryRouter>
    </I18nextProvider>,
  )
}

afterEach(() => {
  state.mobile = false
  vi.clearAllMocks()
})

describe("ConnectionGroupTable", () => {
  it("renders the empty action", async () => {
    const onEmptyAction = vi.fn()
    renderTable([], onEmptyAction)

    expect(screen.getByText("暂无分组")).toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole("button", { name: "清除筛选" }))
    expect(onEmptyAction).toHaveBeenCalledOnce()
  })

  it("renders rates in the mobile group card", () => {
    state.mobile = true
    renderTable([group])

    expect(screen.getByText(/实时速率:/)).toHaveTextContent("↑ 1.00 KB/s · ↓ 2.00 KB/s")
    expect(screen.getByRole("link", { name: "查看列表: proxy" })).toHaveAttribute(
      "href",
      "/observability/connections?outbound=proxy",
    )
  })

  it("renders the desktop group row and confirms closing", async () => {
    const onCloseGroup = vi.fn()
    renderTable([group], undefined, onCloseGroup)

    expect(screen.getByRole("row", {
      name: /proxy 2 100 B 200 B ↑ 1.00 KB\/s · ↓ 2.00 KB\/s/,
    })).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getByRole("button", { name: "关闭该组" }))
    await user.click(await screen.findByRole("button", { name: "确认关闭" }))
    await waitFor(() => expect(onCloseGroup).toHaveBeenCalledWith("outbound", "proxy"))
  })
})
