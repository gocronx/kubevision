import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import api from "@/lib/api"

export interface Webhook {
  id: number
  createdAt: string
  updatedAt: string
  name: string
  url: string
  events: string[]
  clusters: string[]
  resources: string[]
  isActive: boolean
}

export interface WebhookPayload {
  name: string
  url: string
  secret?: string
  events: string[]
  clusters: string[]
  resources: string[]
  isActive: boolean
}

const WEBHOOKS_QUERY_KEY = ["webhooks"] as const

export function useWebhooks() {
  return useQuery<Webhook[]>({
    queryKey: WEBHOOKS_QUERY_KEY,
    queryFn: async () => {
      const res = await api.get("/webhooks")
      return (res as unknown as Webhook[]) ?? []
    },
  })
}

export function useCreateWebhook() {
  const queryClient = useQueryClient()
  return useMutation<Webhook, Error, WebhookPayload>({
    mutationFn: async (payload) => {
      const res = await api.post("/webhooks", payload)
      return res as unknown as Webhook
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WEBHOOKS_QUERY_KEY })
      toast.success("Webhook created")
    },
  })
}

export function useUpdateWebhook() {
  const queryClient = useQueryClient()
  return useMutation<Webhook, Error, { id: number; payload: WebhookPayload }>({
    mutationFn: async ({ id, payload }) => {
      const res = await api.put(`/webhooks/${id}`, payload)
      return res as unknown as Webhook
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WEBHOOKS_QUERY_KEY })
      toast.success("Webhook updated")
    },
  })
}

export function useDeleteWebhook() {
  const queryClient = useQueryClient()
  return useMutation<void, Error, number>({
    mutationFn: async (id) => {
      await api.delete(`/webhooks/${id}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WEBHOOKS_QUERY_KEY })
      toast.success("Webhook deleted")
    },
  })
}

export function useTestWebhook() {
  return useMutation<void, Error, number>({
    mutationFn: async (id) => {
      await api.post(`/webhooks/${id}/test`)
    },
    onSuccess: () => {
      toast.success("Test webhook sent successfully")
    },
    onError: () => {
      toast.error("Failed to send test webhook")
    },
  })
}
