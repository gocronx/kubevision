import { describe, expect, it, vi } from "vitest"
import { readFavoritesPanelOpen, writeFavoritesPanelOpen } from "./favorites-panel-preference"

describe("favorites panel preference", () => {
  it("defaults to expanded when the user has no saved preference", () => {
    expect(readFavoritesPanelOpen(7)).toBe(true)
  })

  it("restores a collapsed panel after a refresh", () => {
    writeFavoritesPanelOpen(7, false)

    expect(readFavoritesPanelOpen(7)).toBe(false)
  })

  it("keeps each user's preference separate", () => {
    writeFavoritesPanelOpen(7, false)
    writeFavoritesPanelOpen(8, true)

    expect(readFavoritesPanelOpen(7)).toBe(false)
    expect(readFavoritesPanelOpen(8)).toBe(true)
  })

  it("falls back to expanded when storage is unavailable", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementationOnce(() => {
      throw new Error("storage unavailable")
    })

    expect(readFavoritesPanelOpen(7)).toBe(true)
  })
})
