import type { ChatWorkspace } from "./ai-chat-types"
import { clearStoredChat, loadStoredChat, saveStoredChat } from "./ai-chat-storage"

const workspace: ChatWorkspace = {
  activeSessionId: "second",
  sessions: [
    {
      id: "first", title: "Cluster status", draft: "first draft", updatedAt: 10, isRunning: false,
      messages: [{ id: "user-1", role: "user", content: "Check the cluster" }],
    },
    {
      id: "second", title: "Pod logs", draft: "second draft", updatedAt: 20, isRunning: true,
      messages: [{ id: "user-2", role: "user", content: "Read pod logs" }],
    },
  ],
}

describe("AI chat session storage", () => {
  it("restores multiple sessions and the active selection after refresh", () => {
    saveStoredChat(7, workspace)

    expect(loadStoredChat(7)).toEqual({
      activeSessionId: "second",
      sessions: [
        { ...workspace.sessions[1], isRunning: false },
        workspace.sessions[0],
      ],
    })
  })

  it("isolates workspaces by user and clears only the selected user", () => {
    saveStoredChat(7, workspace)
    saveStoredChat(8, { ...workspace, activeSessionId: "first" })

    clearStoredChat(7)
    expect(loadStoredChat(7)).toBeNull()
    expect(loadStoredChat(8)?.activeSessionId).toBe("first")
  })

  it("migrates a valid legacy single-conversation value", () => {
    sessionStorage.setItem("kubevision:ai-chat:7", JSON.stringify({
      version: 1,
      messages: [{ id: "legacy", role: "user", content: "old question" }],
    }))

    expect(loadStoredChat(7)).toMatchObject({
      activeSessionId: "migrated",
      sessions: [{ id: "migrated", title: "old question", draft: "", messages: [{ id: "legacy", content: "old question" }] }],
    })
  })

  it.each([
    "not json",
    '{"version":2,"sessions":"invalid"}',
    '{"version":2,"activeSessionId":"x","sessions":[{"id":4}]}',
  ])("ignores damaged cached value: %s", (raw) => {
    sessionStorage.setItem("kubevision:ai-chat:7", raw)
    expect(loadStoredChat(7)).toBeNull()
  })
})
