import { act, render } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { PodTerminal } from "./pod-terminal"
import { createWebSocketTicket } from "@/lib/websocket-ticket"

vi.mock("@/lib/websocket-ticket", () => ({
  createWebSocketTicket: vi.fn(),
}))

const xtermState = vi.hoisted(() => ({
  instances: [] as Array<{
    onData: ReturnType<typeof vi.fn>
    dispose: ReturnType<typeof vi.fn>
  }>,
}))

vi.mock("@xterm/xterm", () => ({
  Terminal: class {
    cols = 80
    rows = 24
    onData = vi.fn(() => ({ dispose: vi.fn() }))
    dispose = vi.fn()
    loadAddon = vi.fn()
    open = vi.fn()
    reset = vi.fn()
    writeln = vi.fn()
    write = vi.fn()

    constructor() {
      xtermState.instances.push(this)
    }
  },
}))

vi.mock("@xterm/addon-fit", () => ({
  FitAddon: class { fit = vi.fn() },
}))

vi.mock("@xterm/addon-web-links", () => ({
  WebLinksAddon: class {},
}))

class MockWebSocket {
  static readonly OPEN = 1
  static readonly CLOSED = 3
  static instances: MockWebSocket[] = []

  readyState = 0
  sent: string[] = []
  close = vi.fn(() => { this.readyState = MockWebSocket.CLOSED })
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  readonly url: string

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  send(data: string) {
    this.sent.push(data)
  }
}

class MockResizeObserver {
  observe() {}
  disconnect() {}
}

describe("PodTerminal lifecycle", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(createWebSocketTicket).mockReset().mockResolvedValue("short-lived-ticket")
    MockWebSocket.instances = []
    xtermState.instances = []
    vi.stubGlobal("WebSocket", MockWebSocket)
    vi.stubGlobal("ResizeObserver", MockResizeObserver)
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

  it("retries after a transient ticket request failure", async () => {
    vi.mocked(createWebSocketTicket)
      .mockRejectedValueOnce(new Error("temporary network failure"))
      .mockResolvedValue("retry-ticket")

    const view = render(
      <PodTerminal clusterId="1" namespace="default" podName="api" containers={["app"]} />,
    )
    await act(async () => { await Promise.resolve() })

    expect(MockWebSocket.instances).toHaveLength(0)
    expect(vi.getTimerCount()).toBe(1)

    await act(async () => {
      vi.advanceTimersByTime(3000)
      await Promise.resolve()
    })
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toContain("ticket=retry-ticket")

    view.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })

  it("disposes input subscriptions across reconnect and unmount", async () => {
    const view = render(
      <PodTerminal clusterId="1" namespace="default" podName="api" containers={["app"]} />,
    )
    const terminal = xtermState.instances[0]
    await act(async () => { await Promise.resolve() })
    const firstSocket = MockWebSocket.instances[0]
    expect(firstSocket.url).toContain("ticket=short-lived-ticket")
    expect(firstSocket.url).not.toContain("token=")

    act(() => {
      firstSocket.readyState = MockWebSocket.OPEN
      firstSocket.onopen?.()
    })
    const firstSubscription = terminal.onData.mock.results[0].value as { dispose: ReturnType<typeof vi.fn> }

    await act(async () => {
      firstSocket.readyState = MockWebSocket.CLOSED
      firstSocket.onclose?.()
      vi.advanceTimersByTime(3000)
      await Promise.resolve()
    })
    expect(firstSubscription.dispose).toHaveBeenCalledOnce()
    expect(MockWebSocket.instances).toHaveLength(2)

    const secondSocket = MockWebSocket.instances[1]
    act(() => {
      secondSocket.readyState = MockWebSocket.OPEN
      secondSocket.onopen?.()
    })
    const secondSubscription = terminal.onData.mock.results[1].value as { dispose: ReturnType<typeof vi.fn> }

    view.unmount()
    expect(secondSubscription.dispose).toHaveBeenCalledOnce()
    expect(secondSocket.close).toHaveBeenCalledOnce()
    expect(terminal.dispose).toHaveBeenCalledOnce()
    expect(vi.getTimerCount()).toBe(0)
  })
})
