import { BrowserRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "@/components/ui/sonner"
import { AuthProvider } from "@/stores/auth-store"
import { AppearanceProvider } from "@/components/appearance-provider"
import { AppRoutes } from "@/routes"
import { useWebSocket } from "@/hooks/use-websocket"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30 * 1000,
      refetchOnWindowFocus: false,
    },
  },
})

/** Connects to WebSocket when authenticated. Must be inside AuthProvider + QueryClientProvider. */
function WebSocketManager() {
  useWebSocket()
  return null
}

export function App() {
  return (
    <AppearanceProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <WebSocketManager />
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
          <Toaster position="top-right" />
        </AuthProvider>
      </QueryClientProvider>
    </AppearanceProvider>
  )
}
