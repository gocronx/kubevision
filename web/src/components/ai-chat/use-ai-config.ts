import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"
import type { AIConfigUpdate, AIConfigView, AIModel } from "./ai-chat-types"

const KEY = ["ai", "config"]

/** Reads the AI configuration (key never returned). */
export function useAIConfig() {
  return useQuery<AIConfigView>({
    queryKey: KEY,
    queryFn: () => api.get<AIConfigView>("/ai/config"),
  })
}

/** Updates the AI configuration (admin only). */
export function useUpdateAIConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (payload: AIConfigUpdate) =>
      api.put<AIConfigView>("/ai/config", payload),
    onSuccess: (config) => {
      // The update response is authoritative. Publishing it immediately keeps
      // the global chat entry in sync without waiting for a follow-up request.
      qc.setQueryData(KEY, config)
    },
  })
}

export function useDiscoverAIModels() {
  return useMutation({
    mutationFn: async (payload: { baseURL: string; apiKey: string }) =>
      api.post<AIModel[]>("/ai/models", payload),
  })
}
