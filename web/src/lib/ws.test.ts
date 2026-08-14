import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { WebSocketClient } from "./ws"

class MockWebSocket {
  static readonly OPEN = 1
  static readonly CLOSED = 3
  static instances: MockWebSocket[] = []

  readonly url: string
  readyState = 0
  sent: string[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  open() {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.()
  }

  send(data: string) {
    this.sent.push(data)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }
}

describe("WebSocketClient lifecycle", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    MockWebSocket.instances = []
    vi.stubGlobal("WebSocket", MockWebSocket)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it("keeps one reconnect timer and reconnects after an unexpected close", () => {
    const client = new WebSocketClient()
    client.connect("ws://example.test/watch")
    const first = MockWebSocket.instances[0]
    first.open()

    first.close()
    expect(vi.getTimerCount()).toBe(1)

    vi.advanceTimersByTime(999)
    expect(MockWebSocket.instances).toHaveLength(1)
    vi.advanceTimersByTime(1)
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(vi.getTimerCount()).toBe(0)

    client.disconnect()
    expect(vi.getTimerCount()).toBe(0)
  })

  it("does not reconnect after disconnect or when replacing a connection", () => {
    const client = new WebSocketClient()
    client.connect("ws://example.test/watch")
    client.connect("ws://example.test/watch")

    expect(MockWebSocket.instances).toHaveLength(2)
    expect(vi.getTimerCount()).toBe(0)

    const current = MockWebSocket.instances[1]
    current.open()
    vi.advanceTimersByTime(30_000)
    expect(current.sent).toContain(JSON.stringify({ action: "ping" }))

    client.disconnect()
    vi.runAllTimers()
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(vi.getTimerCount()).toBe(0)
  })
})
