import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { streamSSE } from "./ai-chat-stream"
import { loadStoredChat, MAX_CHAT_SESSIONS, saveStoredChat } from "./ai-chat-storage"
import type { APIChatMessage, ChatMessage, ChatSession, ChatWorkspace, PageContext } from "./ai-chat-types"

const CHAT_URL = "/api/v1/ai/chat"
const CONTINUE_URL = "/api/v1/ai/continue-action"
const TITLE_LENGTH = 42

function newID(): string {
  return crypto.randomUUID()
}

function createSession(): ChatSession {
  return { id: newID(), title: "", draft: "", messages: [], updatedAt: Date.now(), isRunning: false }
}

function createWorkspace(): ChatWorkspace {
  const session = createSession()
  return { activeSessionId: session.id, sessions: [session] }
}

function initialWorkspace(userId?: number): ChatWorkspace {
  return loadStoredChat(userId) ?? createWorkspace()
}

function shortTitle(text: string): string {
  const line = text.replace(/\s+/g, " ").trim()
  return line.length <= TITLE_LENGTH ? line : `${line.slice(0, TITLE_LENGTH - 1).trimEnd()}…`
}

/** Manages independently routable local conversations and their streams. */
export function useAIChat(userId?: number) {
  const [workspace, setWorkspace] = useState<ChatWorkspace>(() => initialWorkspace(userId))
  const workspaceRef = useRef(workspace)
  const loadedUserIDRef = useRef(userId)
  const workspaceUserIDRef = useRef(userId)
  const controllersRef = useRef(new Map<string, AbortController>())
  const activeAssistantsRef = useRef(new Map<string, string>())
  const streamClusterIDsRef = useRef(new Map<string, number>())
  workspaceRef.current = workspace

  const updateSession = useCallback((sessionId: string, update: (session: ChatSession) => ChatSession) => {
    setWorkspace((current) => ({
      ...current,
      sessions: current.sessions.map((session) => session.id === sessionId ? update(session) : session),
    }))
  }, [])

  useEffect(() => {
    if (loadedUserIDRef.current === userId) return
    saveStoredChat(workspaceUserIDRef.current, workspaceRef.current)
    controllersRef.current.forEach((controller) => controller.abort())
    controllersRef.current.clear()
    activeAssistantsRef.current.clear()
    streamClusterIDsRef.current.clear()
    const nextWorkspace = initialWorkspace(userId)
    loadedUserIDRef.current = userId
    workspaceUserIDRef.current = userId
    workspaceRef.current = nextWorkspace
    setWorkspace(nextWorkspace)
  }, [userId])

  useEffect(() => {
    if (loadedUserIDRef.current !== userId) return
    const timer = window.setTimeout(() => {
      if (workspaceUserIDRef.current === userId) saveStoredChat(userId, workspaceRef.current)
    }, 250)
    return () => window.clearTimeout(timer)
  }, [workspace, userId])

  useEffect(() => {
    const persistLatest = () => {
      if (workspaceUserIDRef.current === userId) saveStoredChat(userId, workspaceRef.current)
    }
    window.addEventListener("pagehide", persistLatest)
    return () => window.removeEventListener("pagehide", persistLatest)
  }, [userId])

  useEffect(() => () => {
    saveStoredChat(workspaceUserIDRef.current, workspaceRef.current)
    controllersRef.current.forEach((controller) => controller.abort())
  }, [])

  const patchMessage = useCallback((sessionId: string, id: string, patch: Partial<ChatMessage>) => {
    updateSession(sessionId, (session) => ({
      ...session,
      messages: session.messages.map((message) => message.id === id ? { ...message, ...patch } : message),
      updatedAt: Date.now(),
    }))
  }, [updateSession])

  const handleEvent = useCallback((sessionId: string, event: string, data: Record<string, unknown>) => {
    if (!workspaceRef.current.sessions.some((session) => session.id === sessionId)) return
    const append = (message: ChatMessage) => updateSession(sessionId, (session) => ({
      ...session,
      messages: [...session.messages, message],
      updatedAt: Date.now(),
    }))

    switch (event) {
      case "message": {
        const delta = String(data.content ?? "")
        if (!delta) return
        const activeID = activeAssistantsRef.current.get(sessionId)
        if (activeID) {
          updateSession(sessionId, (session) => ({
            ...session,
            messages: session.messages.map((message) =>
              message.id === activeID ? { ...message, content: message.content + delta } : message
            ),
            updatedAt: Date.now(),
          }))
        } else {
          const id = newID()
          activeAssistantsRef.current.set(sessionId, id)
          append({ id, role: "assistant", content: delta })
        }
        break
      }
      case "tool_call":
        activeAssistantsRef.current.delete(sessionId)
        append({
          id: newID(), role: "tool", content: "",
          toolCallId: String(data.tool_call_id ?? ""), toolName: String(data.tool ?? ""),
          toolArgs: (data.args as Record<string, unknown>) ?? {}, actionStatus: "running",
        })
        break
      case "tool_result": {
        const callID = String(data.tool_call_id ?? "")
        updateSession(sessionId, (session) => ({
          ...session,
          messages: session.messages.map((message) => message.toolCallId === callID ? {
            ...message,
            toolResult: String(data.result ?? ""),
            isError: Boolean(data.is_error),
            actionStatus: data.is_error ? "error" : "confirmed",
          } : message),
          updatedAt: Date.now(),
        }))
        break
      }
      case "action_required":
        activeAssistantsRef.current.delete(sessionId)
        append({
          id: newID(), role: "tool", content: "",
          toolCallId: String(data.tool_call_id ?? ""), toolName: String(data.tool ?? ""),
          toolArgs: (data.args as Record<string, unknown>) ?? {},
          clusterId: streamClusterIDsRef.current.get(sessionId),
          pendingSessionId: String(data.session_id ?? ""), actionStatus: "pending",
        })
        break
      case "error":
        activeAssistantsRef.current.delete(sessionId)
        append({
          id: newID(), role: "assistant",
          content: `⚠️ ${String(data.message ?? "Something went wrong")}`, isError: true,
        })
        break
    }
  }, [updateSession])

  const runStream = useCallback(async (sessionId: string, url: string, body: unknown) => {
    if (controllersRef.current.has(sessionId)) return
    const controller = new AbortController()
    controllersRef.current.set(sessionId, controller)
    updateSession(sessionId, (session) => ({ ...session, isRunning: true, updatedAt: Date.now() }))
    try {
      await streamSSE(url, body, {
        onEvent: (event, data) => handleEvent(sessionId, event, data),
        signal: controller.signal,
      })
    } catch (error) {
      if (!controller.signal.aborted) {
        handleEvent(sessionId, "error", { message: (error as Error)?.message ?? "Request failed" })
      }
    } finally {
      activeAssistantsRef.current.delete(sessionId)
      if (controllersRef.current.get(sessionId) === controller) {
        controllersRef.current.delete(sessionId)
        streamClusterIDsRef.current.delete(sessionId)
        updateSession(sessionId, (session) => ({ ...session, isRunning: false, updatedAt: Date.now() }))
      }
    }
  }, [handleEvent, updateSession])

  const createNewSession = useCallback(() => {
    const session = createSession()
    setWorkspace((current) => {
      const retained = current.sessions
        .slice()
        .sort((a, b) => b.updatedAt - a.updatedAt)
        .slice(0, MAX_CHAT_SESSIONS - 1)
      const retainedIDs = new Set(retained.map((item) => item.id))
      current.sessions.forEach((item) => {
        if (!retainedIDs.has(item.id)) {
          controllersRef.current.get(item.id)?.abort()
          streamClusterIDsRef.current.delete(item.id)
        }
      })
      return { activeSessionId: session.id, sessions: [session, ...retained] }
    })
  }, [])

  const selectSession = useCallback((sessionId: string) => {
    setWorkspace((current) => current.sessions.some((session) => session.id === sessionId)
      ? { ...current, activeSessionId: sessionId }
      : current)
  }, [])

  const updateDraft = useCallback((sessionId: string, draft: string) => {
    updateSession(sessionId, (session) => ({ ...session, draft }))
  }, [updateSession])

  const renameSession = useCallback((sessionId: string, title: string) => {
    const trimmed = title.trim()
    if (!trimmed || trimmed.length > 60) return
    updateSession(sessionId, (session) => {
      if (session.isRunning || session.title === trimmed) return session
      return { ...session, title: trimmed, updatedAt: Date.now() }
    })
  }, [updateSession])

  const deleteSession = useCallback((sessionId: string) => {
    controllersRef.current.get(sessionId)?.abort()
    controllersRef.current.delete(sessionId)
    activeAssistantsRef.current.delete(sessionId)
    streamClusterIDsRef.current.delete(sessionId)
    setWorkspace((current) => {
      if (!current.sessions.some((session) => session.id === sessionId)) return current
      const remaining = current.sessions
        .filter((session) => session.id !== sessionId)
        .sort((a, b) => b.updatedAt - a.updatedAt)
      if (!remaining.length) {
        const replacement = createSession()
        return { activeSessionId: replacement.id, sessions: [replacement] }
      }
      return {
        activeSessionId: current.activeSessionId === sessionId ? remaining[0].id : current.activeSessionId,
        sessions: remaining,
      }
    })
  }, [])

  const sendMessage = useCallback((sessionId: string, text: string, clusterId: number, pageContext?: PageContext) => {
    const trimmed = text.trim()
    const session = workspaceRef.current.sessions.find((item) => item.id === sessionId)
    if (!trimmed || !session || session.isRunning || controllersRef.current.has(sessionId)) return
    const userMessage: ChatMessage = { id: newID(), role: "user", content: trimmed }
    streamClusterIDsRef.current.set(sessionId, clusterId)
    const history: APIChatMessage[] = [...session.messages, userMessage]
      .filter((message) => message.role === "user" || message.role === "assistant")
      .map((message) => ({ role: message.role as "user" | "assistant", content: message.content }))
    updateSession(sessionId, (current) => ({
      ...current,
      title: current.messages.some((message) => message.role === "user") ? current.title : shortTitle(trimmed),
      draft: "",
      messages: [...current.messages, userMessage],
      updatedAt: Date.now(),
    }))
    void runStream(sessionId, CHAT_URL, { clusterId, messages: history, pageContext })
  }, [runStream, updateSession])

  const approveAction = useCallback((sessionId: string, message: ChatMessage) => {
    if (!message.pendingSessionId || controllersRef.current.has(sessionId)) return
    const pendingSessionId = message.pendingSessionId
    if (message.clusterId !== undefined) streamClusterIDsRef.current.set(sessionId, message.clusterId)
    patchMessage(sessionId, message.id, { actionStatus: "running", pendingSessionId: undefined })
    void runStream(sessionId, CONTINUE_URL, { sessionId: pendingSessionId })
  }, [patchMessage, runStream])

  const denyAction = useCallback((sessionId: string, message: ChatMessage, cancellationText: string) => {
    patchMessage(sessionId, message.id, { actionStatus: "denied", pendingSessionId: undefined })
    updateSession(sessionId, (session) => ({
      ...session,
      messages: [...session.messages, { id: newID(), role: "assistant", content: cancellationText }],
      updatedAt: Date.now(),
    }))
  }, [patchMessage, updateSession])

  const stop = useCallback((sessionId: string) => {
    controllersRef.current.get(sessionId)?.abort()
  }, [])

  const activeSession = useMemo(
    () => workspace.sessions.find((session) => session.id === workspace.activeSessionId) ?? workspace.sessions[0],
    [workspace]
  )

  return {
    sessions: workspace.sessions,
    activeSession,
    createNewSession,
    selectSession,
    updateDraft,
    renameSession,
    deleteSession,
    sendMessage,
    approveAction,
    denyAction,
    stop,
  }
}
