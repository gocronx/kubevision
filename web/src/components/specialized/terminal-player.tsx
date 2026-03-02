/**
 * TerminalPlayer — replays asciinema v2 recordings using xterm.js.
 *
 * Asciinema v2 format:
 *   Line 1: JSON header  { "version": 2, "width": N, "height": N, ... }
 *   Line N: JSON event   [elapsed_seconds, "o", "data"]
 */

import { useEffect, useRef, useState, useCallback } from "react"
import { Terminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import "@xterm/xterm/css/xterm.css"
import { Play, Pause, RotateCcw, Loader2 } from "lucide-react"
import { Button } from "@/components/ui/button"

// --------------------------------------------------------------------------
// Asciinema v2 parsing
// --------------------------------------------------------------------------

interface AsciinemaHeader {
  version: number
  width: number
  height: number
  timestamp?: number
  title?: string
}

interface AsciinemaOutputEvent {
  elapsed: number  // seconds from session start
  data: string
}

interface ParsedRecording {
  header: AsciinemaHeader
  events: AsciinemaOutputEvent[]
  duration: number  // total seconds
}

function parseAsciinema(raw: string): ParsedRecording {
  const lines = raw.trim().split("\n")
  if (lines.length === 0) {
    return { header: { version: 2, width: 220, height: 50 }, events: [], duration: 0 }
  }

  let header: AsciinemaHeader = { version: 2, width: 220, height: 50 }
  try {
    header = JSON.parse(lines[0]) as AsciinemaHeader
  } catch {
    // If the first line fails to parse as header, use defaults.
  }

  const events: AsciinemaOutputEvent[] = []
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line) continue
    try {
      const tuple = JSON.parse(line) as [number, string, string]
      if (Array.isArray(tuple) && tuple.length >= 3 && tuple[1] === "o") {
        events.push({ elapsed: tuple[0], data: tuple[2] })
      }
    } catch {
      // Skip malformed event lines.
    }
  }

  const duration = events.length > 0 ? events[events.length - 1].elapsed : 0
  return { header, events, duration }
}

// --------------------------------------------------------------------------
// Props
// --------------------------------------------------------------------------

export interface TerminalPlayerProps {
  recording: string
  durationMs?: number
}

const SPEED_OPTIONS: { label: string; value: number }[] = [
  { label: "0.5x", value: 0.5 },
  { label: "1x",   value: 1 },
  { label: "2x",   value: 2 },
  { label: "4x",   value: 4 },
]

// --------------------------------------------------------------------------
// Component
// --------------------------------------------------------------------------

export function TerminalPlayer({ recording, durationMs }: TerminalPlayerProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<Terminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const playbackRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const parsedRef = useRef<ParsedRecording | null>(null)
  const eventIndexRef = useRef(0)
  const playbackStartWallRef = useRef(0)
  const playbackStartElapsedRef = useRef(0)

  const [isPlaying, setIsPlaying] = useState(false)
  const [speed, setSpeed] = useState(1)
  const [elapsed, setElapsed] = useState(0)
  const [duration, setDuration] = useState(durationMs ? durationMs / 1000 : 0)

  // ---- xterm setup ----------------------------------------------------------

  useEffect(() => {
    if (!terminalRef.current) return

    const parsed = parseAsciinema(recording)
    parsedRef.current = parsed
    setDuration(parsed.duration || (durationMs ? durationMs / 1000 : 0))
    setElapsed(0)
    eventIndexRef.current = 0

    const term = new Terminal({
      cursorBlink: false,
      fontFamily: '"Cascadia Code", "JetBrains Mono", "Fira Code", monospace',
      fontSize: 12,
      lineHeight: 1.2,
      cols: parsed.header.width || 220,
      rows: Math.min(parsed.header.height || 50, 50),
      theme: {
        background: "#09090b",
        foreground: "#e4e4e7",
        cursor: "#a1a1aa",
        selectionBackground: "#3f3f46",
      },
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(terminalRef.current)
    requestAnimationFrame(() => fitAddon.fit())

    xtermRef.current = term
    fitAddonRef.current = fitAddon

    return () => {
      stopPlayback()
      term.dispose()
      xtermRef.current = null
      fitAddonRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recording])

  // ---- Playback engine -------------------------------------------------------

  const stopPlayback = useCallback(() => {
    if (playbackRef.current !== null) {
      clearTimeout(playbackRef.current)
      playbackRef.current = null
    }
    setIsPlaying(false)
  }, [])

  const scheduleNextEvent = useCallback((currentSpeed: number, currentElapsed: number, startWall: number, startElapsed: number) => {
    const parsed = parsedRef.current
    const term = xtermRef.current
    if (!parsed || !term) return

    const idx = eventIndexRef.current
    if (idx >= parsed.events.length) {
      setIsPlaying(false)
      return
    }

    const event = parsed.events[idx]
    const wallNow = Date.now()
    const recordingNow = startElapsed + ((wallNow - startWall) / 1000) * currentSpeed
    const delay = Math.max(0, ((event.elapsed - recordingNow) / currentSpeed) * 1000)

    playbackRef.current = setTimeout(() => {
      if (!xtermRef.current) return
      xtermRef.current.write(event.data)
      setElapsed(event.elapsed)
      eventIndexRef.current += 1
      scheduleNextEvent(currentSpeed, currentElapsed, startWall, startElapsed)
    }, delay)
  }, [])

  const startPlayback = useCallback(() => {
    const parsed = parsedRef.current
    if (!parsed) return

    const startWall = Date.now()
    const startElapsed = elapsed
    playbackStartWallRef.current = startWall
    playbackStartElapsedRef.current = startElapsed

    // Find event index at the current position.
    let startIdx = parsed.events.length
    for (let i = 0; i < parsed.events.length; i++) {
      if (parsed.events[i].elapsed >= startElapsed) {
        startIdx = i
        break
      }
    }
    eventIndexRef.current = startIdx
    setIsPlaying(true)
    scheduleNextEvent(speed, startElapsed, startWall, startElapsed)
  }, [elapsed, speed, scheduleNextEvent])

  const handlePlayPause = useCallback(() => {
    if (isPlaying) {
      stopPlayback()
    } else {
      startPlayback()
    }
  }, [isPlaying, stopPlayback, startPlayback])

  const handleRestart = useCallback(() => {
    stopPlayback()
    const term = xtermRef.current
    if (term) term.reset()
    setElapsed(0)
    eventIndexRef.current = 0

    setTimeout(() => {
      const startWall = Date.now()
      playbackStartWallRef.current = startWall
      playbackStartElapsedRef.current = 0
      setIsPlaying(true)
      scheduleNextEvent(speed, 0, startWall, 0)
    }, 50)
  }, [stopPlayback, speed, scheduleNextEvent])

  const handleSeek = useCallback((seconds: number) => {
    const parsed = parsedRef.current
    const term = xtermRef.current
    if (!parsed || !term) return

    stopPlayback()
    term.reset()

    for (const ev of parsed.events) {
      if (ev.elapsed <= seconds) {
        term.write(ev.data)
      }
    }
    setElapsed(seconds)
  }, [stopPlayback])

  // ---- Helpers ---------------------------------------------------------------

  function formatTime(sec: number): string {
    const m = Math.floor(sec / 60)
    const s = Math.floor(sec % 60)
    return `${m}:${s.toString().padStart(2, "0")}`
  }

  const progressPct = duration > 0 ? (elapsed / duration) * 100 : 0

  // ---- Render ---------------------------------------------------------------

  return (
    <div className="flex flex-col gap-3 h-full min-h-0">
      {/* Controls */}
      <div className="flex items-center gap-2 flex-wrap">
        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={handlePlayPause}
        >
          {isPlaying ? (
            <Pause className="size-3" />
          ) : (
            <Play className="size-3" />
          )}
          {isPlaying ? "Pause" : "Play"}
        </Button>

        <Button
          variant="outline"
          size="sm"
          className="h-7 gap-1 text-xs"
          onClick={handleRestart}
        >
          <RotateCcw className="size-3" />
          Restart
        </Button>

        {/* Speed selector — plain HTML select for simplicity */}
        <select
          value={String(speed)}
          onChange={(e) => setSpeed(Number(e.target.value))}
          className="h-7 rounded-md border border-input bg-background px-2 text-xs text-foreground"
        >
          {SPEED_OPTIONS.map((opt) => (
            <option key={opt.value} value={String(opt.value)}>
              {opt.label}
            </option>
          ))}
        </select>

        {/* Timestamp */}
        <span className="ml-auto font-mono text-xs text-muted-foreground tabular-nums">
          {formatTime(elapsed)} / {formatTime(duration)}
        </span>
      </div>

      {/* Progress bar — native range input for seek */}
      {duration > 0 && (
        <div className="relative w-full">
          <input
            type="range"
            min={0}
            max={duration}
            step={0.1}
            value={elapsed}
            onChange={(e) => handleSeek(Number(e.target.value))}
            className="w-full accent-primary h-1.5"
          />
          {/* Visual fill */}
          <div
            className="pointer-events-none absolute top-0 left-0 h-1.5 rounded-full bg-primary/30"
            style={{ width: `${progressPct}%` }}
          />
        </div>
      )}

      {/* Terminal viewport */}
      <div
        ref={terminalRef}
        className="flex-1 min-h-0 rounded-md overflow-hidden bg-[#09090b] border border-border"
        style={{ padding: "4px" }}
      />
    </div>
  )
}

export function TerminalPlayerLoading() {
  return (
    <div className="flex items-center justify-center h-64 text-muted-foreground">
      <Loader2 className="size-6 animate-spin mr-2" />
      Loading recording...
    </div>
  )
}
