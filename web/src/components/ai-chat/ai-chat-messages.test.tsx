import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { vi } from "vitest"
import { ChatMessages } from "./ai-chat-messages"
import type { ChatMessage } from "./ai-chat-types"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string, values?: { count?: number }) =>
    key === "ai.activitySteps" ? `Steps (${values?.count})` : key }),
}))

const completedTools: ChatMessage[] = [
  { id: "1", role: "tool", content: "", toolName: "list_resources", toolArgs: { kind: "pods" }, actionStatus: "confirmed", toolResult: "pods" },
  { id: "2", role: "tool", content: "", toolName: "get_cluster_overview", actionStatus: "confirmed", toolResult: "healthy" },
]

describe("ChatMessages", () => {
  it("keeps the conversation in its own vertical scroll container", () => {
    render(
      <ChatMessages messages={[]} isLoading={false} onApprove={vi.fn()} onDeny={vi.fn()} />
    )

    expect(screen.getByLabelText("ai.conversation")).toHaveClass("overflow-y-auto", "min-h-0")
  })

  it("collapses completed tool activity until the user expands it", () => {
    render(
      <ChatMessages messages={completedTools} isLoading={false} onApprove={vi.fn()} onDeny={vi.fn()} />
    )

    expect(screen.getByRole("button", { name: "Steps (2)" })).toHaveAttribute("aria-expanded", "false")
    expect(screen.queryByText("List pods")).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "Steps (2)" }))
    expect(screen.getByText("List pods")).toBeInTheDocument()
  })

  it("keeps an action awaiting confirmation visible", () => {
    const pending: ChatMessage = {
      id: "pending",
      role: "tool",
      content: "",
      toolName: "delete_resource",
      toolArgs: { kind: "Pod", namespace: "default", name: "web" },
      actionStatus: "pending",
    }

    render(
      <ChatMessages messages={[pending]} isLoading={false} onApprove={vi.fn()} onDeny={vi.fn()} />
    )

    expect(screen.getByRole("button", { name: "ai.confirm" })).toBeInTheDocument()
  })

  it("does not force the user back to the bottom after they scroll up", () => {
    const { rerender } = render(
      <ChatMessages messages={[{ id: "1", role: "assistant", content: "first" }]} isLoading onApprove={vi.fn()} onDeny={vi.fn()} />
    )
    const viewport = screen.getByLabelText("ai.conversation")
    Object.defineProperties(viewport, {
      scrollHeight: { configurable: true, value: 1000 },
      clientHeight: { configurable: true, value: 300 },
    })
    viewport.scrollTop = 100
    fireEvent.scroll(viewport)

    rerender(
      <ChatMessages messages={[{ id: "1", role: "assistant", content: "first and more output" }]} isLoading onApprove={vi.fn()} onDeny={vi.fn()} />
    )

    expect(viewport.scrollTop).toBe(100)
    expect(screen.getByRole("button", { name: "ai.jumpToLatest" })).toBeInTheDocument()
  })

  it("copies the raw content of user and assistant messages", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    })
    render(
      <ChatMessages
        messages={[
          { id: "user", role: "user", content: "inspect nginx" },
          { id: "assistant", role: "assistant", content: "**Nginx** is healthy." },
        ]}
        isLoading={false}
        onApprove={vi.fn()}
        onDeny={vi.fn()}
      />
    )

    const copyButtons = screen.getAllByRole("button", { name: "ai.copyMessage" })
    fireEvent.click(copyButtons[0])
    fireEvent.click(copyButtons[1])

    await waitFor(() => {
      expect(writeText).toHaveBeenNthCalledWith(1, "inspect nginx")
      expect(writeText).toHaveBeenNthCalledWith(2, "**Nginx** is healthy.")
    })
    expect(screen.getAllByRole("button", { name: "ai.copiedMessage" })).toHaveLength(2)
  })
})
