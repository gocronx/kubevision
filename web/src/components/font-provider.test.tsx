import { act, render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { FontProvider, useFont } from "./font-provider"

function FontHarness() {
  const { font, setFont } = useFont()
  return <button onClick={() => setFont("inter")}>{font}</button>
}

describe("FontProvider", () => {
  it("uses the system font by default", () => {
    render(<FontProvider><FontHarness /></FontProvider>)

    expect(screen.getByRole("button", { name: "system" })).toBeInTheDocument()
    expect(document.documentElement.style.getPropertyValue("--app-font-sans")).toContain("system-ui")
  })

  it("keeps an explicitly selected font", () => {
    localStorage.setItem("kubevision-font", "jetbrains-mono")
    render(<FontProvider><FontHarness /></FontProvider>)

    expect(screen.getByRole("button", { name: "jetbrains-mono" })).toBeInTheDocument()
    act(() => screen.getByRole("button").click())
    expect(localStorage.getItem("kubevision-font")).toBe("inter")
  })
})
