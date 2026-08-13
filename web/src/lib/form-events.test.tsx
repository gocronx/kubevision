import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { preventSubmitWhileComposing } from "./form-events"

function FormHarness() {
  return (
    <form onKeyDownCapture={preventSubmitWhileComposing}>
      <input aria-label="name" />
    </form>
  )
}

describe("preventSubmitWhileComposing", () => {
  it("prevents an IME confirmation Enter", () => {
    render(<FormHarness />)
    const input = screen.getByRole("textbox", { name: "name" })

    expect(fireEvent.keyDown(input, { key: "Enter", isComposing: true })).toBe(false)
  })

  it("prevents the keyCode 229 IME fallback", () => {
    render(<FormHarness />)
    const input = screen.getByRole("textbox", { name: "name" })

    expect(fireEvent.keyDown(input, { key: "Enter", keyCode: 229 })).toBe(false)
  })

  it("allows an ordinary Enter", () => {
    const preventDefault = vi.fn()
    preventSubmitWhileComposing({
      key: "Enter",
      nativeEvent: { isComposing: false, keyCode: 13 },
      preventDefault,
    } as never)

    expect(preventDefault).not.toHaveBeenCalled()
  })
})
