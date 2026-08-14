import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"

export interface APIKey {
  id: number
  name: string
  keyPrefix: string
  expiresAt: string | null
  createdAt: string
}

export interface GeneratedAPIKey extends APIKey {
  key: string // plain-text key shown only once
}

export interface GenerateAPIKeyPayload {
  name: string
  expiresAt?: string | null
}

const API_KEYS_QUERY_KEY = ["api-keys"] as const

// ---------------------------------------------------------------------------
// List user's API keys
// ---------------------------------------------------------------------------

export function useApiKeys() {
  return useQuery<APIKey[]>({
    queryKey: API_KEYS_QUERY_KEY,
    queryFn: async () => {
      return (await api.get<APIKey[]>("/api-keys")) ?? []
    },
  })
}

// ---------------------------------------------------------------------------
// Generate a new API key
// ---------------------------------------------------------------------------

export function useGenerateApiKey() {
  const queryClient = useQueryClient()

  return useMutation<GeneratedAPIKey, Error, GenerateAPIKeyPayload>({
    mutationFn: async (payload) => {
      return api.post<GeneratedAPIKey>("/api-keys", payload)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: API_KEYS_QUERY_KEY })
    },
  })
}

// ---------------------------------------------------------------------------
// Revoke (delete) an API key
// ---------------------------------------------------------------------------

export function useRevokeApiKey() {
  const queryClient = useQueryClient()

  return useMutation<void, Error, number>({
    mutationFn: async (id) => {
      await api.delete(`/api-keys/${id}`)
    },
    onMutate: async (id) => {
      await queryClient.cancelQueries({ queryKey: API_KEYS_QUERY_KEY })
      const previous = queryClient.getQueryData<APIKey[]>(API_KEYS_QUERY_KEY)
      queryClient.setQueryData<APIKey[]>(API_KEYS_QUERY_KEY, (old) =>
        old ? old.filter((k) => k.id !== id) : []
      )
      return { previous }
    },
    onError: (_err, _id, context) => {
      const ctx = context as { previous?: APIKey[] } | undefined
      if (ctx?.previous) {
        queryClient.setQueryData(API_KEYS_QUERY_KEY, ctx.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: API_KEYS_QUERY_KEY })
    },
  })
}
