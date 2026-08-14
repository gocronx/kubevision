import { act, fireEvent, render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { PodLogs } from "./pod-logs"

class MockWebSocket {
  static instances: MockWebSocket[] = []

  readonly url: string
  close = vi.fn()
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
}

describe("PodLogs lifecycle", () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal("WebSocket", MockWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("reconnects with the current follow value and closes on unmount", () => {
    const view = render(
      <PodLogs clusterId="1" namespace="default" podName="api" containers={["app"]} />,
    )

    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toContain("follow=true")

    act(() => {
      MockWebSocket.instances[0].onopen?.()
    })
    fireEvent.click(screen.getByRole("button", { name: /pause/i }))

    expect(MockWebSocket.instances[0].close).toHaveBeenCalledOnce()
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(MockWebSocket.instances[1].url).toContain("follow=false")

    view.unmount()
    expect(MockWebSocket.instances[1].close).toHaveBeenCalledOnce()
  })
})
