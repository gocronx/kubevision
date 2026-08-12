import type { ActionStatus, ChatMessage, ChatRole, ChatSession, ChatWorkspace } from "./ai-chat-types"

const STORAGE_PREFIX = "kubevision:ai-chat:"
const STORAGE_VERSION = 2
const LEGACY_STORAGE_VERSION = 1
export const MAX_CHAT_SESSIONS = 20
const MAX_MESSAGES = 100
const MAX_CONTENT_LENGTH = 200_000
const MAX_DRAFT_LENGTH = 20_000
const MAX_TOOL_RESULT_LENGTH = 200_000

interface StoredChatV1 {
  version: number
  messages: ChatMessage[]
}

interface StoredChatV2 extends ChatWorkspace {
  version: number
}

function storageKey(userId: number): string {
  return `${STORAGE_PREFIX}${userId}`
}

function isRole(value: unknown): value is ChatRole {
  return value === "user" || value === "assistant" || value === "tool"
}

function isStatus(value: unknown): value is ActionStatus | undefined {
  return value === undefined || value === "pending" || value === "running" || value === "confirmed" || value === "denied" || value === "error"
}

function isChatMessage(value: unknown): value is ChatMessage {
  if (!value || typeof value !== "object") return false
  const message = value as Partial<ChatMessage>
  return typeof message.id === "string"
    && isRole(message.role)
    && typeof message.content === "string"
    && message.content.length <= MAX_CONTENT_LENGTH
    && isStatus(message.actionStatus)
}

function isChatSession(value: unknown): value is ChatSession {
  if (!value || typeof value !== "object") return false
  const session = value as Partial<ChatSession>
  return typeof session.id === "string"
    && typeof session.title === "string"
    && session.title.length <= 200
    && (session.draft === undefined || (typeof session.draft === "string" && session.draft.length <= MAX_DRAFT_LENGTH))
    && typeof session.updatedAt === "number"
    && Number.isFinite(session.updatedAt)
    && Array.isArray(session.messages)
}

function cleanMessages(messages: unknown[]): ChatMessage[] {
  return messages.filter(isChatMessage).slice(-MAX_MESSAGES)
}

function messageForStorage(message: ChatMessage): ChatMessage {
  return {
    ...message,
    content: message.content.slice(0, MAX_CONTENT_LENGTH),
    toolResult: message.toolResult?.slice(0, MAX_TOOL_RESULT_LENGTH),
  }
}

export function loadStoredChat(userId?: number): ChatWorkspace | null {
  if (!userId) return null
  try {
    const raw = sessionStorage.getItem(storageKey(userId))
    if (!raw) return null
    const stored = JSON.parse(raw) as Partial<StoredChatV1 & StoredChatV2>
    if (stored.version === LEGACY_STORAGE_VERSION && Array.isArray(stored.messages)) {
      const messages = cleanMessages(stored.messages)
      if (!messages.length) return null
      const firstUserMessage = messages.find((message) => message.role === "user")
      return {
        activeSessionId: "migrated",
        sessions: [{
          id: "migrated",
          title: firstUserMessage?.content.replace(/\s+/g, " ").trim().slice(0, 42) ?? "",
          draft: "",
          messages,
          updatedAt: Date.now(),
          isRunning: false,
        }],
      }
    }
    if (stored.version !== STORAGE_VERSION || !Array.isArray(stored.sessions)) return null
    const sessions = stored.sessions
      .filter(isChatSession)
      .map((session) => ({
        ...session,
        draft: session.draft ?? "",
        messages: cleanMessages(session.messages),
        isRunning: false,
      }))
      .sort((a, b) => b.updatedAt - a.updatedAt)
      .slice(0, MAX_CHAT_SESSIONS)
    if (!sessions.length) return null
    const requestedActive = typeof stored.activeSessionId === "string" ? stored.activeSessionId : ""
    const activeSessionId = sessions.some((session) => session.id === requestedActive)
      ? requestedActive
      : sessions[0].id
    return { sessions, activeSessionId }
  } catch {
    return null
  }
}

export function saveStoredChat(userId: number | undefined, workspace: ChatWorkspace): void {
  if (!userId) return
  try {
    const stored: StoredChatV2 = {
      version: STORAGE_VERSION,
      activeSessionId: workspace.activeSessionId,
      sessions: workspace.sessions
        .slice()
        .sort((a, b) => b.updatedAt - a.updatedAt)
        .slice(0, MAX_CHAT_SESSIONS)
        .map((session) => ({
          ...session,
          draft: session.draft.slice(0, MAX_DRAFT_LENGTH),
          messages: session.messages.slice(-MAX_MESSAGES).map(messageForStorage),
        })),
    }
    sessionStorage.setItem(storageKey(userId), JSON.stringify(stored))
  } catch {
    // Storage may be disabled or full; chat remains available in memory.
  }
}

export function clearStoredChat(userId?: number): void {
  if (!userId) return
  sessionStorage.removeItem(storageKey(userId))
}
