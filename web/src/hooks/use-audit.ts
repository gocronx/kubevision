import { useQuery } from "@tanstack/react-query"
import { getWithMeta } from "@/lib/api"

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
  source: string
  tool?: string
  correlationId?: string
  outcome: string
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

      const res = await getWithMeta<AuditLog[]>(`/audit-logs?${params.toString()}`)
      const items = Array.isArray(res.data) ? res.data : []
      return { items, total: res.meta?.total ?? items.length }
    },
    placeholderData: (prev) => prev,
  })
}
