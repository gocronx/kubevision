/**
 * PodLogs — real-time pod log viewer connected to the backend WebSocket
 * logs endpoint.
 *
 * Message protocol (JSON text frames):
 *   Server → Browser { type: "log",   data: "<line>" }
 *   Server → Browser { type: "error", data: "<message>" }
 *   Server → Browser { type: "close", data: "<reason>" }
 */

import {
  useEffect,
  useRef,
  useState,
  useCallback,
  useMemo,
} from "react"
import {
  Play,
  Pause,
  RotateCcw,
  Download,
  Search,
  Clock,
  ChevronDown,
  Circle,
  Loader2,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { createWebSocketTicket } from "@/lib/websocket-ticket"
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

interface LogMsg {
  type: "log" | "error" | "close"
  data: string
}

interface LogLine {
  id: number
  text: string
}

const TAIL_OPTIONS = [
  { label: "Last 100 lines",  value: "100" },
  { label: "Last 500 lines",  value: "500" },
  { label: "Last 1000 lines", value: "1000" },
  { label: "All lines",       value: "" },
]

const MAX_RETAINED_LOG_LINES = 10_000

export interface PodLogsProps {
  /** Numeric cluster DB id. */
  clusterId: string
  namespace: string
  podName: string
  containers: string[]
}

// ---- Component --------------------------------------------------------------

let lineCounter = 0

export function PodLogs({
  clusterId,
  namespace,
  podName,
  containers,
}: PodLogsProps) {
  const wsRef = useRef<WebSocket | null>(null)
  const ticketAbortRef = useRef<AbortController | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)
  const mountedRef = useRef(true)

  const [lines, setLines] = useState<LogLine[]>([])
  const [status, setStatus] = useState<ConnectionStatus>("disconnected")
  const [container, setContainer] = useState<string>(containers[0] ?? "")
  const [follow, setFollow] = useState(true)
  const [showTimestamps, setShowTimestamps] = useState(false)
  const [tailOption, setTailOption] = useState("100")
  const [searchQuery, setSearchQuery] = useState("")

  // ---- Filtered lines -------------------------------------------------------

  const filteredLines = useMemo(() => {
    if (!searchQuery.trim()) return lines
    const q = searchQuery.toLowerCase()
    return lines.filter((l) => l.text.toLowerCase().includes(q))
  }, [lines, searchQuery])

  // ---- Auto-scroll ----------------------------------------------------------

  useEffect(() => {
    if (!follow || !scrollRef.current) return
    scrollRef.current.scrollTop = scrollRef.current.scrollHeight
  }, [filteredLines, follow])

  // ---- WebSocket connection --------------------------------------------------

  const disconnect = useCallback(() => {
    ticketAbortRef.current?.abort()
    ticketAbortRef.current = null
    if (wsRef.current) {
      wsRef.current.onopen = null
      wsRef.current.onmessage = null
      wsRef.current.onclose = null
      wsRef.current.onerror = null
      wsRef.current.close()
      wsRef.current = null
    }
  }, [])

  const connect = useCallback(async () => {
    if (!mountedRef.current) return
    disconnect()

    setLines([])
    setStatus("connecting")

    const controller = new AbortController()
    ticketAbortRef.current = controller
    try {
      const ticket = await createWebSocketTicket(controller.signal)
      if (!mountedRef.current || controller.signal.aborted) return
      ticketAbortRef.current = null

      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
      const params = new URLSearchParams({
        ticket,
        container,
        follow: follow ? "true" : "false",
        timestamps: showTimestamps ? "true" : "false",
      })
      if (tailOption) params.set("tailLines", tailOption)

      const url = `${protocol}//${window.location.host}/api/v1/clusters/${clusterId}/namespaces/${namespace}/pods/${podName}/logs?${params}`
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        if (!mountedRef.current || wsRef.current !== ws) return
        setStatus("connected")
      }

      ws.onmessage = (ev) => {
        if (!mountedRef.current || wsRef.current !== ws) return
        try {
          const msg = JSON.parse(ev.data as string) as LogMsg
          switch (msg.type) {
            case "log":
              setLines((prev) => {
                const next = [...prev, { id: ++lineCounter, text: msg.data }]
                return next.length > MAX_RETAINED_LOG_LINES
                  ? next.slice(next.length - MAX_RETAINED_LOG_LINES)
                  : next
              })
              break
            case "error":
              setLines((prev) => [
                ...prev,
                { id: ++lineCounter, text: `[ERROR] ${msg.data}` },
              ])
              setStatus("error")
              break
            case "close":
              setStatus("disconnected")
              break
          }
        } catch {
          // Non-JSON frame — ignore.
        }
      }

      ws.onerror = () => {
        if (!mountedRef.current || wsRef.current !== ws) return
        setStatus("error")
      }

      ws.onclose = () => {
        if (!mountedRef.current || wsRef.current !== ws) return
        wsRef.current = null
        setStatus("disconnected")
      }
    } catch {
      if (mountedRef.current && !controller.signal.aborted) setStatus("error")
    }
  }, [clusterId, namespace, podName, container, follow, showTimestamps, tailOption, disconnect])

  useEffect(() => {
    mountedRef.current = true
    connect()
    return () => {
      mountedRef.current = false
      disconnect()
    }
  }, [connect, disconnect])

  // ---- Follow toggle --------------------------------------------------------

  const toggleFollow = useCallback(() => {
    setFollow((prev) => !prev)
  }, [])

  // ---- Download logs --------------------------------------------------------

  const handleDownload = useCallback(() => {
    const text = lines.map((l) => l.text).join("\n")
    const blob = new Blob([text], { type: "text/plain" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = `${podName}${container ? "-" + container : ""}.log`
    a.click()
    URL.revokeObjectURL(url)
  }, [lines, podName, container])

  // ---- Render ---------------------------------------------------------------

  const statusConfig: Record<ConnectionStatus, { label: string; color: string }> = {
    connected:    { label: "Streaming",    color: "text-green-500" },
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

        {/* Tail lines selector */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm" className="h-7 gap-1 text-xs">
              {TAIL_OPTIONS.find((o) => o.value === tailOption)?.label ?? "Lines"}
              <ChevronDown className="size-3 ml-1" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel className="text-xs">Tail lines</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {TAIL_OPTIONS.map((o) => (
              <DropdownMenuItem
                key={o.value}
                onClick={() => setTailOption(o.value)}
                className="text-xs"
              >
                {o.label}
                {o.value === tailOption && <span className="ml-auto text-primary">active</span>}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        {/* Timestamps toggle */}
        <Button
          variant={showTimestamps ? "default" : "outline"}
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={() => setShowTimestamps((v) => !v)}
        >
          <Clock className="size-3" />
          Timestamps
        </Button>

        {/* Follow / Pause */}
        <Button
          variant={follow ? "default" : "outline"}
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={toggleFollow}
        >
          {follow ? (
            <>
              <Pause className="size-3" />
              Pause
            </>
          ) : (
            <>
              <Play className="size-3" />
              Follow
            </>
          )}
        </Button>

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
          Reload
        </Button>

        {/* Download */}
        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={handleDownload}
          disabled={lines.length === 0}
        >
          <Download className="size-3" />
          Download
        </Button>

        {/* Status indicator */}
        <div className={`flex items-center gap-1.5 text-xs ${statusColor}`}>
          {status === "connecting" ? (
            <Loader2 className="size-3 animate-spin" />
          ) : (
            <Circle className="size-2 fill-current" />
          )}
          {statusLabel}
        </div>
      </div>

      {/* Search bar */}
      <div className="relative">
        <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
        <Input
          className="pl-7 h-7 text-xs font-mono"
          placeholder="Search logs..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      {/* Log lines */}
      <div
        ref={scrollRef}
        className="flex-1 min-h-0 overflow-auto rounded-md border border-border bg-[#09090b] p-3"
      >
        {filteredLines.length === 0 ? (
          <p className="text-xs text-muted-foreground italic">
            {status === "connecting" ? "Connecting…" : "No log lines to display."}
          </p>
        ) : (
          <pre className="text-xs font-mono leading-5 whitespace-pre-wrap break-all text-zinc-300">
            {filteredLines.map((line) => (
              <LogLine key={line.id} text={line.text} search={searchQuery} />
            ))}
          </pre>
        )}
      </div>

      {/* Line count */}
      <p className="text-xs text-muted-foreground text-right">
        {filteredLines.length.toLocaleString()}
        {searchQuery ? ` / ${lines.length.toLocaleString()}` : ""} lines
      </p>
    </div>
  )
}

// ---- LogLine — highlights search matches ------------------------------------

function LogLine({ text, search }: { text: string; search: string }) {
  if (!search.trim()) {
    return <span className="block">{text + "\n"}</span>
  }

  const lower = text.toLowerCase()
  const q = search.toLowerCase()
  const parts: React.ReactNode[] = []
  let lastIdx = 0
  let idx = lower.indexOf(q)

  while (idx !== -1) {
    if (idx > lastIdx) {
      parts.push(text.slice(lastIdx, idx))
    }
    parts.push(
      <mark key={idx} className="bg-yellow-400/30 text-yellow-200 rounded-sm px-px">
        {text.slice(idx, idx + q.length)}
      </mark>
    )
    lastIdx = idx + q.length
    idx = lower.indexOf(q, lastIdx)
  }

  if (lastIdx < text.length) {
    parts.push(text.slice(lastIdx))
  }

  return <span className="block">{parts}{"\n"}</span>
}
