/**
 * PodTerminal — interactive xterm.js terminal connected to the backend
 * WebSocket exec endpoint.
 *
 * Message protocol (JSON text frames):
 *   Browser → Server  { type: "input",  data: "<chars>" }
 *   Browser → Server  { type: "resize", cols: number, rows: number }
 *   Server  → Browser { type: "output", data: "<chars>" }
 *   Server  → Browser { type: "error",  data: "<message>" }
 *   Server  → Browser { type: "close",  data: "<reason>" }
 */

import { useEffect, useRef, useState, useCallback } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { WebLinksAddon } from "@xterm/addon-web-links"
import "@xterm/xterm/css/xterm.css"
import {
  RotateCcw,
  ChevronDown,
  Circle,
  Loader2,
  TerminalSquare,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

// ---- Types ------------------------------------------------------------------

type ConnectionStatus = "disconnected" | "connecting" | "connected" | "error"

interface TermMsg {
  type: "output" | "error" | "close"
  data: string
  cols?: number
  rows?: number
}

export interface PodTerminalProps {
  /** Numeric cluster DB id (matches the :id route param). */
  clusterId: string
  namespace: string
  podName: string
  /** List of container names in the pod. */
  containers: string[]
}

// ---- Constants --------------------------------------------------------------

const SHELLS = ["auto", "sh", "bash", "zsh"]
const RECONNECT_DELAY_MS = 3000

// ---- Component --------------------------------------------------------------

export function PodTerminal({
  clusterId,
  namespace,
  podName,
  containers,
}: PodTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const pingTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const mountedRef = useRef(true)

  const [status, setStatus] = useState<ConnectionStatus>("disconnected")
  const [container, setContainer] = useState<string>(containers[0] ?? "")
  const [shell, setShell] = useState<string>("auto")

  // ---- xterm setup ----------------------------------------------------------

  useEffect(() => {
    if (!terminalRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      fontFamily: '"Cascadia Code", "JetBrains Mono", "Fira Code", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      theme: {
        background: "#09090b",
        foreground: "#e4e4e7",
        cursor: "#a1a1aa",
        selectionBackground: "#3f3f46",
        black: "#09090b",
        red: "#ef4444",
        green: "#22c55e",
        yellow: "#eab308",
        blue: "#3b82f6",
        magenta: "#a855f7",
        cyan: "#06b6d4",
        white: "#e4e4e7",
        brightBlack: "#52525b",
        brightRed: "#f87171",
        brightGreen: "#4ade80",
        brightYellow: "#facc15",
        brightBlue: "#60a5fa",
        brightMagenta: "#c084fc",
        brightCyan: "#22d3ee",
        brightWhite: "#fafafa",
      },
    })

    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()
    term.loadAddon(fitAddon)
    term.loadAddon(webLinksAddon)
    term.open(terminalRef.current)

    // Small delay lets the container render before fitting.
    requestAnimationFrame(() => {
      fitAddon.fit()
    })

    xtermRef.current = term
    fitAddonRef.current = fitAddon

    return () => {
      term.dispose()
      xtermRef.current = null
      fitAddonRef.current = null
    }
  }, [])

  // ---- Resize observer -------------------------------------------------------

  useEffect(() => {
    if (!terminalRef.current) return

    const observer = new ResizeObserver(() => {
      if (!fitAddonRef.current || !xtermRef.current) return
      fitAddonRef.current.fit()
      sendResize()
    })
    observer.observe(terminalRef.current)
    return () => observer.disconnect()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ---- WebSocket connection --------------------------------------------------

  const disconnect = useCallback(() => {
    if (pingTimerRef.current) {
      clearInterval(pingTimerRef.current)
      pingTimerRef.current = null
    }
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    if (wsRef.current) {
      wsRef.current.onclose = null
      wsRef.current.onerror = null
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const sendResize = useCallback(() => {
    const term = xtermRef.current
    const ws = wsRef.current
    if (!term || !ws || ws.readyState !== WebSocket.OPEN) return
    const msg = { type: "resize", cols: term.cols, rows: term.rows }
    ws.send(JSON.stringify(msg))
  }, [])

  const connect = useCallback(() => {
    if (!mountedRef.current) return
    disconnect()

    setStatus("connecting")
    const term = xtermRef.current
    if (term) {
      term.reset()
      term.writeln("\x1b[90m--- Connecting to " + podName + " ---\x1b[0m")
    }

    const token = localStorage.getItem("token") ?? ""
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const params = new URLSearchParams({ token, container })
    if (shell !== "auto") {
      params.set("command", shell)
    }
    const url = `${protocol}//${window.location.host}/api/v1/clusters/${clusterId}/namespaces/${namespace}/pods/${podName}/exec?${params}`

    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      if (!mountedRef.current) return
      setStatus("connected")
      // Send initial terminal size.
      sendResize()

      // Pipe user keystrokes → WebSocket.
      if (xtermRef.current) {
        xtermRef.current.onData((data) => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: "input", data }))
          }
        })
      }

      // Keep-alive ping every 30 s.
      pingTimerRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "ping" }))
        }
      }, 30_000)
    }

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string) as TermMsg
        if (!xtermRef.current) return
        switch (msg.type) {
          case "output":
            xtermRef.current.write(msg.data)
            break
          case "error":
            xtermRef.current.writeln("\r\n\x1b[31mError: " + msg.data + "\x1b[0m")
            setStatus("error")
            break
          case "close":
            xtermRef.current.writeln("\r\n\x1b[90m--- Session ended ---\x1b[0m")
            setStatus("disconnected")
            break
        }
      } catch {
        // Binary or non-JSON frame — ignore.
      }
    }

    ws.onerror = () => {
      if (!mountedRef.current) return
      setStatus("error")
    }

    ws.onclose = () => {
      if (!mountedRef.current) return
      if (pingTimerRef.current) {
        clearInterval(pingTimerRef.current)
        pingTimerRef.current = null
      }
      if (status !== "disconnected") {
        setStatus("disconnected")
        if (xtermRef.current) {
          xtermRef.current.writeln("\r\n\x1b[90m--- Disconnected. Reconnecting in 3s... ---\x1b[0m")
        }
        reconnectTimerRef.current = setTimeout(() => {
          if (mountedRef.current) connect()
        }, RECONNECT_DELAY_MS)
      }
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace, podName, container, shell, disconnect, sendResize])

  // Auto-connect on mount and whenever the key params change.
  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [connect, disconnect])

  // ---- Render ---------------------------------------------------------------

  const statusConfig: Record<ConnectionStatus, { label: string; color: string }> = {
    connected:    { label: "Connected",    color: "text-green-500" },
    connecting:   { label: "Connecting",   color: "text-yellow-500" },
    disconnected: { label: "Disconnected", color: "text-muted-foreground" },
    error:        { label: "Error",        color: "text-red-500" },
  }
  const { label: statusLabel, color: statusColor } = statusConfig[status]

  return (
    <div className="flex flex-col gap-3 h-full min-h-0">
      {/* Toolbar */}
      <div className="flex items-center gap-2 flex-wrap">
        {/* Container selector */}
        {containers.length > 1 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="outline" size="sm" className="h-7 gap-1 text-xs">
                <TerminalSquare className="size-3" />
                {container || "Container"}
                <ChevronDown className="size-3 ml-1" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start">
              <DropdownMenuLabel className="text-xs">Container</DropdownMenuLabel>
              <DropdownMenuSeparator />
              {containers.map((c) => (
                <DropdownMenuItem
                  key={c}
                  onClick={() => setContainer(c)}
                  className="text-xs"
                >
                  {c}
                  {c === container && <span className="ml-auto text-primary">active</span>}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}

        {/* Shell selector */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs">
              {shell}
              <ChevronDown className="size-3 ml-1" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel className="text-xs">Shell</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {SHELLS.map((s) => (
              <DropdownMenuItem
                key={s}
                onClick={() => setShell(s)}
                className="text-xs font-mono"
              >
                {s}
                {s === shell && <span className="ml-auto text-primary">active</span>}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Reconnect */}
        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={connect}
          disabled={status === "connecting"}
        >
          {status === "connecting" ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <RotateCcw className="size-3" />
          )}
          Reconnect
        </Button>

        {/* Status indicator */}
        <div className={`ml-auto flex items-center gap-1.5 text-xs ${statusColor}`}>
          <Circle className="size-2 fill-current" />
          {statusLabel}
        </div>
      </div>

      {/* Terminal viewport */}
      <div
        ref={terminalRef}
        className="flex-1 min-h-0 rounded-md overflow-hidden bg-[#09090b] border border-border"
        style={{ padding: "4px" }}
      />
    </div>
  )
}
