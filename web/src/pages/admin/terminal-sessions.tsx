import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Play, TerminalSquare } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  useTerminalSessions,
  useTerminalSessionPlay,
  type TerminalSessionMeta,
} from "@/hooks/use-terminal-sessions"
import { TerminalPlayer, TerminalPlayerLoading } from "@/components/specialized/terminal-player"

// --------------------------------------------------------------------------
// Session playback dialog
// --------------------------------------------------------------------------

interface PlayDialogProps {
  session: TerminalSessionMeta | null
  onClose: () => void
}

function PlayDialog({ session, onClose }: PlayDialogProps) {
  const { data, isLoading } = useTerminalSessionPlay(session?.id ?? null)

  return (
    <Dialog open={!!session} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="flex h-[calc(100dvh-1rem)] flex-col gap-4 sm:h-[80vh] sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <TerminalSquare className="size-4" />
            {session?.pod} — {session?.namespace} / {session?.cluster}
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 min-h-0">
          {isLoading || !data ? (
            <TerminalPlayerLoading />
          ) : (
            <TerminalPlayer
              recording={data.recording}
              durationMs={data.durationMs}
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

// --------------------------------------------------------------------------
// Duration formatting helper
// --------------------------------------------------------------------------

function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000)
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return m > 0 ? `${m}m ${s}s` : `${s}s`
}

// --------------------------------------------------------------------------
// Terminal sessions page
// --------------------------------------------------------------------------

export function TerminalSessionsPage() {
  const { t } = useTranslation()
  const [page, setPage] = useState(0)
  const limit = 50
  const { data, isLoading } = useTerminalSessions(page * limit, limit)
  const sessions = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / limit))

  const [playing, setPlaying] = useState<TerminalSessionMeta | null>(null)

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("terminalSession.title")}</h1>
        <p className="text-muted-foreground text-sm">{t("terminalSession.description")}</p>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : sessions.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-2">
          <TerminalSquare className="size-10 opacity-30" />
          <p>{t("terminalSession.empty")}</p>
        </div>
      ) : (
        <>
          <div className="max-w-full overflow-x-auto rounded-md border">
            <table className="w-full min-w-[52rem] text-sm">
              <thead className="bg-muted/60">
                <tr>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("terminalSession.user")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("terminalSession.pod")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("common.cluster")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("common.namespace")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("terminalSession.duration")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("terminalSession.date")}</th>
                  <th className="text-left px-4 py-2 font-medium text-muted-foreground">{t("common.actions")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {sessions.map((sess) => (
                  <tr key={sess.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-2">
                      <Badge variant="outline" className="text-xs font-mono">
                        uid:{sess.userId}
                      </Badge>
                    </td>
                    <td className="px-4 py-2 font-mono text-xs">{sess.pod}</td>
                    <td className="px-4 py-2 text-xs text-muted-foreground">{sess.cluster}</td>
                    <td className="px-4 py-2 text-xs text-muted-foreground">{sess.namespace}</td>
                    <td className="px-4 py-2 text-xs tabular-nums">
                      {formatDuration(sess.durationMs)}
                    </td>
                    <td className="px-4 py-2 text-xs text-muted-foreground">
                      {new Date(sess.createdAt).toLocaleString()}
                    </td>
                    <td className="px-4 py-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 gap-1 text-xs"
                        onClick={() => setPlaying(sess)}
                      >
                        <Play className="size-3" />
                        {t("terminalSession.play")}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex flex-col gap-2 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
              <span>
                {t("terminalSession.showingPage", { page: page + 1, total: totalPages })}
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page === 0}
                  onClick={() => setPage((p) => Math.max(0, p - 1))}
                >
                  {t("terminalSession.prev")}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages - 1}
                  onClick={() => setPage((p) => p + 1)}
                >
                  {t("terminalSession.next")}
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      <PlayDialog session={playing} onClose={() => setPlaying(null)} />
    </div>
  )
}
