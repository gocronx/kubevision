import { fireEvent, render, screen } from "@testing-library/react"
import { vi } from "vitest"
import { ChatSessions } from "./ai-chat-sessions"
import type { ChatSession } from "./ai-chat-types"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, values?: { title?: string }) => values?.title ? `${key}:${values.title}` : key,
    i18n: { language: "en" },
  }),
}))

const sessions: ChatSession[] = [
  { id: "first", title: "Cluster health", draft: "", messages: [], updatedAt: 1, isRunning: false },
  { id: "second", title: "Pod logs", draft: "", messages: [], updatedAt: 2, isRunning: false },
]

describe("ChatSessions", () => {
  const openMenu = () => {
    fireEvent.keyDown(screen.getByRole("button", { name: /Cluster health/ }), { key: "Enter" })
  }

  const renderSessions = (props: Partial<React.ComponentProps<typeof ChatSessions>> = {}) => {
    const handlers = {
      onCreate: vi.fn(),
      onSelect: vi.fn(),
      onRename: vi.fn(),
      onDelete: vi.fn(),
    }
    render(
      <ChatSessions
        sessions={sessions}
        activeSessionId="first"
        {...handlers}
        {...props}
      />
    )
    return handlers
  }

  it("shows the active conversation and switches from the dropdown", () => {
    const { onSelect } = renderSessions()

    openMenu()
    fireEvent.click(screen.getByText("Pod logs"))

    expect(onSelect).toHaveBeenCalledWith("second")
  })

  it("creates a conversation from the dropdown", () => {
    const { onCreate } = renderSessions()

    openMenu()
    fireEvent.click(screen.getByText("ai.newSession"))

    expect(onCreate).toHaveBeenCalledOnce()
  })

  it("trims a renamed conversation and submits its id", () => {
    const { onRename, onSelect } = renderSessions()

    openMenu()
    fireEvent.click(screen.getByRole("button", { name: "ai.renameSession:Pod logs" }))
    fireEvent.change(screen.getByLabelText("ai.sessionName"), { target: { value: "  Production logs  " } })
    fireEvent.click(screen.getByRole("button", { name: "ai.renameSessionSave" }))

    expect(onRename).toHaveBeenCalledWith("second", "Production logs")
    expect(onSelect).not.toHaveBeenCalled()
  })

  it("does not submit an empty or unchanged name", () => {
    const { onRename } = renderSessions()

    openMenu()
    fireEvent.click(screen.getByRole("button", { name: "ai.renameSession:Pod logs" }))
    fireEvent.change(screen.getByLabelText("ai.sessionName"), { target: { value: "   " } })
    expect(screen.getByRole("button", { name: "ai.renameSessionSave" })).toBeDisabled()
    fireEvent.change(screen.getByLabelText("ai.sessionName"), { target: { value: " Pod logs " } })
    expect(screen.getByRole("button", { name: "ai.renameSessionSave" })).toBeDisabled()
    expect(onRename).not.toHaveBeenCalled()
  })

  it("deletes only the selected conversation after explicit confirmation", () => {
    const onDelete = vi.fn()
    renderSessions({ onDelete })

    openMenu()
    fireEvent.click(screen.getByRole("button", { name: "ai.deleteSession:Pod logs" }))
    expect(onDelete).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole("button", { name: "ai.deleteSessionConfirm" }))
    expect(onDelete).toHaveBeenCalledOnce()
    expect(onDelete).toHaveBeenCalledWith("second")
  })
})
