export interface ChatDimensions {
  width: number
  height: number
}

export const CHAT_MIN_DIMENSIONS: ChatDimensions = { width: 320, height: 360 }
export const CHAT_DEFAULT_DIMENSIONS: ChatDimensions = { width: 400, height: 600 }
export const CHAT_MAX_DIMENSIONS: ChatDimensions = { width: 900, height: 900 }

const VIEWPORT_MARGIN = 32

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(maximum, Math.max(minimum, value))
}

export function clampChatDimensions(
  dimensions: ChatDimensions,
  viewport: ChatDimensions
): ChatDimensions {
  const maxWidth = Math.max(0, Math.min(CHAT_MAX_DIMENSIONS.width, viewport.width - VIEWPORT_MARGIN))
  const maxHeight = Math.max(0, Math.min(CHAT_MAX_DIMENSIONS.height, viewport.height - VIEWPORT_MARGIN))
  const minWidth = Math.min(CHAT_MIN_DIMENSIONS.width, maxWidth)
  const minHeight = Math.min(CHAT_MIN_DIMENSIONS.height, maxHeight)

  return {
    width: clamp(dimensions.width, minWidth, maxWidth),
    height: clamp(dimensions.height, minHeight, maxHeight),
  }
}

export function resizeChat(
  current: ChatDimensions,
  direction: -1 | 1,
  viewport: ChatDimensions
): ChatDimensions {
  return clampChatDimensions({
    width: current.width + direction * 80,
    height: current.height + direction * 120,
  }, viewport)
}

export function dragResizeChat(
  start: ChatDimensions,
  pointerDelta: ChatDimensions,
  viewport: ChatDimensions
): ChatDimensions {
  return clampChatDimensions({
    width: start.width - pointerDelta.width,
    height: start.height - pointerDelta.height,
  }, viewport)
}
