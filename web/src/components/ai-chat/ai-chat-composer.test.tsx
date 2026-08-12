import { fireEvent, render, screen } from "@testing-library/react"
import { vi } from "vitest"
import { ChatComposer } from "./ai-chat-composer"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

describe("ChatComposer", () => {
  it("does not send when Enter is pressed", () => {
    const onSend = vi.fn()
    render(
      <ChatComposer
        isLoading={false}
        value="inspect the cluster"
        onChange={vi.fn()}
        onSend={onSend}
        onStop={vi.fn()}
      />
    )

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" })

    expect(onSend).not.toHaveBeenCalled()
  })

  it("sends the trimmed message only from the send button", () => {
    const onSend = vi.fn()
    render(
      <ChatComposer
        isLoading={false}
        value="  inspect the cluster  "
        onChange={vi.fn()}
        onSend={onSend}
        onStop={vi.fn()}
      />
    )

    fireEvent.click(screen.getByRole("button", { name: "ai.send" }))

    expect(onSend).toHaveBeenCalledOnce()
    expect(onSend).toHaveBeenCalledWith("inspect the cluster")
  })
})
