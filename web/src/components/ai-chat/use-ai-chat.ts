import { useCallback, useRef, useState } from "react"
import { streamSSE } from "./ai-chat-stream"
import type {
  APIChatMessage,
  ChatMessage,
  PageContext,
} from "./ai-chat-types"

const CHAT_URL = "/api/v1/ai/chat"
const CONTINUE_URL = "/api/v1/ai/continue-action"

function newID(): string {
  return crypto.randomUUID()
}

/** Manages a single AI conversation: streaming, tool calls, and mutation
 *  approval. */
export function useAIChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  // ID of the assistant bubble currently receiving streamed text, if any.
  const activeAssistantRef = useRef<string | null>(null)

  const patchMessage = useCallback((id: string, patch: Partial<ChatMessage>) => {
    setMessages((prev) => prev.map((m) => (m.id === id ? { ...m, ...patch } : m)))
  }, [])

  const patchToolByCallID = useCallback(
    (callID: string, patch: Partial<ChatMessage>) => {
      setMessages((prev) =>
        prev.map((m) => (m.toolCallId === callID ? { ...m, ...patch } : m))
      )
    },
    []
  )

  // Routes one SSE event into message state.
  const handleEvent = useCallback(
    (event: string, data: Record<string, unknown>) => {
      switch (event) {
        case "message": {
          const delta = String(data.content ?? "")
          if (!delta) return
          setMessages((prev) => {
            const activeID = activeAssistantRef.current
            if (activeID) {
              return prev.map((m) =>
                m.id === activeID ? { ...m, content: m.content + delta } : m
              )
            }
            const id = newID()
            activeAssistantRef.current = id
            return [...prev, { id, role: "assistant", content: delta }]
          })
          break
        }
        case "tool_call": {
          activeAssistantRef.current = null
          setMessages((prev) => [
            ...prev,
            {
              id: newID(),
              role: "tool",
              content: "",
              toolCallId: String(data.tool_call_id ?? ""),
              toolName: String(data.tool ?? ""),
              toolArgs: (data.args as Record<string, unknown>) ?? {},
              actionStatus: "running",
            },
          ])
          break
        }
        case "tool_result": {
          const callID = String(data.tool_call_id ?? "")
          patchToolByCallID(callID, {
            toolResult: String(data.result ?? ""),
            isError: Boolean(data.is_error),
            actionStatus: data.is_error ? "error" : "confirmed",
          })
          break
        }
        case "action_required": {
          activeAssistantRef.current = null
          setMessages((prev) => [
            ...prev,
            {
              id: newID(),
              role: "tool",
              content: "",
              toolCallId: String(data.tool_call_id ?? ""),
              toolName: String(data.tool ?? ""),
              toolArgs: (data.args as Record<string, unknown>) ?? {},
              pendingSessionId: String(data.session_id ?? ""),
              actionStatus: "pending",
            },
          ])
          break
        }
        case "error": {
          activeAssistantRef.current = null
          setMessages((prev) => [
            ...prev,
            {
              id: newID(),
              role: "assistant",
              content: `⚠️ ${String(data.message ?? "Something went wrong")}`,
              isError: true,
            },
          ])
          break
        }
        // "done" needs no state change; loading is cleared by the caller.
      }
    },
    [patchToolByCallID]
  )

  const runStream = useCallback(
    async (url: string, body: unknown) => {
      setIsLoading(true)
      const controller = new AbortController()
      abortRef.current = controller
      try {
        await streamSSE(url, body, {
          onEvent: handleEvent,
          signal: controller.signal,
        })
      } catch (err) {
        handleEvent("error", { message: (err as Error)?.message ?? "Request failed" })
      } finally {
        activeAssistantRef.current = null
        setIsLoading(false)
        abortRef.current = null
      }
    },
    [handleEvent]
  )

  const sendMessage = useCallback(
    (text: string, clusterId: number, pageContext?: PageContext) => {
      const trimmed = text.trim()
      if (!trimmed || isLoading) return

      const userMsg: ChatMessage = { id: newID(), role: "user", content: trimmed }
      // Build history from the messages we already have plus the new turn.
      const history: APIChatMessage[] = [...messages, userMsg]
        .filter((m) => m.role === "user" || m.role === "assistant")
        .map((m) => ({ role: m.role as "user" | "assistant", content: m.content }))

      setMessages((prev) => [...prev, userMsg])
      void runStream(CHAT_URL, { clusterId, messages: history, pageContext })
    },
    [messages, isLoading, runStream]
  )

  const approveAction = useCallback(
    (message: ChatMessage) => {
      if (!message.pendingSessionId) return
      patchMessage(message.id, { actionStatus: "running", pendingSessionId: undefined })
      void runStream(CONTINUE_URL, { sessionId: message.pendingSessionId })
    },
    [patchMessage, runStream]
  )

  const denyAction = useCallback(
    (message: ChatMessage) => {
      patchMessage(message.id, { actionStatus: "denied", pendingSessionId: undefined })
      setMessages((prev) => [
        ...prev,
        { id: newID(), role: "assistant", content: "Action cancelled." },
      ])
    },
    [patchMessage]
  )

  const stop = useCallback(() => {
    abortRef.current?.abort()
    setIsLoading(false)
  }, [])

  const clear = useCallback(() => {
    abortRef.current?.abort()
    activeAssistantRef.current = null
    setMessages([])
    setIsLoading(false)
  }, [])

  return { messages, isLoading, sendMessage, approveAction, denyAction, stop, clear }
}
