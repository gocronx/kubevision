import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  AlertTriangle,
  ArrowDown,
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
import type { ChatMessage } from "./ai-chat-types"

const ChatMarkdown = lazy(() =>
  import("./ai-chat-markdown").then((module) => ({ default: module.ChatMarkdown }))
)

interface Props {
  messages: ChatMessage[]
  isLoading: boolean
  onApprove: (m: ChatMessage) => void
  onDeny: (m: ChatMessage) => void
}

export function ChatMessages({ messages, isLoading, onApprove, onDeny }: Props) {
  const { t } = useTranslation()
  const scrollRef = useRef<HTMLDivElement>(null)
  const followOutputRef = useRef(true)
  const lastMessageIDRef = useRef<string | undefined>(undefined)
  const [showJumpToLatest, setShowJumpToLatest] = useState(false)
  const items = useMemo(() => groupMessages(messages), [messages])

  useEffect(() => {
    const viewport = scrollRef.current
    const lastMessage = messages.at(-1)
    const isNewUserMessage = lastMessage?.role === "user" && lastMessage.id !== lastMessageIDRef.current
    lastMessageIDRef.current = lastMessage?.id
    if (isNewUserMessage) {
      followOutputRef.current = true
      requestAnimationFrame(() => setShowJumpToLatest(false))
    }
    if (!viewport || !followOutputRef.current) return
    viewport.scrollTop = viewport.scrollHeight
  }, [messages])

  const onScroll = () => {
    const viewport = scrollRef.current
    if (!viewport) return
    const nearBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight < 48
    followOutputRef.current = nearBottom
    setShowJumpToLatest(!nearBottom)
  }

  const jumpToLatest = () => {
    const viewport = scrollRef.current
    if (!viewport) return
    followOutputRef.current = true
    setShowJumpToLatest(false)
    viewport.scrollTo({ top: viewport.scrollHeight, behavior: "smooth" })
  }

  return (
    <div className="relative min-h-0 flex-1 overflow-hidden">
      <div
        ref={scrollRef}
        onScroll={onScroll}
        className="size-full min-h-0 overflow-y-auto overscroll-contain scroll-smooth"
        aria-live="polite"
        aria-label={t("ai.conversation")}
      >
        <div className="flex min-w-0 flex-col gap-3 p-4">
          {items.map((item) =>
            item.type === "activity" ? (
              <ActivityGroup key={item.messages.map((m) => m.id).join(":")} messages={item.messages} />
            ) : item.message.role === "tool" ? (
              <ToolMessage key={item.message.id} message={item.message} onApprove={onApprove} onDeny={onDeny} />
            ) : (
              <TextMessage key={item.message.id} message={item.message} />
            )
          )}
          {isLoading && (
            <div className="flex h-5 items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="size-3 animate-spin" />
              <span>{t("ai.responding")}</span>
            </div>
          )}
        </div>
      </div>
      {showJumpToLatest && (
        <Button
          size="icon-sm"
          variant="secondary"
          className="absolute bottom-3 left-1/2 -translate-x-1/2 rounded-full shadow-md"
          onClick={jumpToLatest}
          aria-label={t("ai.jumpToLatest")}
        >
          <ArrowDown className="size-4" />
        </Button>
      )}
    </div>
  )
}

type MessageItem =
  | { type: "message"; message: ChatMessage }
  | { type: "activity"; messages: ChatMessage[] }

function groupMessages(messages: ChatMessage[]): MessageItem[] {
  const items: MessageItem[] = []
  for (const message of messages) {
    const collapsibleTool = message.role === "tool" && message.actionStatus !== "pending"
    const previous = items.at(-1)
    if (collapsibleTool && previous?.type === "activity") {
      previous.messages.push(message)
    } else if (collapsibleTool) {
      items.push({ type: "activity", messages: [message] })
    } else {
      items.push({ type: "message", message })
    }
  }
  return items
}

function ActivityGroup({ messages }: { messages: ChatMessage[] }) {
  const { t } = useTranslation()
  const running = messages.some((message) => message.actionStatus === "running")
  const failed = messages.some((message) => message.actionStatus === "error")

  return (
    <Collapsible className="rounded-md border bg-muted/20 px-3 py-2">
      <CollapsibleTrigger className="group flex w-full items-center gap-2 text-left text-xs text-muted-foreground hover:text-foreground">
        <ChevronRight className="size-3.5 shrink-0 transition-transform group-data-[state=open]:rotate-90" />
        <span>{t("ai.activitySteps", { count: messages.length })}</span>
        {running ? (
          <Loader2 className="ml-auto size-3.5 animate-spin" />
        ) : failed ? (
          <AlertTriangle className="ml-auto size-3.5 text-destructive" />
        ) : (
          <Check className="ml-auto size-3.5 text-green-600" />
        )}
      </CollapsibleTrigger>
      <CollapsibleContent className="pt-2">
        <div className="space-y-2 border-l pl-2">
          {messages.map((message) => <ToolActivity key={message.id} message={message} />)}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}

function ToolActivity({ message }: { message: ChatMessage }) {
  const { t } = useTranslation()
  return (
    <div className="min-w-0 text-xs">
      <div className="flex items-center gap-2">
        <Wrench className="size-3 shrink-0 text-muted-foreground" />
        <span className="min-w-0 flex-1 truncate">{describeAction(message)}</span>
        <StatusIcon status={message.actionStatus} />
      </div>
      {message.toolResult && (
        <Collapsible className="mt-1 pl-5">
          <CollapsibleTrigger className="group flex items-center gap-1 text-muted-foreground hover:text-foreground">
            <ChevronRight className="size-3 transition-transform group-data-[state=open]:rotate-90" />
            {message.isError ? t("ai.toolError") : t("ai.toolResult")}
          </CollapsibleTrigger>
          <CollapsibleContent>
            <pre className={cn("mt-1 max-h-32 overflow-auto whitespace-pre-wrap break-words rounded bg-muted p-2 font-mono text-xs", message.isError && "text-destructive")}>{message.toolResult}</pre>
          </CollapsibleContent>
        </Collapsible>
      )}
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
          <Suspense fallback={<p className="whitespace-pre-wrap break-words text-sm">{message.content}</p>}>
            <ChatMarkdown content={message.content} />
          </Suspense>
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
            <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap break-words rounded bg-muted p-2 font-mono text-xs">
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
