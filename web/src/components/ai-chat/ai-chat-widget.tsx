import { useRef, useState, type KeyboardEvent, type PointerEvent as ReactPointerEvent } from "react"
import { Link, useLocation, useParams, useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Bot, MoveDiagonal2, Settings, X, ZoomIn, ZoomOut } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { useCluster } from "@/hooks/use-cluster"
import { useAuth } from "@/stores/auth-store"
import { useAIChat } from "./use-ai-chat"
import { useAIConfig } from "./use-ai-config"
import { ChatMessages } from "./ai-chat-messages"
import { ChatComposer } from "./ai-chat-composer"
import { ChatSessions } from "./ai-chat-sessions"
import {
  CHAT_DEFAULT_DIMENSIONS,
  CHAT_MAX_DIMENSIONS,
  CHAT_MIN_DIMENSIONS,
  dragResizeChat,
  resizeChat,
  type ChatDimensions,
} from "./ai-chat-size"
import type { PageContext } from "./ai-chat-types"

/** Global floating AI assistant. Mounted once inside the authenticated layout. */
export function AIChatWidget() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [dimensions, setDimensions] = useState<ChatDimensions>(CHAT_DEFAULT_DIMENSIONS)
  const [isDragging, setIsDragging] = useState(false)
  const dragStartRef = useRef<{ pointerId: number; x: number; y: number; dimensions: ChatDimensions } | null>(null)
  const { data: config } = useAIConfig()
  const { user } = useAuth()
  const { currentCluster } = useCluster()
  const {
    sessions, activeSession, createNewSession, selectSession, updateDraft, renameSession, deleteSession,
    sendMessage, approveAction, denyAction, stop,
  } = useAIChat(user?.id)

  const location = useLocation()
  const params = useParams()
  const [searchParams] = useSearchParams()

  const isAdmin = user?.role === "admin" || user?.role === "super-admin"
  const enabled = config?.enabled === true
  const ready = Boolean(enabled && config?.hasApiKey)
  const clusterId = Number(currentCluster)

  const pageContext: PageContext = {
    page: params.resource ?? location.pathname.replace(/^\//, "") ?? "overview",
    namespace: searchParams.get("namespace") ?? undefined,
    resourceName: params.name,
    resourceKind: params.resource,
  }

  const viewportDimensions = () => ({ width: window.innerWidth, height: window.innerHeight })

  const resizeByStep = (direction: -1 | 1) => {
    setDimensions((current) => resizeChat(current, direction, viewportDimensions()))
  }

  const startDragResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const panel = event.currentTarget.parentElement
    if (!panel) return
    const bounds = panel.getBoundingClientRect()
    dragStartRef.current = {
      pointerId: event.pointerId,
      x: event.clientX,
      y: event.clientY,
      dimensions: { width: bounds.width, height: bounds.height },
    }
    event.currentTarget.setPointerCapture(event.pointerId)
    setIsDragging(true)
  }

  const dragResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    const start = dragStartRef.current
    if (!start || start.pointerId !== event.pointerId) return
    setDimensions(dragResizeChat(start.dimensions, {
      width: event.clientX - start.x,
      height: event.clientY - start.y,
    }, viewportDimensions()))
  }

  const stopDragResize = (event: ReactPointerEvent<HTMLButtonElement>) => {
    if (dragStartRef.current?.pointerId !== event.pointerId) return
    dragStartRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    setIsDragging(false)
  }

  const resizeWithKeyboard = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
      event.preventDefault()
      resizeByStep(1)
    } else if (event.key === "ArrowRight" || event.key === "ArrowDown") {
      event.preventDefault()
      resizeByStep(-1)
    }
  }

  // A disabled assistant is an administrative setting, not a broken feature.
  // Hide it from regular users while leaving administrators a clear route to
  // re-enable it. Wait for config before rendering to avoid a misleading flash.
  if (!config || (!enabled && !isAdmin)) return null

  if (!open) {
    return (
      <Button
        size="icon-lg"
        variant={enabled ? "default" : "outline"}
        className="fixed bottom-6 right-6 z-50 rounded-full shadow-lg"
        onClick={() => setOpen(true)}
        aria-label={enabled ? t("ai.title") : t("ai.disabledTitle")}
        title={enabled ? t("ai.title") : t("ai.disabledTitle")}
      >
        <Bot className={enabled ? "size-5" : "size-5 text-muted-foreground"} />
      </Button>
    )
  }

  return (
    <div
      className={`fixed bottom-4 right-4 z-50 flex max-h-[calc(100dvh-2rem)] max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-xl border bg-background shadow-2xl sm:bottom-6 sm:right-6 ${isDragging ? "select-none" : "transition-[width,height] duration-200"}`}
      style={{ width: dimensions.width, height: dimensions.height }}
    >
      <button
        type="button"
        aria-label={t("ai.resizeChat")}
        className="absolute left-0 top-0 z-10 flex size-11 touch-none cursor-nwse-resize items-center justify-center text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
        onPointerDown={startDragResize}
        onPointerMove={dragResize}
        onPointerUp={stopDragResize}
        onPointerCancel={stopDragResize}
        onKeyDown={resizeWithKeyboard}
      >
        <MoveDiagonal2 className="size-4" />
      </button>
      {/* Header */}
      <div className="flex items-center gap-2 border-b py-2.5 pl-11 pr-3">
        <Bot className="size-4 shrink-0 text-primary" />
        <span className="min-w-0 truncate font-semibold">{t("ai.title")}</span>
        <TooltipProvider>
          <div className="ml-auto flex shrink-0 items-center gap-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-9"
                  disabled={dimensions.width <= CHAT_MIN_DIMENSIONS.width && dimensions.height <= CHAT_MIN_DIMENSIONS.height}
                  onClick={() => resizeByStep(-1)}
                  aria-label={t("ai.shrinkChat")}
                >
                  <ZoomOut className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{t("ai.shrinkChat")}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-9"
                  disabled={dimensions.width >= CHAT_MAX_DIMENSIONS.width && dimensions.height >= CHAT_MAX_DIMENSIONS.height}
                  onClick={() => resizeByStep(1)}
                  aria-label={t("ai.enlargeChat")}
                >
                  <ZoomIn className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{t("ai.enlargeChat")}</TooltipContent>
            </Tooltip>
            <Button
              size="icon"
              variant="ghost"
              className="size-9"
              onClick={() => setOpen(false)}
              aria-label={t("common.close")}
            >
              <X className="size-4" />
            </Button>
          </div>
        </TooltipProvider>
      </div>

      {/* Body */}
      {!enabled ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-sm text-muted-foreground">
          <Bot className="size-8 opacity-40" />
          <p>{t("ai.disabledMessage")}</p>
          <Button asChild size="sm" variant="outline">
            <Link to="/settings/ai" onClick={() => setOpen(false)}>
              <Settings className="size-4" />
              {t("ai.openSettings")}
            </Link>
          </Button>
        </div>
      ) : !ready ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-sm text-muted-foreground">
          <Bot className="size-8 opacity-40" />
          <p>{t("ai.missingAPIKey")}</p>
          <Button asChild size="sm" variant="outline">
            <Link to="/settings/ai" onClick={() => setOpen(false)}>
              <Settings className="size-4" />
              {t("ai.openSettings")}
            </Link>
          </Button>
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          <ChatSessions
            sessions={sessions}
            activeSessionId={activeSession.id}
            onCreate={createNewSession}
            onSelect={selectSession}
            onRename={renameSession}
            onDelete={deleteSession}
          />
          {activeSession.messages.length === 0 ? (
            <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 p-8 text-center text-sm text-muted-foreground">
              <Bot className="size-8 opacity-40" />
              <p>{t("ai.empty")}</p>
            </div>
          ) : (
            <ChatMessages
              messages={activeSession.messages}
              isLoading={activeSession.isRunning}
              onApprove={(message) => approveAction(activeSession.id, message)}
              onDeny={(message) => denyAction(activeSession.id, message, t("ai.actionCancelled"))}
            />
          )}
          <ChatComposer
            isLoading={activeSession.isRunning}
            disabled={!clusterId}
            value={activeSession.draft}
            onChange={(value) => updateDraft(activeSession.id, value)}
            onSend={(text) => sendMessage(activeSession.id, text, clusterId, pageContext)}
            onStop={() => stop(activeSession.id)}
          />
        </div>
      )}
    </div>
  )
}
