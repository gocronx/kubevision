import { fireEvent, render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { AIChatWidget } from "./ai-chat-widget"

const state = vi.hoisted(() => ({
  config: { enabled: false, hasApiKey: true },
  role: "viewer",
}))

vi.mock("./use-ai-config", () => ({
  useAIConfig: () => ({ data: state.config }),
}))

vi.mock("@/stores/auth-store", () => ({
  useAuth: () => ({ user: { id: 7, role: state.role } }),
}))

vi.mock("@/hooks/use-cluster", () => ({
  useCluster: () => ({ currentCluster: "1" }),
}))

vi.mock("./use-ai-chat", () => ({
  useAIChat: () => ({
    sessions: [{ id: "session", title: "Chat", draft: "", messages: [], updatedAt: 1, isRunning: false }],
    activeSession: { id: "session", title: "Chat", draft: "", messages: [], updatedAt: 1, isRunning: false },
    createNewSession: vi.fn(), selectSession: vi.fn(), updateDraft: vi.fn(),
    renameSession: vi.fn(), deleteSession: vi.fn(), sendMessage: vi.fn(),
    approveAction: vi.fn(), denyAction: vi.fn(), stop: vi.fn(),
  }),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
}))

function renderWidget() {
  return render(<MemoryRouter><AIChatWidget /></MemoryRouter>)
}

describe("AIChatWidget availability", () => {
  beforeEach(() => {
    localStorage.clear()
    state.config = { enabled: false, hasApiKey: true }
    state.role = "viewer"
  })

  it("hides a disabled assistant from regular users", () => {
    renderWidget()
    expect(screen.queryByRole("button", { name: "ai.title" })).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "ai.disabledTitle" })).not.toBeInTheDocument()
  })

  it("shows administrators why the assistant is disabled", () => {
    state.role = "admin"
    renderWidget()

    fireEvent.click(screen.getByRole("button", { name: "ai.disabledTitle" }))
    expect(screen.getByText("ai.disabledMessage")).toBeInTheDocument()
    expect(screen.getByRole("link", { name: "ai.openSettings" })).toHaveAttribute("href", "/settings/ai")
  })

  it("distinguishes an enabled assistant with no API key", () => {
    state.config = { enabled: true, hasApiKey: false }
    renderWidget()

    fireEvent.click(screen.getByRole("button", { name: "ai.title" }))
    expect(screen.getByText("ai.missingAPIKey")).toBeInTheDocument()
    expect(screen.queryByText("ai.disabledMessage")).not.toBeInTheDocument()
  })
})

describe("AIChatWidget layout", () => {
  beforeEach(() => {
    localStorage.clear()
    state.config = { enabled: true, hasApiKey: true }
    state.role = "viewer"
  })

  it("switches between floating and full-page chat", () => {
    const { container } = renderWidget()
    fireEvent.click(screen.getByRole("button", { name: "ai.title" }))

    expect(container.querySelector('[data-chat-mode="floating"]')).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "ai.fullPageChat" }))
    expect(container.querySelector('[data-chat-mode="full"]')).toBeInTheDocument()

    fireEvent.click(screen.getByRole("button", { name: "ai.floatingChat" }))
    expect(container.querySelector('[data-chat-mode="floating"]')).toBeInTheDocument()
  })

  it("restores the saved layout when reopened after remounting", () => {
    const firstRender = renderWidget()
    fireEvent.click(screen.getByRole("button", { name: "ai.title" }))
    fireEvent.click(screen.getByRole("button", { name: "ai.fullPageChat" }))
    expect(localStorage.getItem("kubevision-ai-chat-mode")).toBe("full")
    firstRender.unmount()

    const { container } = renderWidget()
    fireEvent.click(screen.getByRole("button", { name: "ai.title" }))
    expect(container.querySelector('[data-chat-mode="full"]')).toBeInTheDocument()
  })
})
