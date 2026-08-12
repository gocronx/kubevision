import { useEffect, useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Loader2 } from "lucide-react"
import api from "@/lib/api"
import { useAuth } from "@/stores/auth-store"

interface OAuthCallbackResponse {
  accessToken: string
  refreshToken: string
  user: {
    id: number
    username: string
    role: string
    totpEnabled: boolean
  }
}

export function OAuthCallbackPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { login } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const missingParameters = !searchParams.get("code") || !searchParams.get("state")

  useEffect(() => {
    const provider = window.location.pathname.split("/").at(-2) ?? ""
    const code = searchParams.get("code")
    const state = searchParams.get("state")

    if (!code || !state) {
      return
    }

    api
      .get(`/auth/oauth/${provider}/callback`, { params: { code, state } })
      .then((res) => {
        const data = res as unknown as OAuthCallbackResponse
        login(data.accessToken, data.user, data.refreshToken)
        navigate("/overview", { replace: true })
      })
      .catch(() => {
        setError(t("oauth.callbackError"))
      })
  }, [searchParams, navigate, login, t])

  const displayedError = missingParameters ? "Missing OAuth callback parameters" : error

  if (displayedError) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="text-center">
          <p className="text-destructive">{displayedError}</p>
          <button
            className="mt-4 text-primary underline"
            onClick={() => navigate("/login")}
          >
            {t("common.login")}
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex items-center gap-2">
        <Loader2 className="size-5 animate-spin" />
        <span>{t("oauth.authenticating")}</span>
      </div>
    </div>
  )
}
