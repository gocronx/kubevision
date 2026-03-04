import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

export interface Template {
  id: number
  createdAt: string
  name: string
  category: string
  resourceType: string
  content: string
  isBuiltin: boolean
}

/**
 * Hook for fetching resource templates.
 * Optionally filter by category.
 */
export function useTemplates(category?: string) {
  return useQuery<Template[]>({
    queryKey: ["templates", category ?? ""],
    queryFn: async () => {
      const params: Record<string, string> = {}
      if (category) params.category = category
      const res = await api.get("/templates", { params })
      return (res as unknown as Template[]) ?? []
    },
    staleTime: 60_000,
  })
}
