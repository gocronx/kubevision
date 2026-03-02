import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

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

interface TerminalSessionListResponse {
  items: TerminalSessionMeta[]
  total: number
}

const SESSIONS_QUERY_KEY = ["terminal-sessions"] as const

export function useTerminalSessions(offset = 0, limit = 50) {
  return useQuery<{ items: TerminalSessionMeta[]; total: number }>({
    queryKey: [...SESSIONS_QUERY_KEY, offset, limit],
    queryFn: async () => {
      const res = await api.get(`/terminal-sessions?offset=${offset}&limit=${limit}`)
      // The api interceptor returns the data directly; for paginated endpoints
      // the data is the array and meta.total is in the meta field.
      // We rely on the raw response here.
      const raw = res as unknown as TerminalSessionMeta[] | TerminalSessionListResponse
      if (Array.isArray(raw)) {
        return { items: raw, total: raw.length }
      }
      return { items: raw.items ?? [], total: raw.total ?? 0 }
    },
  })
}

export function useTerminalSessionPlay(id: number | null) {
  return useQuery<TerminalSessionPlayData>({
    queryKey: [...SESSIONS_QUERY_KEY, id, "play"],
    queryFn: async () => {
      const res = await api.get(`/terminal-sessions/${id}/play`)
      return res as unknown as TerminalSessionPlayData
    },
    enabled: id !== null,
  })
}
