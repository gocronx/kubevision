import { useState, type FormEvent, type MouseEvent } from "react"
import { useTranslation } from "react-i18next"
import { Check, ChevronDown, Loader2, MessageSquare, Pencil, Plus, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"
import { preventSubmitWhileComposing } from "@/lib/form-events"
import type { ChatSession } from "./ai-chat-types"

const MAX_SESSION_TITLE_LENGTH = 60

interface Props {
  sessions: ChatSession[]
  activeSessionId: string
  onCreate: () => void
  onSelect: (sessionId: string) => void
  onRename: (sessionId: string, title: string) => void
  onDelete: (sessionId: string) => void
}

export function ChatSessions({ sessions, activeSessionId, onCreate, onSelect, onRename, onDelete }: Props) {
  const { t, i18n } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  const [renameTarget, setRenameTarget] = useState<ChatSession | null>(null)
  const [renameValue, setRenameValue] = useState("")
  const [deleteTarget, setDeleteTarget] = useState<ChatSession | null>(null)
  const activeSession = sessions.find((session) => session.id === activeSessionId) ?? sessions[0]
  const trimmedRename = renameValue.trim()
  const renameTooLong = trimmedRename.length > MAX_SESSION_TITLE_LENGTH
  const canRename = Boolean(trimmedRename && !renameTooLong && trimmedRename !== renameTarget?.title)

  const titleFor = (session: ChatSession) => session.title || t("ai.untitledSession")
  const timeFor = (session: ChatSession) => session.isRunning
    ? t("ai.running")
    : new Intl.DateTimeFormat(i18n.language, { hour: "2-digit", minute: "2-digit" }).format(session.updatedAt)

  const openRename = (event: MouseEvent, session: ChatSession) => {
    event.preventDefault()
    event.stopPropagation()
    if (session.isRunning) return
    setMenuOpen(false)
    setRenameValue(session.title)
    setRenameTarget(session)
  }

  const openDelete = (event: MouseEvent, session: ChatSession) => {
    event.preventDefault()
    event.stopPropagation()
    setMenuOpen(false)
    setDeleteTarget(session)
  }

  const submitRename = (event: FormEvent) => {
    event.preventDefault()
    if (!renameTarget || !canRename) return
    onRename(renameTarget.id, trimmedRename)
    setRenameTarget(null)
  }

  const confirmDelete = () => {
    if (!deleteTarget) return
    onDelete(deleteTarget.id)
    setDeleteTarget(null)
  }

  if (!activeSession) return null

  return (
    <>
      <div className="shrink-0 border-b bg-muted/20 p-2">
        <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" className="h-10 w-full min-w-0 justify-start bg-background px-3">
              {activeSession.isRunning ? (
                <Loader2 className="size-4 shrink-0 animate-spin text-primary" />
              ) : (
                <MessageSquare className="size-4 shrink-0 text-muted-foreground" />
              )}
              <span className="min-w-0 flex-1 truncate text-left">{titleFor(activeSession)}</span>
              <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
              <span className="sr-only">{t("ai.sessions")}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-[var(--radix-dropdown-menu-trigger-width)] max-w-[calc(100vw-1rem)] sm:min-w-72">
            {sessions.map((session) => {
              const selected = session.id === activeSessionId
              return (
                <DropdownMenuItem
                  key={session.id}
                  className={cn("min-h-12 gap-2 pr-1", selected && "bg-accent")}
                  onSelect={() => onSelect(session.id)}
                >
                  {selected ? (
                    <Check className="size-4 shrink-0 text-primary" />
                  ) : session.isRunning ? (
                    <Loader2 className="size-4 shrink-0 animate-spin text-primary" />
                  ) : (
                    <MessageSquare className="size-4 shrink-0" />
                  )}
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-medium">{titleFor(session)}</span>
                    <span className="block text-xs text-muted-foreground">{timeFor(session)}</span>
                  </span>
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    disabled={session.isRunning}
                    onClick={(event) => openRename(event, session)}
                    aria-label={t("ai.renameSession", { title: titleFor(session) })}
                  >
                    <Pencil className="size-3.5" />
                  </Button>
                  <Button
                    type="button"
                    size="icon-sm"
                    variant="ghost"
                    onClick={(event) => openDelete(event, session)}
                    aria-label={t("ai.deleteSession", { title: titleFor(session) })}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </DropdownMenuItem>
              )
            })}
            <DropdownMenuSeparator />
            <DropdownMenuItem className="min-h-10 font-medium" onSelect={onCreate}>
              <Plus className="size-4" />
              {t("ai.newSession")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      <Dialog open={Boolean(renameTarget)} onOpenChange={(open) => !open && setRenameTarget(null)}>
        <DialogContent className="sm:max-w-sm">
          <form onSubmit={submitRename} onKeyDownCapture={preventSubmitWhileComposing} className="grid gap-4">
            <DialogHeader>
              <DialogTitle>{t("ai.renameSessionTitle")}</DialogTitle>
              <DialogDescription>{t("ai.renameSessionDescription")}</DialogDescription>
            </DialogHeader>
            <div className="grid gap-2">
              <Label htmlFor="ai-session-name">{t("ai.sessionName")}</Label>
              <Input
                id="ai-session-name"
                autoFocus
                value={renameValue}
                maxLength={MAX_SESSION_TITLE_LENGTH + 1}
                aria-invalid={renameTooLong}
                placeholder={t("ai.sessionNamePlaceholder")}
                onChange={(event) => setRenameValue(event.target.value)}
              />
              {renameTooLong && <p className="text-xs text-destructive">{t("ai.sessionNameTooLong", { max: MAX_SESSION_TITLE_LENGTH })}</p>}
            </div>
            <DialogFooter>
              <DialogClose asChild>
                <Button type="button" variant="outline">{t("common.cancel")}</Button>
              </DialogClose>
              <Button type="submit" disabled={!canRename}>{t("ai.renameSessionSave")}</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(deleteTarget)} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("ai.deleteSessionTitle")}</DialogTitle>
            <DialogDescription>{t("ai.deleteSessionDescription")}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">{t("common.cancel")}</Button>
            </DialogClose>
            <Button variant="destructive" onClick={confirmDelete}>{t("ai.deleteSessionConfirm")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
