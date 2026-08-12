import { act, renderHook, waitFor } from "@testing-library/react"
import { vi } from "vitest"
import type { StreamCallbacks } from "./ai-chat-stream"
import { useAIChat } from "./use-ai-chat"

interface PendingStream {
  callbacks: StreamCallbacks
  resolve: () => void
}

const pending: PendingStream[] = []

vi.mock("./ai-chat-stream", () => ({
  streamSSE: vi.fn((_url: string, _body: unknown, callbacks: StreamCallbacks) =>
    new Promise<void>((resolve) => pending.push({ callbacks, resolve }))
  ),
}))

describe("useAIChat sessions", () => {
  beforeEach(() => pending.splice(0))

  it("creates and switches between isolated conversations", async () => {
    const { result } = renderHook(() => useAIChat(7))
    const first = result.current.activeSession.id
    act(() => result.current.sendMessage(first, "first question", 1))
    await waitFor(() => expect(result.current.activeSession.messages).toHaveLength(1))

    act(() => result.current.createNewSession())
    const second = result.current.activeSession.id
    act(() => result.current.sendMessage(second, "second question", 1))

    expect(result.current.activeSession.messages[0].content).toBe("second question")
    act(() => result.current.selectSession(first))
    expect(result.current.activeSession.messages[0].content).toBe("first question")
  })

  it("keeps unsent drafts isolated between conversations", () => {
    const { result } = renderHook(() => useAIChat(7))
    const first = result.current.activeSession.id
    act(() => result.current.updateDraft(first, "draft for first"))
    act(() => result.current.createNewSession())
    const second = result.current.activeSession.id
    act(() => result.current.updateDraft(second, "draft for second"))

    act(() => result.current.selectSession(first))
    expect(result.current.activeSession.draft).toBe("draft for first")
    act(() => result.current.selectSession(second))
    expect(result.current.activeSession.draft).toBe("draft for second")
  })

  it("renames only the requested conversation and trims the name", () => {
    const { result } = renderHook(() => useAIChat(7))
    const first = result.current.activeSession.id
    act(() => result.current.createNewSession())
    const second = result.current.activeSession.id

    act(() => result.current.renameSession(first, "  Cluster review  "))

    expect(result.current.sessions.find((session) => session.id === first)?.title).toBe("Cluster review")
    expect(result.current.sessions.find((session) => session.id === second)?.title).toBe("")
  })

  it("rejects empty, unchanged, oversized, and running conversation names", async () => {
    const { result } = renderHook(() => useAIChat(7))
    const sessionId = result.current.activeSession.id
    act(() => result.current.renameSession(sessionId, "Initial name"))
    const updatedAt = result.current.activeSession.updatedAt

    act(() => result.current.renameSession(sessionId, "  "))
    act(() => result.current.renameSession(sessionId, "Initial name"))
    act(() => result.current.renameSession(sessionId, "x".repeat(61)))
    expect(result.current.activeSession.title).toBe("Initial name")
    expect(result.current.activeSession.updatedAt).toBe(updatedAt)

    act(() => result.current.sendMessage(sessionId, "question", 1))
    await waitFor(() => expect(result.current.activeSession.isRunning).toBe(true))
    act(() => result.current.renameSession(sessionId, "Changed while running"))
    expect(result.current.activeSession.title).toBe("question")
  })

  it("routes a background stream without changing the selected conversation", async () => {
    const { result } = renderHook(() => useAIChat(7))
    const first = result.current.activeSession.id
    act(() => result.current.sendMessage(first, "background", 1))
    await waitFor(() => expect(pending).toHaveLength(1))
    act(() => result.current.createNewSession())
    const selected = result.current.activeSession.id

    act(() => pending[0].callbacks.onEvent("message", { content: "finished" }))
    act(() => pending[0].resolve())

    await waitFor(() => expect(result.current.sessions.find((item) => item.id === first)?.isRunning).toBe(false))
    expect(result.current.activeSession.id).toBe(selected)
    expect(result.current.sessions.find((item) => item.id === first)?.messages.at(-1)?.content).toBe("finished")
  })

  it("deletes only the requested session and selects the most recent remaining one", () => {
    const { result } = renderHook(() => useAIChat(7))
    const first = result.current.activeSession.id
    act(() => result.current.createNewSession())
    const second = result.current.activeSession.id

    act(() => result.current.deleteSession(second))
    expect(result.current.sessions.map((session) => session.id)).toEqual([first])
    expect(result.current.activeSession.id).toBe(first)
    act(() => result.current.deleteSession(second))
    expect(result.current.sessions).toHaveLength(1)
  })

  it("aborts a running stream before deleting its conversation", async () => {
    const { result } = renderHook(() => useAIChat(7))
    const running = result.current.activeSession.id
    act(() => result.current.sendMessage(running, "long request", 1))
    await waitFor(() => expect(pending).toHaveLength(1))
    const signal = pending[0].callbacks.signal

    act(() => result.current.deleteSession(running))

    expect(signal?.aborted).toBe(true)
    expect(result.current.sessions.some((session) => session.id === running)).toBe(false)
    act(() => pending[0].callbacks.onEvent("message", { content: "late output" }))
    expect(result.current.sessions.some((session) => session.id === running)).toBe(false)
  })

  it("does not persist the previous user's workspace under a new user", async () => {
    const { result, rerender } = renderHook(({ userId }) => useAIChat(userId), {
      initialProps: { userId: 7 },
    })
    const first = result.current.activeSession.id
    act(() => result.current.sendMessage(first, "private to seven", 1))
    await waitFor(() => expect(result.current.activeSession.messages).toHaveLength(1))

    rerender({ userId: 8 })
    await waitFor(() => expect(result.current.activeSession.messages).toHaveLength(0))
    await new Promise((resolve) => setTimeout(resolve, 300))

    expect(sessionStorage.getItem("kubevision:ai-chat:8")).not.toContain("private to seven")
  })
})
