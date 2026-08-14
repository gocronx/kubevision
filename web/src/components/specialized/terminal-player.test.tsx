import { fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { TerminalPlayer } from "./terminal-player"

const xtermState = vi.hoisted(() => ({
  instances: [] as Array<{ dispose: ReturnType<typeof vi.fn> }>,
}))

vi.mock("@xterm/xterm", () => ({
  Terminal: class {
    dispose = vi.fn()
    loadAddon = vi.fn()
    open = vi.fn()
    reset = vi.fn()
    write = vi.fn()

    constructor() {
      xtermState.instances.push(this)
    }
  },
}))

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class { fit = vi.fn() },
}))

describe("TerminalPlayer lifecycle", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    xtermState.instances = []
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      callback(0)
      return 1
    })
    vi.stubGlobal("cancelAnimationFrame", vi.fn())
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it("clears restart timers and disposes the terminal on unmount", () => {
    const recording = [
      JSON.stringify({ version: 2, width: 80, height: 24 }),
      JSON.stringify([1, "o", "ready"]),
    ].join("\n")
    const view = render(<TerminalPlayer recording={recording} />)

    fireEvent.click(screen.getByRole("button", { name: /restart/i }))
    expect(vi.getTimerCount()).toBe(1)

    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
    expect(xtermState.instances[0].dispose).toHaveBeenCalledOnce()
  })
})
