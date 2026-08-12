import { beforeEach, describe, expect, it } from "vitest"
import { readSidebarOpen } from "./sidebar-state-preference"

describe("sidebar state preference", () => {
  beforeEach(() => {
    document.cookie = "sidebar_state=; path=/; max-age=0"
  })

  it("defaults to expanded without a saved cookie", () => {
    expect(readSidebarOpen()).toBe(true)
  })

  it("restores the collapsed state from the sidebar cookie", () => {
    document.cookie = "sidebar_state=false; path=/"

    expect(readSidebarOpen()).toBe(false)
  })

  it("restores the expanded state among other cookies", () => {
    document.cookie = "unrelated=value; path=/"
    document.cookie = "sidebar_state=true; path=/"

    expect(readSidebarOpen()).toBe(true)
  })
})
