import { useEffect, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ShieldCheck, ShieldOff, Copy, CheckCircle, Loader2, Eye, EyeOff, Lock, KeyRound, Pencil, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/stores/auth-store"
import {
  useSetup2FA,
  useEnable2FA,
  useDisable2FA,
  type Setup2FAResponse,
} from "@/hooks/use-2fa"
import { useChangePassword } from "@/hooks/use-users"
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { getPublicKeyConfig, listPublicKeys, publicKeyAvailable, registerPublicKey, renamePublicKey, revokePublicKey, type PublicKeyCredentialInfo } from "@/lib/public-key-auth"

type SetupStep = "idle" | "qrcode" | "verify" | "recovery"

function PublicKeyCredentialsCard() {
  const { t, i18n } = useTranslation()
  const [credentials, setCredentials] = useState<PublicKeyCredentialInfo[]>([])
  const [label, setLabel] = useState("")
  const [password, setPassword] = useState("")
  const [totpCode, setTotpCode] = useState("")
  const [busy, setBusy] = useState(false)
  const [enabled, setEnabled] = useState(false)

  async function refresh() {
    try { setCredentials(await listPublicKeys()) } catch { setCredentials([]) }
  }
  useEffect(() => {
    void getPublicKeyConfig()
      .then((config) => {
        setEnabled(config.enabled)
        if (config.enabled) void refresh()
      })
      .catch(() => setEnabled(false))
  }, [])

  async function register() {
    setBusy(true)
    try {
      await registerPublicKey(label.trim(), password, totpCode)
      setLabel(""); setPassword(""); setTotpCode("")
      toast.success(t("publicKey.registeredToast"))
      await refresh()
    } finally { setBusy(false) }
  }

  async function rename(item: PublicKeyCredentialInfo) {
    const next = window.prompt(t("publicKey.label"), item.label)?.trim()
    if (!next || next === item.label) return
    await renamePublicKey(item.id, next)
    await refresh()
  }

  async function revoke(item: PublicKeyCredentialInfo) {
    if (!window.confirm(t("publicKey.revokeConfirm", { label: item.label }))) return
    await revokePublicKey(item.id)
    toast.success(t("publicKey.revokedToast"))
    await refresh()
  }

  if (!enabled) return null

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3"><KeyRound className="size-5" /><div><CardTitle className="text-base">{t("publicKey.title")}</CardTitle><CardDescription>{t("publicKey.description")}</CardDescription></div></div>
      </CardHeader>
      <CardContent className="space-y-4">
        {publicKeyAvailable() && (
          <div className="grid gap-3 sm:grid-cols-2">
            <Input aria-label={t("publicKey.label")} placeholder={t("publicKey.label")} value={label} onChange={(event) => setLabel(event.target.value)} />
            <Input aria-label={t("users.oldPassword")} type="password" autoComplete="current-password" placeholder={t("users.oldPassword")} value={password} onChange={(event) => setPassword(event.target.value)} />
            <Input aria-label={t("publicKey.authenticatorCode")} inputMode="numeric" placeholder={t("publicKey.authenticatorCodeOptional")} value={totpCode} onChange={(event) => setTotpCode(event.target.value.replace(/\D/g, "").slice(0, 6))} />
            <Button disabled={busy || !label.trim() || (!password && totpCode.length !== 6)} onClick={register}>{busy ? <Loader2 className="animate-spin" /> : <KeyRound />}{t("publicKey.register")}</Button>
          </div>
        )}
        <div className="divide-y rounded-md border">
          {credentials.length === 0 ? <p className="p-4 text-sm text-muted-foreground">{t("publicKey.empty")}</p> : credentials.map((item) => (
            <div key={item.id} className="flex items-center justify-between gap-3 p-3">
              <div className="min-w-0"><p className="truncate text-sm font-medium">{item.label}</p><p className="text-xs text-muted-foreground">{item.lastUsedAt ? t("publicKey.lastUsed", { date: new Date(item.lastUsedAt).toLocaleDateString(i18n.language) }) : t("publicKey.added", { date: new Date(item.createdAt).toLocaleDateString(i18n.language) })}</p></div>
              <div className="flex shrink-0 gap-1"><Button variant="ghost" size="icon" title={t("publicKey.rename")} onClick={() => void rename(item)}><Pencil /></Button><Button variant="ghost" size="icon" title={t("publicKey.revoke")} onClick={() => void revoke(item)}><Trash2 /></Button></div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Change Password Card
// ---------------------------------------------------------------------------

function ChangePasswordCard() {
  const { t } = useTranslation()
  const { logout } = useAuth()
  const navigate = useNavigate()
  const changePasswordMutation = useChangePassword()

  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")

  async function handleSubmit() {
    if (newPassword !== confirmPassword) {
      toast.error(t("users.passwordMismatch"))
      return
    }
    if (newPassword.length < 6) {
      toast.error(t("users.passwordTooShort"))
      return
    }
    try {
      await changePasswordMutation.mutateAsync({ oldPassword, newPassword })
      toast.success(t("users.passwordChangedToast"))
      // Token version was bumped — log out and redirect to login.
      logout()
      navigate("/login")
    } catch {
      // toasted by api interceptor
    }
    setOldPassword("")
    setNewPassword("")
    setConfirmPassword("")
  }

  const isValid =
    oldPassword.trim().length > 0 &&
    newPassword.trim().length > 0 &&
    confirmPassword.trim().length > 0

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <Lock className="size-5 text-muted-foreground" />
          <div>
            <CardTitle className="text-base">
              {t("users.changePassword")}
            </CardTitle>
            <CardDescription>
              {t("users.changePasswordDescription")}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-1">
          <Label htmlFor="cp-old">{t("users.oldPassword")}</Label>
          <Input
            id="cp-old"
            type="password"
            value={oldPassword}
            onChange={(e) => setOldPassword(e.target.value)}
            placeholder="••••••••"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="cp-new">{t("users.newPassword")}</Label>
          <Input
            id="cp-new"
            type="password"
            value={newPassword}
            onChange={(e) => setNewPassword(e.target.value)}
            placeholder="••••••••"
          />
        </div>
        <div className="space-y-1">
          <Label htmlFor="cp-confirm">{t("users.confirmPassword")}</Label>
          <Input
            id="cp-confirm"
            type="password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            placeholder="••••••••"
          />
        </div>
        <Button
          onClick={handleSubmit}
          disabled={!isValid || changePasswordMutation.isPending}
        >
          {changePasswordMutation.isPending && <Loader2 className="animate-spin" />}
          {t("users.changePassword")}
        </Button>
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Security Settings Page
// ---------------------------------------------------------------------------

export function SecuritySettingsPage() {
  const { t } = useTranslation()
  const { user, login, token } = useAuth()
  const [is2FAEnabled, setIs2FAEnabled] = useState(user?.totpEnabled ?? false)

  // Setup flow state
  const [setupStep, setSetupStep] = useState<SetupStep>("idle")
  const [setupData, setSetupData] = useState<Setup2FAResponse | null>(null)
  const [enableCode, setEnableCode] = useState("")
  const [disableCode, setDisableCode] = useState("")
  const [showDisableDialog, setShowDisableDialog] = useState(false)
  const [showSecret, setShowSecret] = useState(false)
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null)

  const setup2FA = useSetup2FA()
  const enable2FA = useEnable2FA()
  const disable2FA = useDisable2FA()

  async function handleStartSetup() {
    try {
      const data = await setup2FA.mutateAsync()
      setSetupData(data)
      setSetupStep("qrcode")
      setEnableCode("")
    } catch {
      // toasted by api interceptor
    }
  }

  async function handleEnable() {
    if (!setupData || enableCode.length !== 6) return
    try {
      await enable2FA.mutateAsync({ code: enableCode })
      setSetupStep("recovery")
      setIs2FAEnabled(true)
      // Refresh user state if possible
      if (user && token) {
        login(token, { ...user, totpEnabled: true })
      }
      toast.success(t("twofa.enabledSuccess", "Two-factor authentication enabled"))
    } catch {
      setEnableCode("")
    }
  }

  async function handleDisable() {
    if (!disableCode.trim()) return
    try {
      await disable2FA.mutateAsync({ code: disableCode })
      setShowDisableDialog(false)
      setDisableCode("")
      setIs2FAEnabled(false)
      if (user && token) {
        login(token, { ...user, totpEnabled: false })
      }
      toast.success(t("twofa.disabledSuccess", "Two-factor authentication disabled"))
    } catch {
      setDisableCode("")
    }
  }

  async function copyToClipboard(text: string, index: number) {
    await navigator.clipboard.writeText(text)
    setCopiedIndex(index)
    setTimeout(() => setCopiedIndex(null), 2000)
  }

  async function copyAllCodes() {
    if (!setupData) return
    await navigator.clipboard.writeText(setupData.recoveryCodes.join("\n"))
    toast.success(t("twofa.codesCopied", "Recovery codes copied to clipboard"))
  }

  function handleDoneWithRecovery() {
    setSetupStep("idle")
    setSetupData(null)
    setEnableCode("")
  }

  // ---- Render ----

  return (
    <div className="container max-w-2xl py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("settings.security", "Security")}</h1>
        <p className="text-muted-foreground mt-1">
          {t("settings.securityDescription", "Manage your account security settings")}
        </p>
      </div>

      <Separator />

      {/* 2FA section */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              {is2FAEnabled ? (
                <ShieldCheck className="size-5 text-green-500" />
              ) : (
                <ShieldOff className="size-5 text-muted-foreground" />
              )}
              <div>
                <CardTitle className="text-base">
                  {t("twofa.title", "Two-Factor Authentication")}
                </CardTitle>
                <CardDescription>
                  {t("twofa.settingsDescription", "Add an extra layer of security using a TOTP authenticator app")}
                </CardDescription>
              </div>
            </div>
            <Badge variant={is2FAEnabled ? "default" : "secondary"}>
              {is2FAEnabled
                ? t("twofa.statusEnabled", "Enabled")
                : t("twofa.statusDisabled", "Disabled")}
            </Badge>
          </div>
        </CardHeader>

        <CardContent className="space-y-4">
          {/* Idle — offer to set up or disable */}
          {setupStep === "idle" && (
            <div className="flex gap-3">
              {!is2FAEnabled ? (
                <Button onClick={handleStartSetup} disabled={setup2FA.isPending}>
                  {setup2FA.isPending && <Loader2 className="animate-spin" />}
                  {t("twofa.enable", "Enable 2FA")}
                </Button>
              ) : (
                <Button
                  variant="destructive"
                  onClick={() => setShowDisableDialog(true)}
                >
                  {t("twofa.disable", "Disable 2FA")}
                </Button>
              )}
            </div>
          )}

          {/* QR code scan step */}
          {setupStep === "qrcode" && setupData && (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                {t("twofa.setupStep1", "Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)")}
              </p>

              {/* QR code image via a free QR API */}
              <div className="flex justify-center">
                <img
                  src={`https://api.qrserver.com/v1/create-qr-code/?data=${encodeURIComponent(setupData.otpauthUrl)}&size=180x180&margin=10`}
                  alt="TOTP QR Code"
                  className="rounded-md border p-1"
                  width={180}
                  height={180}
                />
              </div>

              {/* Manual entry secret */}
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">
                  {t("twofa.orEnterManually", "Or enter this key manually:")}
                </p>
                <div className="flex items-center gap-2">
                  <code className="flex-1 rounded bg-muted px-2 py-1 text-xs font-mono break-all">
                    {showSecret ? setupData.secret : setupData.secret.replace(/./g, "•")}
                  </code>
                  <Button
                    size="icon"
                    variant="ghost"
                    onClick={() => setShowSecret((v) => !v)}
                    title={showSecret ? t("twofa.hideSecret") : t("twofa.showSecret")}
                  >
                    {showSecret ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </Button>
                  <Button
                    size="icon"
                    variant="ghost"
                    onClick={() => copyToClipboard(setupData.secret, -1)}
                    title={t("twofa.copySecret")}
                  >
                    {copiedIndex === -1 ? (
                      <CheckCircle className="size-4 text-green-500" />
                    ) : (
                      <Copy className="size-4" />
                    )}
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="enableCode">
                  {t("twofa.enterCode", "Enter the 6-digit code to confirm")}
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="enableCode"
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    placeholder={t("twofa.codePlaceholder", "Enter 6-digit code")}
                    value={enableCode}
                    onChange={(e) => setEnableCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                    maxLength={6}
                    className="text-center tracking-widest text-lg w-36"
                    autoFocus
                  />
                  <Button
                    onClick={handleEnable}
                    disabled={enable2FA.isPending || enableCode.length !== 6}
                  >
                    {enable2FA.isPending && <Loader2 className="animate-spin" />}
                    {t("twofa.verify", "Verify")}
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => { setSetupStep("idle"); setSetupData(null) }}
                  >
                    {t("common.cancel")}
                  </Button>
                </div>
              </div>
            </div>
          )}

          {/* Recovery codes step */}
          {setupStep === "recovery" && setupData && (
            <div className="space-y-4">
              <div className="rounded-md border border-amber-300 bg-amber-50 dark:bg-amber-950/20 p-3 text-sm text-amber-800 dark:text-amber-300">
                {t("twofa.recoveryWarning", "Save these recovery codes in a safe place. Each code can only be used once.")}
              </div>

              <div className="grid grid-cols-2 gap-2">
                {setupData.recoveryCodes.map((code, i) => (
                  <div
                    key={code}
                    className="flex items-center justify-between rounded border px-3 py-1.5 font-mono text-sm"
                  >
                    <span>{code}</span>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="size-6"
                      onClick={() => copyToClipboard(code, i)}
                    >
                      {copiedIndex === i ? (
                        <CheckCircle className="size-3 text-green-500" />
                      ) : (
                        <Copy className="size-3" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>

              <div className="flex gap-3">
                <Button variant="outline" onClick={copyAllCodes}>
                  <Copy className="size-4" />
                  {t("twofa.copyAll", "Copy All")}
                </Button>
                <Button onClick={handleDoneWithRecovery}>
                  {t("twofa.doneSaved", "I've saved these codes")}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <PublicKeyCredentialsCard />

      {/* Change password */}
      <ChangePasswordCard />

      {/* Disable 2FA confirmation dialog */}
      <Dialog open={showDisableDialog} onOpenChange={setShowDisableDialog}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{t("twofa.disableConfirmTitle", "Disable Two-Factor Authentication")}</DialogTitle>
            <DialogDescription>
              {t("twofa.disableConfirmDescription", "Enter your current authenticator code to confirm.")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="disableCode">{t("twofa.code", "Verification Code")}</Label>
              <Input
                id="disableCode"
                type="text"
                inputMode="numeric"
                pattern="[0-9]*"
                placeholder={t("twofa.codePlaceholder", "Enter 6-digit code")}
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                maxLength={6}
                className="text-center tracking-widest text-lg"
                autoFocus
              />
            </div>
            <div className="flex justify-end gap-3">
              <Button
                variant="ghost"
                onClick={() => { setShowDisableDialog(false); setDisableCode("") }}
              >
                {t("common.cancel")}
              </Button>
              <Button
                variant="destructive"
                onClick={handleDisable}
                disabled={disable2FA.isPending || disableCode.length !== 6}
              >
                {disable2FA.isPending && <Loader2 className="animate-spin" />}
                {t("twofa.disable", "Disable 2FA")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
