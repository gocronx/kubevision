import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import {
  ClipboardList,
  Key,
  Plus,
  Trash2,
  Copy,
  Check,
  RefreshCw,
  Eye,
  EyeOff,
} from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"
import { useAuditLogs, type AuditLog, type AuditLogFilter } from "@/hooks/use-audit"
import { useApiKeys, useGenerateApiKey, useRevokeApiKey, type APIKey, type GeneratedAPIKey } from "@/hooks/use-apikeys"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"

// ---------------------------------------------------------------------------
// Audit Logs Tab
// ---------------------------------------------------------------------------

function AuditLogsTab() {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<AuditLogFilter>({ limit: 50, offset: 0 })
  const [actionFilter, setActionFilter] = useState("")
  const [clusterFilter, setClusterFilter] = useState("")
  const { data, isLoading, refetch } = useAuditLogs(filter)

  const items = data?.items ?? []

  function applyFilters() {
    setFilter((prev) => ({
      ...prev,
      action: actionFilter || undefined,
      cluster: clusterFilter || undefined,
      offset: 0,
    }))
  }

  function prevPage() {
    setFilter((prev) => ({ ...prev, offset: Math.max(0, (prev.offset ?? 0) - (prev.limit ?? 50)) }))
  }

  function nextPage() {
    setFilter((prev) => ({ ...prev, offset: (prev.offset ?? 0) + (prev.limit ?? 50) }))
  }

  function statusColor(code: number) {
    if (code >= 200 && code < 300) return "default"
    if (code >= 400) return "destructive"
    return "secondary"
  }

  return (
    <div className="space-y-4">
      {/* Filters */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-end">
            <div className="flex flex-col gap-1 sm:w-auto">
              <Label className="text-xs">{t("audit.filterAction")}</Label>
              <Input
                className="h-8 w-full sm:w-36"
                placeholder={t("audit.actionPlaceholder")}
                value={actionFilter}
                onChange={(e) => setActionFilter(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1 sm:w-auto">
              <Label className="text-xs">{t("audit.filterCluster")}</Label>
              <Input
                className="h-8 w-full sm:w-36"
                placeholder={t("audit.clusterPlaceholder")}
                value={clusterFilter}
                onChange={(e) => setClusterFilter(e.target.value)}
              />
            </div>
            <Button size="sm" onClick={applyFilters}>{t("common.search")}</Button>
            <Button size="sm" variant="outline" onClick={() => refetch()}>
              <RefreshCw className="size-3.5" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Table */}
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full min-w-[980px] text-sm">
          <thead className="bg-muted/50">
            <tr className="border-b">
              <th className="px-3 py-2 text-left font-medium">{t("audit.time")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.user")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.action")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.source")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.resource")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.cluster")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.status")}</th>
              <th className="px-3 py-2 text-left font-medium">{t("audit.duration")}</th>
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i} className="border-b">
                  {Array.from({ length: 8 }).map((_, j) => (
                    <td key={j} className="px-3 py-2">
                      <Skeleton className="h-4 w-full" />
                    </td>
                  ))}
                </tr>
              ))
            ) : items.length === 0 ? (
              <tr>
                <td colSpan={8} className="px-3 py-8 text-center text-muted-foreground">
                  {t("common.noData")}
                </td>
              </tr>
            ) : (
              items.map((log: AuditLog) => (
                <tr key={log.id} className="border-b hover:bg-muted/30 transition-colors">
                  <td className="px-3 py-2 text-xs text-muted-foreground whitespace-nowrap">
                    {new Date(log.createdAt).toLocaleString()}
                  </td>
                  <td className="px-3 py-2 font-medium">{log.username || String(log.userId)}</td>
                  <td className="px-3 py-2">
                    <Badge variant="outline" className="text-xs">{log.action}</Badge>
                  </td>
                  <td className="px-3 py-2 text-xs">
                    <div>{log.source === "ai-assistant" ? t("audit.sourceAI") : t("audit.sourceHTTP")}</div>
                    {log.tool ? <div className="text-muted-foreground">{log.tool}</div> : null}
                  </td>
                  <td className="px-3 py-2 text-muted-foreground">
                    {log.resource}{log.name ? `/${log.name}` : ""}
                  </td>
                  <td className="px-3 py-2 text-xs">{log.cluster || "-"}</td>
                  <td className="px-3 py-2">
                    <Badge variant={statusColor(log.statusCode)} className="text-xs">
                      {log.outcome || log.statusCode}
                    </Badge>
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">{log.durationMs}ms</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex flex-wrap items-center justify-end gap-2">
        <Button
          size="sm"
          variant="outline"
          onClick={prevPage}
          disabled={(filter.offset ?? 0) === 0}
        >
          {t("audit.prev")}
        </Button>
        <span className="text-sm text-muted-foreground">
          {t("audit.page", { page: Math.floor((filter.offset ?? 0) / (filter.limit ?? 50)) + 1 })}
        </span>
        <Button
          size="sm"
          variant="outline"
          onClick={nextPage}
          disabled={items.length < (filter.limit ?? 50)}
        >
          {t("audit.next")}
        </Button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// API Keys Tab
// ---------------------------------------------------------------------------

function APIKeysTab() {
  const { t } = useTranslation()
  const { data: keys = [], isLoading } = useApiKeys()
  const generateKey = useGenerateApiKey()
  const revokeKey = useRevokeApiKey()

  const [showGenDialog, setShowGenDialog] = useState(false)
  const [newKeyName, setNewKeyName] = useState("")
  const [generatedKey, setGeneratedKey] = useState<GeneratedAPIKey | null>(null)
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)
  const [revokeTarget, setRevokeTarget] = useState<APIKey | null>(null)
  const copiedTimerRef = useRef<number | null>(null)

  useEffect(() => () => {
    if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
  }, [])

  async function handleGenerate() {
    if (!newKeyName.trim()) return
    try {
      const result = await generateKey.mutateAsync({ name: newKeyName.trim() })
      setGeneratedKey(result)
      setNewKeyName("")
      setShowKey(false)
      setCopied(false)
    } catch {
      // toasted by api interceptor
    }
  }

  async function handleRevoke(key: APIKey) {
    try {
      await revokeKey.mutateAsync(key.id)
      toast.success(t("apikeys.revokedToast"))
    } catch {
      // toasted
    }
    setRevokeTarget(null)
  }

  function copyKey(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true)
      if (copiedTimerRef.current !== null) window.clearTimeout(copiedTimerRef.current)
      copiedTimerRef.current = window.setTimeout(() => {
        copiedTimerRef.current = null
        setCopied(false)
      }, 2000)
    })
  }

  function closeGenDialog() {
    setShowGenDialog(false)
    setGeneratedKey(null)
    setNewKeyName("")
    setShowKey(false)
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">{t("apikeys.description")}</p>
        <Button size="sm" onClick={() => setShowGenDialog(true)}>
          <Plus className="size-3.5 mr-1" />
          {t("apikeys.generate")}
        </Button>
      </div>

      {/* Keys list */}
      <div className="max-w-full overflow-x-auto rounded-md border">
        {isLoading ? (
          <div className="p-4 space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : keys.length === 0 ? (
          <div className="px-4 py-8 text-center text-muted-foreground text-sm">
            {t("apikeys.empty")}
          </div>
        ) : (
          <table className="w-full min-w-[40rem] text-sm">
            <thead className="bg-muted/50">
              <tr className="border-b">
                <th className="px-3 py-2 text-left font-medium">{t("common.name")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("apikeys.prefix")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("apikeys.created")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("apikeys.expires")}</th>
                <th className="px-3 py-2 text-left font-medium">{t("common.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key: APIKey) => (
                <tr key={key.id} className="border-b hover:bg-muted/30 transition-colors">
                  <td className="px-3 py-2 font-medium">{key.name}</td>
                  <td className="px-3 py-2">
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">{key.keyPrefix}...</code>
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">
                    {new Date(key.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-3 py-2 text-xs text-muted-foreground">
                    {key.expiresAt
                      ? new Date(key.expiresAt).toLocaleDateString()
                      : <span className="italic">{t("apikeys.noExpiry")}</span>
                    }
                  </td>
                  <td className="px-3 py-2">
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => setRevokeTarget(key)}
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Generate dialog */}
      <Dialog open={showGenDialog} onOpenChange={closeGenDialog}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t("apikeys.generate")}</DialogTitle>
            <DialogDescription>{t("apikeys.generateDescription")}</DialogDescription>
          </DialogHeader>

          {generatedKey ? (
            <div className="space-y-4">
              <div className="rounded-md bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-3 text-sm text-amber-800 dark:text-amber-200">
                {t("apikeys.showOnce")}
              </div>
              <div className="space-y-1">
                <Label className="text-xs">{t("apikeys.yourKey")}</Label>
                <div className="flex items-center gap-2">
                  <code className="flex-1 text-xs bg-muted rounded px-2 py-1.5 font-mono break-all">
                    {showKey ? generatedKey.key : generatedKey.key.replace(/./g, "*")}
                  </code>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setShowKey((s) => !s)}
                  >
                    {showKey ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => copyKey(generatedKey.key)}
                  >
                    {copied ? <Check className="size-3.5 text-green-500" /> : <Copy className="size-3.5" />}
                  </Button>
                </div>
              </div>
              <Button className="w-full" onClick={closeGenDialog}>
                {t("common.confirm")}
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-1">
                <Label htmlFor="key-name">{t("common.name")}</Label>
                <Input
                  id="key-name"
                  placeholder={t("apikeys.namePlaceholder")}
                  value={newKeyName}
                  onChange={(e) => setNewKeyName(e.target.value)}
                />
              </div>
              <div className="flex gap-2 justify-end">
                <Button variant="outline" onClick={closeGenDialog}>{t("common.cancel")}</Button>
                <Button
                  onClick={handleGenerate}
                  disabled={!newKeyName.trim() || generateKey.isPending}
                >
                  {generateKey.isPending ? t("common.loading") : t("apikeys.generate")}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Revoke confirm dialog */}
      <Dialog open={revokeTarget !== null} onOpenChange={(open) => !open && setRevokeTarget(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{t("apikeys.revokeTitle")}</DialogTitle>
            <DialogDescription>
              {t("apikeys.revokeDescription", { name: revokeTarget?.name ?? "" })}
            </DialogDescription>
          </DialogHeader>
          <div className="flex gap-2 justify-end">
            <Button variant="outline" onClick={() => setRevokeTarget(null)}>{t("common.cancel")}</Button>
            <Button
              variant="destructive"
              onClick={() => revokeTarget && handleRevoke(revokeTarget)}
              disabled={revokeKey.isPending}
            >
              {t("apikeys.revoke")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Admin Page
// ---------------------------------------------------------------------------

export function AdminPage() {
  const { t } = useTranslation()
  const { user } = useAuth()

  // Only super-admin and admin roles can access this page.
  if (!canAccessAdmin(user?.role ?? "")) {
    return (
      <div className="flex items-center justify-center h-full py-16 text-muted-foreground">
        {t("common.forbidden", "Access denied")}
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">{t("admin.title")}</h1>
        <p className="text-sm text-muted-foreground mt-1">{t("admin.description")}</p>
      </div>

      <Tabs defaultValue="audit-logs">
        <TabsList>
          <TabsTrigger value="audit-logs">
            <ClipboardList className="size-3.5 mr-1.5" />
            {t("audit.title")}
          </TabsTrigger>
          <TabsTrigger value="api-keys">
            <Key className="size-3.5 mr-1.5" />
            {t("apikeys.title")}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="audit-logs" className="mt-4">
          <AuditLogsTab />
        </TabsContent>

        <TabsContent value="api-keys" className="mt-4">
          <APIKeysTab />
        </TabsContent>
      </Tabs>
    </div>
  )
}
