export type AIChatMode = "floating" | "full"

const AI_CHAT_MODE_STORAGE_KEY = "kubevision-ai-chat-mode"

export function loadAIChatMode(): AIChatMode {
  try {
    const mode = localStorage.getItem(AI_CHAT_MODE_STORAGE_KEY)
    return mode === "full" || mode === "floating" ? mode : "floating"
  } catch {
    return "floating"
  }
}

export function saveAIChatMode(mode: AIChatMode) {
  try {
    localStorage.setItem(AI_CHAT_MODE_STORAGE_KEY, mode)
  } catch {
    // Storage can be unavailable in restricted browser contexts.
  }
}
