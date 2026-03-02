import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

export interface AuditLog {
  id: number
  createdAt: string
  userId: number
  username: string
  action: string
  resource: string
  name: string
  namespace: string
  cluster: string
  statusCode: number
  durationMs: number
  clientIp: string
}

export interface AuditLogFilter {
  action?: string
  cluster?: string
  since?: string
  offset?: number
  limit?: number
}

interface AuditLogsResult {
  items: AuditLog[]
  total: number
}

const AUDIT_QUERY_KEY = ["audit-logs"] as const

export function useAuditLogs(filter: AuditLogFilter = {}) {
  return useQuery<AuditLogsResult>({
    queryKey: [...AUDIT_QUERY_KEY, filter],
    queryFn: async () => {
      const params = new URLSearchParams()
      if (filter.action) params.set("action", filter.action)
      if (filter.cluster) params.set("cluster", filter.cluster)
      if (filter.since) params.set("since", filter.since)
      params.set("offset", String(filter.offset ?? 0))
      params.set("limit", String(filter.limit ?? 50))

      const res = await api.get(`/audit-logs?${params.toString()}`)
      // The interceptor extracts body.data. For paginated endpoints the
      // data is an array; we coerce accordingly.
      const items = Array.isArray(res) ? (res as AuditLog[]) : []
      return { items, total: items.length }
    },
    placeholderData: (prev) => prev,
  })
}
