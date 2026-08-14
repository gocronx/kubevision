import { useQuery } from "@tanstack/react-query"
import api, { getWithMeta } from "@/lib/api"

export interface TerminalSessionMeta {
  id: number
  createdAt: string
  userId: number
  cluster: string
  namespace: string
  pod: string
  container: string
  durationMs: number
  expiresAt: string
}

export interface TerminalSessionPlayData {
  id: number
  recording: string
  durationMs: number
}

const SESSIONS_QUERY_KEY = ["terminal-sessions"] as const

export function useTerminalSessions(offset = 0, limit = 50) {
  return useQuery<{ items: TerminalSessionMeta[]; total: number }>({
    queryKey: [...SESSIONS_QUERY_KEY, offset, limit],
    queryFn: async () => {
      const res = await getWithMeta<TerminalSessionMeta[]>(
        `/terminal-sessions?offset=${offset}&limit=${limit}`
      )
      const items = Array.isArray(res.data) ? res.data : []
      return { items, total: res.meta?.total ?? items.length }
    },
  })
}

export function useTerminalSessionPlay(id: number | null) {
  return useQuery<TerminalSessionPlayData>({
    queryKey: [...SESSIONS_QUERY_KEY, id, "play"],
    queryFn: async () => {
      return api.get<TerminalSessionPlayData>(`/terminal-sessions/${id}/play`)
    },
    enabled: id !== null,
  })
}
