import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"

export interface OAuthProvider {
  name: string
  authUrl: string
}

export function useOAuthProviders() {
  return useQuery<OAuthProvider[]>({
    queryKey: ["oauth-providers"],
    queryFn: async () => {
      const res = await api.get("/auth/oauth/providers")
      return (res as unknown as OAuthProvider[]) ?? []
    },
  })
}

export async function startOAuthFlow(provider: string) {
  const res = await api.get(`/auth/oauth/${provider}/authorize`)
  const { authUrl } = res as unknown as { authUrl: string }
  window.location.href = authUrl
}
