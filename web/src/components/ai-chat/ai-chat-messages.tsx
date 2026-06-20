import { useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import {
  AlertTriangle,
  Check,
  ChevronRight,
  Loader2,
  Wrench,
  X,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { cn } from "@/lib/utils"
import { ChatMarkdown } from "./ai-chat-markdown"
import type { ChatMessage } from "./ai-chat-types"

interface Props {
  messages: ChatMessage[]
  isLoading: boolean
  onApprove: (m: ChatMessage) => void
  onDeny: (m: ChatMessage) => void
}

export function ChatMessages({ messages, isLoading, onApprove, onDeny }: Props) {
  const endRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [messages])

  return (
    <div className="flex flex-col gap-3 p-4">
      {messages.map((m) =>
        m.role === "tool" ? (
          <ToolMessage key={m.id} message={m} onApprove={onApprove} onDeny={onDeny} />
        ) : (
          <TextMessage key={m.id} message={m} />
        )
      )}
      {isLoading && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="size-3 animate-spin" />
        </div>
      )}
      <div ref={endRef} />
    </div>
  )
}

function TextMessage({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user"
  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "max-w-[85%] rounded-lg px-3 py-2",
          isUser
            ? "bg-primary text-primary-foreground"
            : message.isError
              ? "bg-destructive/10 text-destructive"
              : "bg-muted"
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap break-words text-sm">{message.content}</p>
        ) : (
          <ChatMarkdown content={message.content} />
        )}
      </div>
    </div>
  )
}

function ToolMessage({
  message,
  onApprove,
  onDeny,
}: {
  message: ChatMessage
  onApprove: (m: ChatMessage) => void
  onDeny: (m: ChatMessage) => void
}) {
  const { t } = useTranslation()
  const pending = message.actionStatus === "pending"

  return (
    <div
      className={cn(
        "rounded-lg border px-3 py-2 text-sm",
        pending ? "border-amber-500/40 bg-amber-500/5" : "border-border bg-muted/40"
      )}
    >
      <div className="flex items-center gap-2">
        <Wrench className="size-3.5 text-muted-foreground" />
        <span className="font-medium">{describeAction(message)}</span>
        <StatusIcon status={message.actionStatus} />
      </div>

      {pending && (
        <>
          {hasPreview(message) && (
            <pre className="mt-2 max-h-48 overflow-auto rounded bg-muted p-2 font-mono text-xs">
              {previewText(message)}
            </pre>
          )}
          <div className="mt-2 flex gap-2">
            <Button size="sm" onClick={() => onApprove(message)}>
              <Check className="size-3.5" />
              {t("ai.confirm")}
            </Button>
            <Button size="sm" variant="outline" onClick={() => onDeny(message)}>
              {t("ai.cancel")}
            </Button>
          </div>
        </>
      )}

      {message.toolResult && (
        <Collapsible className="mt-2">
          <CollapsibleTrigger className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground">
            <ChevronRight className="size-3 transition-transform data-[state=open]:rotate-90" />
            {message.isError ? t("ai.toolError") : t("ai.toolResult")}
          </CollapsibleTrigger>
          <CollapsibleContent>
            <pre
              className={cn(
                "mt-1 max-h-48 overflow-auto rounded bg-muted p-2 font-mono text-xs",
                message.isError && "text-destructive"
              )}
            >
              {message.toolResult}
            </pre>
          </CollapsibleContent>
        </Collapsible>
      )}
    </div>
  )
}

function StatusIcon({ status }: { status?: ChatMessage["actionStatus"] }) {
  switch (status) {
    case "running":
      return <Loader2 className="ml-auto size-3.5 animate-spin text-muted-foreground" />
    case "confirmed":
      return <Check className="ml-auto size-3.5 text-green-600" />
    case "error":
      return <X className="ml-auto size-3.5 text-destructive" />
    case "denied":
      return <X className="ml-auto size-3.5 text-muted-foreground" />
    case "pending":
      return <AlertTriangle className="ml-auto size-3.5 text-amber-500" />
    default:
      return null
  }
}

// describeAction renders a short human summary of a tool call.
function describeAction(m: ChatMessage): string {
  const a = m.toolArgs ?? {}
  const ref = [a.namespace, a.name].filter(Boolean).join("/")
  const kind = String(a.kind ?? "")
  switch (m.toolName) {
    case "create_resource":
      return `Create ${kind}`
    case "update_resource":
      return `Update ${kind} ${ref}`
    case "patch_resource":
      return `Patch ${kind} ${ref}`
    case "delete_resource":
      return `Delete ${kind} ${ref}`
    case "get_resource":
      return `Get ${kind} ${ref}`
    case "list_resources":
      return `List ${kind}`
    case "get_pod_logs":
      return `Logs ${ref}`
    case "get_cluster_overview":
      return "Cluster overview"
    case "query_prometheus":
      return "Prometheus query"
    default:
      return m.toolName ?? "tool"
  }
}

function hasPreview(m: ChatMessage): boolean {
  const a = m.toolArgs ?? {}
  return Boolean(a.yaml || a.patch || a.query)
}

function previewText(m: ChatMessage): string {
  const a = m.toolArgs ?? {}
  if (a.yaml) return String(a.yaml)
  if (a.patch) return String(a.patch)
  if (a.query) return String(a.query)
  return JSON.stringify(a, null, 2)
}
