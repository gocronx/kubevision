import { useState, useRef, useEffect } from "react"
import type { FormEvent, ChangeEvent } from "react"
import { useNavigate, useLocation } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Loader2, ShieldCheck, KeyRound, Eye, EyeOff, Languages } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/stores/auth-store"
import api from "@/lib/api"
import { useVerify2FA, useRecoveryCode } from "@/hooks/use-2fa"
import { loginWithPublicKey, publicKeyEnabled } from "@/lib/public-key-auth"
import { preventSubmitWhileComposing } from "@/lib/form-events"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

interface LoginResponse {
  accessToken: string
  refreshToken: string
  user: {
    id: number
    username: string
    role: string
    totpEnabled: boolean
  }
}

interface TwoFARequired {
  tempToken: string
}

type LoginStep = "credentials" | "2fa" | "recovery"

export function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuth()

  // Credentials step state
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [provider, setProvider] = useState<"local" | "directory">("local")
  const [showPublicKey, setShowPublicKey] = useState(false)

  // 2FA step state
  const [step, setStep] = useState<LoginStep>("credentials")
  const [tempToken, setTempToken] = useState("")
  const [totpCode, setTotpCode] = useState("")
  const [recoveryCode, setRecoveryCode] = useState("")

  const totpInputRef = useRef<HTMLInputElement>(null)
  const autoSubmitTimerRef = useRef<number | null>(null)

  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname ?? "/overview"

  const verify2FA = useVerify2FA()
  const useRecovery = useRecoveryCode()

  useEffect(() => {
    void publicKeyEnabled().then(setShowPublicKey)
  }, [])

  // Auto-focus the TOTP input when switching to the 2FA step
  useEffect(() => {
    if (step === "2fa") {
      const timer = window.setTimeout(() => totpInputRef.current?.focus(), 50)
      return () => window.clearTimeout(timer)
    }
  }, [step])

  useEffect(() => () => {
    if (autoSubmitTimerRef.current !== null) window.clearTimeout(autoSubmitTimerRef.current)
  }, [])

  function handleLoginSuccess(data: LoginResponse) {
    login(data.accessToken, data.user, data.refreshToken)
    toast.success(t("login.success", "Login successful"), { duration: 1500 })
    navigate(from, { replace: true })
  }

  async function handleCredentialsSubmit(e: FormEvent) {
    e.preventDefault()
    if (!username.trim() || !password.trim()) return

    setLoading(true)
    try {
      const data = await api.post<LoginResponse>("/auth/login", { username, password, provider })
      handleLoginSuccess(data)
    } catch (err: unknown) {
      // Check if it is the 2FA-required signal from the api interceptor
      if (err && typeof err === "object" && "is2FARequired" in err) {
        const payload = (err as { is2FARequired: boolean; data: TwoFARequired }).data
        setTempToken(payload.tempToken)
        setStep("2fa")
        setTotpCode("")
      }
      // All other errors are already toasted by the api interceptor
    } finally {
      setLoading(false)
    }
  }

  async function handlePublicKeyLogin() {
    setLoading(true)
    try {
      const data = await loginWithPublicKey<LoginResponse>(username.trim())
      handleLoginSuccess(data)
    } finally {
      setLoading(false)
    }
  }

  async function handleTotpSubmit(e: FormEvent) {
    e.preventDefault()
    if (totpCode.length !== 6) return

    try {
      const data = await verify2FA.mutateAsync({ tempToken, code: totpCode })
      handleLoginSuccess(data as LoginResponse)
    } catch {
      setTotpCode("")
      totpInputRef.current?.focus()
    }
  }

  function handleTotpChange(e: ChangeEvent<HTMLInputElement>) {
    const value = e.target.value.replace(/\D/g, "").slice(0, 6)
    setTotpCode(value)
    if (autoSubmitTimerRef.current !== null) {
      window.clearTimeout(autoSubmitTimerRef.current)
      autoSubmitTimerRef.current = null
    }
    // Auto-submit when 6 digits are entered
    if (value.length === 6) {
      // Use a small timeout to let React update the state before reading it
      autoSubmitTimerRef.current = window.setTimeout(async () => {
        autoSubmitTimerRef.current = null
        try {
          const data = await verify2FA.mutateAsync({ tempToken, code: value })
          handleLoginSuccess(data as LoginResponse)
        } catch {
          setTotpCode("")
          totpInputRef.current?.focus()
        }
      }, 50)
    }
  }

  async function handleRecoverySubmit(e: FormEvent) {
    e.preventDefault()
    if (!recoveryCode.trim()) return

    try {
      const data = await useRecovery.mutateAsync({ tempToken, recoveryCode: recoveryCode.trim() })
      handleLoginSuccess(data as LoginResponse)
    } catch {
      setRecoveryCode("")
    }
  }

  // ---- Render: credentials step ----
  if (step === "credentials") {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background p-4">
        <Card className="relative w-full max-w-sm">
          <LanguageToggle i18n={i18n} />
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">{t("login.title")}</CardTitle>
            <CardDescription>{t("login.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCredentialsSubmit} onKeyDownCapture={preventSubmitWhileComposing} className="flex flex-col gap-4">
				<div className="grid grid-cols-2 rounded-md border p-1" aria-label={t("login.provider")}>
					<Button type="button" size="sm" variant={provider === "local" ? "secondary" : "ghost"} onClick={() => setProvider("local")}>{t("login.local")}</Button>
					<Button type="button" size="sm" variant={provider === "directory" ? "secondary" : "ghost"} onClick={() => setProvider("directory")}>{t("login.directory")}</Button>
				</div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="username">{t("common.username")}</Label>
                <Input
                  id="username"
                  type="text"
                  placeholder={t("common.username")}
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  autoComplete="username"
                  required
                />
              </div>
              <div className="flex flex-col gap-2">
                <Label htmlFor="password">{t("common.password")}</Label>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    placeholder={t("common.password")}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="current-password"
                    className="pr-10"
                    required
                  />
                  <button
                    type="button"
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    onClick={() => setShowPassword((v) => !v)}
                    tabIndex={-1}
                  >
                    {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </button>
                </div>
              </div>
              <Button type="submit" className="w-full" disabled={loading}>
                {loading && <Loader2 className="animate-spin" />}
                {t("login.submit")}
              </Button>
              {showPublicKey && (
                <Button type="button" variant="outline" className="w-full" disabled={loading} onClick={handlePublicKeyLogin}>
                  <KeyRound className="size-4" />
                  {t("login.publicKey")}
                </Button>
              )}
            </form>
            <div className="mt-4 flex justify-center">
              <a
                href="https://github.com/gocronx/kubevision"
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
              >
                <svg className="size-4" viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/></svg>
                <span>GitHub</span>
              </a>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  // ---- Render: recovery code step ----
  if (step === "recovery") {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background p-4">
        <Card className="relative w-full max-w-sm">
          <LanguageToggle i18n={i18n} />
          <CardHeader className="text-center">
            <div className="flex justify-center mb-2">
              <KeyRound className="size-8 text-muted-foreground" />
            </div>
            <CardTitle className="text-xl">{t("twofa.recoveryTitle", "Recovery Code")}</CardTitle>
            <CardDescription>{t("twofa.recoveryDescription", "Enter one of your backup recovery codes")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleRecoverySubmit} onKeyDownCapture={preventSubmitWhileComposing} className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <Label htmlFor="recoveryCode">{t("twofa.recoveryCode", "Recovery Code")}</Label>
                <Input
                  id="recoveryCode"
                  type="text"
                  placeholder="XXXX-XXXX"
                  value={recoveryCode}
                  onChange={(e) => setRecoveryCode(e.target.value.toUpperCase())}
                  autoComplete="off"
                  autoFocus
                />
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={useRecovery.isPending || !recoveryCode.trim()}
              >
                {useRecovery.isPending && <Loader2 className="animate-spin" />}
                {t("twofa.verify", "Verify")}
              </Button>
              <Button
                type="button"
                variant="ghost"
                className="w-full text-sm"
                onClick={() => setStep("2fa")}
              >
                {t("twofa.useTotp", "Use authenticator app instead")}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  // ---- Render: TOTP code step ----
  return (
    <div className="flex min-h-svh items-center justify-center bg-background p-4">
      <Card className="relative w-full max-w-sm">
        <LanguageToggle i18n={i18n} />
        <CardHeader className="text-center">
          <div className="flex justify-center mb-2">
            <ShieldCheck className="size-8 text-primary" />
          </div>
          <CardTitle className="text-xl">{t("twofa.title", "Two-Factor Authentication")}</CardTitle>
          <CardDescription>
            {t("twofa.description", "Enter the 6-digit code from your authenticator app")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleTotpSubmit} onKeyDownCapture={preventSubmitWhileComposing} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="totpCode">{t("twofa.code", "Verification Code")}</Label>
              <Input
                ref={totpInputRef}
                id="totpCode"
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder={t("twofa.codePlaceholder", "Enter 6-digit code")}
                value={totpCode}
                onChange={handleTotpChange}
                autoComplete="one-time-code"
                maxLength={6}
                className="text-center text-2xl tracking-widest"
              />
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={verify2FA.isPending || totpCode.length !== 6}
            >
              {verify2FA.isPending && <Loader2 className="animate-spin" />}
              {t("twofa.verify", "Verify")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              className="w-full text-sm text-muted-foreground"
              onClick={() => setStep("recovery")}
            >
              {t("twofa.useRecovery", "Use a recovery code instead")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

function LanguageToggle({ i18n }: { i18n: { language: string; changeLanguage: (lng: string) => void } }) {
  return (
    <button
      type="button"
      className="absolute top-3 right-3 z-10 inline-flex items-center justify-center rounded-full size-8 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
      onClick={() => {
        const next = i18n.language === "zh" ? "en" : "zh"
        i18n.changeLanguage(next)
        localStorage.setItem("language", next)
      }}
      title={i18n.language === "zh" ? "Switch to English" : "切换为中文"}
    >
      <Languages className="size-4" />
    </button>
  )
}
