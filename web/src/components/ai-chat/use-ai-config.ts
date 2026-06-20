import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import api from "@/lib/api"
import type { AIConfigUpdate, AIConfigView } from "./ai-chat-types"

const KEY = ["ai", "config"]

/** Reads the AI configuration (key never returned). */
export function useAIConfig() {
  return useQuery<AIConfigView>({
    queryKey: KEY,
    queryFn: async () => (await api.get("/ai/config")) as unknown as AIConfigView,
  })
}

/** Updates the AI configuration (admin only). */
export function useUpdateAIConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (payload: AIConfigUpdate) =>
      (await api.put("/ai/config", payload)) as unknown as AIConfigView,
    onSuccess: () => qc.invalidateQueries({ queryKey: KEY }),
  })
}
