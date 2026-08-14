/** ResourceEvent matches the backend informer.ResourceEvent JSON structure. */
export interface ResourceEvent {
  type: "ADDED" | "MODIFIED" | "DELETED"
  clusterId: string
  resource: string
  namespace: string
  name: string
  object?: Record<string, unknown>
}

type EventHandler = (event: ResourceEvent) => void

/**
 * WebSocketClient connects to the KubeVision WebSocket endpoint and handles
 * topic-based subscription, heartbeat, and exponential backoff reconnection.
 *
 * Protocol:
 * - Send: {"action":"subscribe","topics":["cluster1:pods"]}
 * - Send: {"action":"unsubscribe","topics":["cluster1:pods"]}
 * - Receive: ResourceEvent JSON per line
 */
export class WebSocketClient {
  private ws: WebSocket | null = null
  private url = ""
  private reconnectAttempts = 0
  private maxReconnectAttempts = 10
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private handlers = new Set<EventHandler>()
  private subscribedTopics = new Set<string>()
  private connected = false
  private shouldReconnect = false

  connect(url: string) {
    this.url = url
    this.shouldReconnect = true
    this.reconnectAttempts = 0
    this.clearReconnectTimer()
    this.doConnect()
  }

  private doConnect() {
    this.clearReconnectTimer()
    this.closeSocket()

    const token = localStorage.getItem("token")
    const wsUrl = token ? `${this.url}?token=${token}` : this.url
    const socket = new WebSocket(wsUrl)
    this.ws = socket

    socket.onopen = () => {
      if (this.ws !== socket) return
      this.connected = true
      this.reconnectAttempts = 0
      this.startHeartbeat()

      // Re-subscribe to any previously subscribed topics after reconnect.
      if (this.subscribedTopics.size > 0) {
        this.sendSubscribe([...this.subscribedTopics])
      }
    }

    socket.onmessage = (event: MessageEvent) => {
      if (this.ws !== socket) return
      try {
        // Backend may batch multiple events separated by newlines.
        const lines = (event.data as string).split("\n")
        for (const line of lines) {
          if (!line.trim()) continue
          const resourceEvent = JSON.parse(line) as ResourceEvent
          this.handlers.forEach((handler) => handler(resourceEvent))
        }
      } catch {
        // ignore malformed messages
      }
    }

    socket.onclose = () => {
      if (this.ws !== socket) return
      this.ws = null
      this.connected = false
      this.stopHeartbeat()
      if (this.shouldReconnect) this.scheduleReconnect()
    }

    socket.onerror = () => {
      socket.close()
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ action: "ping" }))
      }
    }, 30000)
  }

  private stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }

  private scheduleReconnect() {
    if (!this.shouldReconnect || this.reconnectAttempts >= this.maxReconnectAttempts) {
      return
    }
    this.clearReconnectTimer()
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    this.reconnectAttempts++
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      if (!this.shouldReconnect) return
      this.doConnect()
    }, delay)
  }

  private clearReconnectTimer() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
  }

  private closeSocket() {
    if (!this.ws) return
    const socket = this.ws
    this.ws = null
    socket.onopen = null
    socket.onmessage = null
    socket.onclose = null
    socket.onerror = null
    socket.close()
  }

  private sendSubscribe(topics: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: "subscribe", topics }))
    }
  }

  private sendUnsubscribe(topics: string[]) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: "unsubscribe", topics }))
    }
  }

  /** Subscribe to resource events for a specific cluster + resource type. */
  subscribe(clusterId: string, resource: string) {
    const topic = `${clusterId}:${resource}`
    if (!this.subscribedTopics.has(topic)) {
      this.subscribedTopics.add(topic)
      this.sendSubscribe([topic])
    }
  }

  /** Unsubscribe from resource events for a specific cluster + resource type. */
  unsubscribe(clusterId: string, resource: string) {
    const topic = `${clusterId}:${resource}`
    if (this.subscribedTopics.has(topic)) {
      this.subscribedTopics.delete(topic)
      this.sendUnsubscribe([topic])
    }
  }

  /** Register a handler to receive all resource events. */
  onEvent(handler: EventHandler) {
    this.handlers.add(handler)
  }

  /** Remove a previously registered event handler. */
  offEvent(handler: EventHandler) {
    this.handlers.delete(handler)
  }

  disconnect() {
    this.shouldReconnect = false
    this.stopHeartbeat()
    this.clearReconnectTimer()
    this.reconnectAttempts = 0
    this.closeSocket()
    this.connected = false
    this.subscribedTopics.clear()
    this.handlers.clear()
  }

  get isConnected() {
    return this.connected
  }
}

export const wsClient = new WebSocketClient()
