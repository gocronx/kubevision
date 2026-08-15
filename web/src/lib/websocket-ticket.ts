import api from "@/lib/api"

interface WebSocketTicketResponse {
  ticket: string
  expiresIn: number
}

export async function createWebSocketTicket(signal?: AbortSignal): Promise<string> {
  const response = await api.post<WebSocketTicketResponse>("/ws/ticket", undefined, { signal })
  return response.ticket
}
