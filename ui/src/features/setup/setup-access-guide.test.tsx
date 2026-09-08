import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { afterEach, describe, expect, it, vi } from "vitest"
import { toast } from "sonner"

import { SetupAccessGuide } from "@/features/setup/setup-access-guide"
import { copyText } from "@/lib/clipboard"
import { renderApp } from "@/test/render"
import { setupStatus } from "@/test/setup-fixtures"

vi.mock("@/lib/clipboard", () => ({ copyText: vi.fn() }))
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))
afterEach(() => vi.clearAllMocks())

describe("SetupAccessGuide", () => {
  it("copies publication and verification commands while explaining their scope", async () => {
    vi.mocked(copyText).mockResolvedValue()
    const user = userEvent.setup()
    renderApp(<SetupAccessGuide status={setupStatus({ listeners: [{ tag: "mixed", type: "mixed", listen: "0.0.0.0", listen_port: 1080 }] })} />)
    expect(screen.getByText(/不代表客户端或宿主机已能连接/)).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "复制命令: Podman 宿主端口映射" }))
    expect(copyText).toHaveBeenLastCalledWith("-p 127.0.0.1:1080:1080")
    expect(toast.success).toHaveBeenCalled()
    vi.mocked(copyText).mockRejectedValue(new Error("clipboard unavailable"))
    await user.click(screen.getByRole("button", { name: "复制命令: 客户端验证命令" }))
    expect(toast.error).toHaveBeenCalledWith("复制失败，请手动复制")
  })
  it("explains why a container loopback listener cannot be published", () => {
    renderApp(<SetupAccessGuide status={setupStatus({ listeners: [{ tag: "mixed", type: "mixed", listen: "127.0.0.1", listen_port: 1080 }] })} />)
    expect(screen.getByText(/端口映射无法转发到它/)).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /复制命令/ })).not.toBeInTheDocument()
  })
  it("explains TUN namespace scope and handles empty listener lists", () => {
    const view = renderApp(<SetupAccessGuide status={setupStatus({ listeners: [{ tag: "tun", type: "tun", listen: "" }] })} />)
    expect(screen.getByText(/此 TUN 位于服务端网络环境/)).toBeInTheDocument()
    view.unmount()
    renderApp(<SetupAccessGuide status={setupStatus()} />)
    expect(screen.getByText(/尚无可用的代理监听/)).toBeInTheDocument()
  })
})
