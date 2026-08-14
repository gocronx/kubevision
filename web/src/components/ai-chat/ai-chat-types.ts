// Shared types for the AI chat assistant.

export type ChatRole = "user" | "assistant" | "tool"

export type ActionStatus = "pending" | "running" | "confirmed" | "denied" | "error"

/** A single rendered message in the conversation. */
export interface ChatMessage {
  id: string
  role: ChatRole
  content: string

  // Tool-call fields (role === "tool").
  toolCallId?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolResult?: string
  isError?: boolean
  clusterId?: number

  // Set when a mutation is awaiting the user's approval.
  pendingSessionId?: string
  actionStatus?: ActionStatus
}

export interface ChatSession {
  id: string
  title: string
  draft: string
  messages: ChatMessage[]
  updatedAt: number
  isRunning: boolean
}

export interface ChatWorkspace {
  activeSessionId: string
  sessions: ChatSession[]
}

/** What gets sent to the backend as conversation history. */
export interface APIChatMessage {
  role: "user" | "assistant"
  content: string
}

/** Page context forwarded to the backend so the assistant knows what the user
 *  is currently looking at. */
export interface PageContext {
  page?: string
  namespace?: string
  resourceName?: string
  resourceKind?: string
}

/** Backend AI configuration (key never returned). */
export interface AIConfigView {
  enabled: boolean
  baseURL: string
  model: string
  maxTokens: number
  hasApiKey: boolean
}

/** Payload for updating the AI configuration. */
export interface AIConfigUpdate {
  enabled: boolean
  baseURL: string
  apiKey: string
  model: string
  maxTokens: number
}

export interface AIModel {
  id: string
}
