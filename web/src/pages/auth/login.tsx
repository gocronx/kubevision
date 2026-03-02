import { useState, useRef, useEffect } from "react"
import type { FormEvent, ChangeEvent } from "react"
import { useNavigate, useLocation } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Loader2, ShieldCheck, KeyRound, Eye, EyeOff } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/stores/auth-store"
import api from "@/lib/api"
import { useVerify2FA, useRecoveryCode } from "@/hooks/use-2fa"
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
  const { t } = useTranslation()
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuth()

  // Credentials step state
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)

  // 2FA step state
  const [step, setStep] = useState<LoginStep>("credentials")
  const [tempToken, setTempToken] = useState("")
  const [totpCode, setTotpCode] = useState("")
  const [recoveryCode, setRecoveryCode] = useState("")

  const totpInputRef = useRef<HTMLInputElement>(null)

  const from = (location.state as { from?: { pathname: string } } | null)?.from?.pathname ?? "/overview"

  const verify2FA = useVerify2FA()
  const useRecovery = useRecoveryCode()

  // Auto-focus the TOTP input when switching to the 2FA step
  useEffect(() => {
    if (step === "2fa") {
      setTimeout(() => totpInputRef.current?.focus(), 50)
    }
  }, [step])

  function handleLoginSuccess(data: LoginResponse) {
    login(data.accessToken, data.user, data.refreshToken)
    toast.success(t("login.success", "Login successful"))
    navigate(from, { replace: true })
  }

  async function handleCredentialsSubmit(e: FormEvent) {
    e.preventDefault()
    if (!username.trim() || !password.trim()) return

    setLoading(true)
    try {
      const data = await api.post("/auth/login", { username, password }) as LoginResponse
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
    // Auto-submit when 6 digits are entered
    if (value.length === 6) {
      // Use a small timeout to let React update the state before reading it
      setTimeout(async () => {
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
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl">{t("login.title")}</CardTitle>
            <CardDescription>{t("login.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCredentialsSubmit} className="flex flex-col gap-4">
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
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  // ---- Render: recovery code step ----
  if (step === "recovery") {
    return (
      <div className="flex min-h-svh items-center justify-center bg-background p-4">
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <div className="flex justify-center mb-2">
              <KeyRound className="size-8 text-muted-foreground" />
            </div>
            <CardTitle className="text-xl">{t("twofa.recoveryTitle", "Recovery Code")}</CardTitle>
            <CardDescription>{t("twofa.recoveryDescription", "Enter one of your backup recovery codes")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleRecoverySubmit} className="flex flex-col gap-4">
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
      <Card className="w-full max-w-sm">
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
          <form onSubmit={handleTotpSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="totpCode">{t("twofa.code", "Verification Code")}</Label>
              <Input
                ref={totpInputRef}
                id="totpCode"
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder="000000"
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
