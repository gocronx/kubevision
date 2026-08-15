import { act, fireEvent, render, screen, waitFor } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { PodLogs } from "./pod-logs"
import { createWebSocketTicket } from "@/lib/websocket-ticket"

vi.mock("@/lib/websocket-ticket", () => ({
  createWebSocketTicket: vi.fn(),
}))

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
    vi.mocked(createWebSocketTicket).mockResolvedValue("short-lived-ticket")
    MockWebSocket.instances = []
    vi.stubGlobal("WebSocket", MockWebSocket)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("cancels an unfinished ticket request on unmount", async () => {
    let signal: AbortSignal | undefined
    vi.mocked(createWebSocketTicket).mockImplementation((nextSignal) => {
      signal = nextSignal
      return new Promise<string>(() => {})
    })

    const view = render(
      <PodLogs clusterId="1" namespace="default" podName="api" containers={["app"]} />,
    )
    await waitFor(() => expect(signal).toBeDefined())

    view.unmount()
    expect(signal?.aborted).toBe(true)
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  it("reconnects with a short-lived ticket and closes on unmount", async () => {
    const view = render(
      <PodLogs clusterId="1" namespace="default" podName="api" containers={["app"]} />,
    )

    await waitFor(() => expect(MockWebSocket.instances).toHaveLength(1))
    expect(MockWebSocket.instances[0].url).toContain("follow=true")
    expect(MockWebSocket.instances[0].url).toContain("ticket=short-lived-ticket")
    expect(MockWebSocket.instances[0].url).not.toContain("token=")

    act(() => {
      MockWebSocket.instances[0].onopen?.()
    })
    fireEvent.click(screen.getByRole("button", { name: /pause/i }))

    expect(MockWebSocket.instances[0].close).toHaveBeenCalledOnce()
    await waitFor(() => expect(MockWebSocket.instances).toHaveLength(2))
    expect(MockWebSocket.instances[1].url).toContain("follow=false")

    view.unmount()
    expect(MockWebSocket.instances[1].close).toHaveBeenCalledOnce()
  })
})
