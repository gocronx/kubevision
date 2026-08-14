import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

export interface RegistryTagPage {
  reference: {
    registry: string
    repository: string
    tag?: string
    digest?: string
  }
  tags: string[]
  nextCursor?: string
  cached: boolean
}

export function useRegistryTags(image: string, enabled = true) {
  return useQuery({
    queryKey: ["registry-tags", image],
    queryFn: () => api.get<RegistryTagPage>("/registry-tags", {
      params: { image, limit: 50 },
    }),
    enabled: enabled && image.trim().length > 0,
    retry: false,
    staleTime: 30_000,
  })
}
