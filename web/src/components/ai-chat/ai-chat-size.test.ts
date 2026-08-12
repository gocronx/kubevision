import { describe, expect, it } from "vitest"
import {
  CHAT_DEFAULT_DIMENSIONS,
  clampChatDimensions,
  dragResizeChat,
  resizeChat,
} from "./ai-chat-size"

const desktopViewport = { width: 1440, height: 1000 }

describe("chat dimensions", () => {
  it("steps the window larger and smaller", () => {
    expect(resizeChat(CHAT_DEFAULT_DIMENSIONS, -1, desktopViewport)).toEqual({ width: 320, height: 480 })
    expect(resizeChat(CHAT_DEFAULT_DIMENSIONS, 1, desktopViewport)).toEqual({ width: 480, height: 720 })
  })

  it("resizes continuously from the top-left drag handle", () => {
    expect(dragResizeChat(CHAT_DEFAULT_DIMENSIONS, { width: -57, height: -83 }, desktopViewport))
      .toEqual({ width: 457, height: 683 })
    expect(dragResizeChat(CHAT_DEFAULT_DIMENSIONS, { width: 45, height: 70 }, desktopViewport))
      .toEqual({ width: 355, height: 530 })
  })

  it("stays within minimum, maximum, and viewport bounds", () => {
    expect(dragResizeChat(CHAT_DEFAULT_DIMENSIONS, { width: 500, height: 500 }, desktopViewport))
      .toEqual({ width: 320, height: 360 })
    expect(dragResizeChat(CHAT_DEFAULT_DIMENSIONS, { width: -2000, height: -2000 }, desktopViewport))
      .toEqual({ width: 900, height: 900 })
    expect(clampChatDimensions({ width: 800, height: 800 }, { width: 375, height: 667 }))
      .toEqual({ width: 343, height: 635 })
  })
})
