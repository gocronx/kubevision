import { useState } from "react"
import { Link, useLocation, useParams, useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Bot, Settings, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useCluster } from "@/hooks/use-cluster"
import { useAuth } from "@/stores/auth-store"
import { useAIChat } from "./use-ai-chat"
import { useAIConfig } from "./use-ai-config"
import { ChatMessages } from "./ai-chat-messages"
import { ChatComposer } from "./ai-chat-composer"
import { ChatSessions } from "./ai-chat-sessions"
import type { PageContext } from "./ai-chat-types"

/** Global floating AI assistant. Mounted once inside the authenticated layout. */
export function AIChatWidget() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
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

  const ready = Boolean(config?.enabled && config?.hasApiKey)
  const clusterId = Number(currentCluster)

  const pageContext: PageContext = {
    page: params.resource ?? location.pathname.replace(/^\//, "") ?? "overview",
    namespace: searchParams.get("namespace") ?? undefined,
    resourceName: params.name,
    resourceKind: params.resource,
  }

  if (!open) {
    return (
      <Button
        size="icon-lg"
        className="fixed bottom-6 right-6 z-50 rounded-full shadow-lg"
        onClick={() => setOpen(true)}
        aria-label={t("ai.title")}
      >
        <Bot className="size-5" />
      </Button>
    )
  }

  return (
    <div className="fixed bottom-6 right-6 z-50 flex h-[600px] max-h-[85vh] w-[400px] max-w-[calc(100vw-2rem)] flex-col overflow-hidden rounded-xl border bg-background shadow-2xl">
      {/* Header */}
      <div className="flex items-center gap-2 border-b px-4 py-3">
        <Bot className="size-4 text-primary" />
        <span className="font-semibold">{t("ai.title")}</span>
        <div className="ml-auto flex items-center gap-1">
          <Button size="icon-sm" variant="ghost" onClick={() => setOpen(false)} aria-label={t("common.close")}>
            <X className="size-4" />
          </Button>
        </div>
      </div>

      {/* Body */}
      {!ready ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 p-6 text-center text-sm text-muted-foreground">
          <Bot className="size-8 opacity-40" />
          <p>{t("ai.notConfigured")}</p>
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
