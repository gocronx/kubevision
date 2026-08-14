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
      return (await api.get<OAuthProvider[]>("/auth/oauth/providers")) ?? []
    },
  })
}

export async function startOAuthFlow(provider: string) {
  const { authUrl } = await api.get<{ authUrl: string }>(`/auth/oauth/${provider}/authorize`)
  window.location.href = authUrl
}
