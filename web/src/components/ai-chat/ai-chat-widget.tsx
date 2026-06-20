import { useState } from "react"
import { Link, useLocation, useParams, useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Bot, Settings, Trash2, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import { useCluster } from "@/hooks/use-cluster"
import { useAIChat } from "./use-ai-chat"
import { useAIConfig } from "./use-ai-config"
import { ChatMessages } from "./ai-chat-messages"
import { ChatComposer } from "./ai-chat-composer"
import type { PageContext } from "./ai-chat-types"

/** Global floating AI assistant. Mounted once inside the authenticated layout. */
export function AIChatWidget() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const { data: config } = useAIConfig()
  const { currentCluster } = useCluster()
  const { messages, isLoading, sendMessage, approveAction, denyAction, stop, clear } =
    useAIChat()

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
          <Button size="icon-sm" variant="ghost" onClick={clear} aria-label={t("ai.clear")}>
            <Trash2 className="size-4" />
          </Button>
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
        <>
          <ScrollArea className="flex-1">
            {messages.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center text-sm text-muted-foreground">
                <Bot className="size-8 opacity-40" />
                <p>{t("ai.empty")}</p>
              </div>
            ) : (
              <ChatMessages
                messages={messages}
                isLoading={isLoading}
                onApprove={approveAction}
                onDeny={denyAction}
              />
            )}
          </ScrollArea>
          <ChatComposer
            isLoading={isLoading}
            disabled={!clusterId}
            onSend={(text) => sendMessage(text, clusterId, pageContext)}
            onStop={stop}
          />
        </>
      )}
    </div>
  )
}
