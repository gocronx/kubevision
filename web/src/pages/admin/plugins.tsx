import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Plug, RefreshCw, CheckCircle, XCircle, Settings } from "lucide-react"
import {
  usePlugins,
  useConfigurePlugin,
  usePluginHealthCheck,
  type PluginConfigPayload,
} from "@/hooks/use-plugins"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

export function PluginsPage() {
  const { t } = useTranslation()
  const { data: plugins, isLoading } = usePlugins()
  const configurePlugin = useConfigurePlugin()
  const healthCheck = usePluginHealthCheck()
  const [editPlugin, setEditPlugin] = useState<string | null>(null)
  const [editConfig, setEditConfig] = useState<Record<string, string>>({})
  const [editEnabled, setEditEnabled] = useState(false)

  const handleConfigure = (name: string) => {
    const plugin = plugins?.find((p) => p.name === name)
    setEditPlugin(name)
    setEditEnabled(plugin?.enabled ?? false)
    setEditConfig({})
  }

  const handleSave = () => {
    if (!editPlugin) return
    const payload: PluginConfigPayload = {
      enabled: editEnabled,
      config: editConfig,
    }
    configurePlugin.mutate(
      { name: editPlugin, payload },
      { onSuccess: () => setEditPlugin(null) }
    )
  }

  const configFields: Record<string, { label: string; key: string }[]> = {
    prometheus: [{ label: "URL", key: "url" }],
    grafana: [
      { label: "URL", key: "url" },
      { label: "Token", key: "token" },
    ],
    argocd: [
      { label: "URL", key: "url" },
      { label: "Token", key: "token" },
    ],
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <RefreshCw className="size-5 animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">{t("plugin.title")}</h1>
        <p className="text-muted-foreground">{t("plugin.description")}</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {plugins?.map((plugin) => (
          <Card key={plugin.name}>
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <CardTitle className="flex items-center gap-2 text-base">
                  <Plug className="size-4" />
                  {plugin.name}
                </CardTitle>
                <Badge variant={plugin.enabled ? "default" : "secondary"}>
                  {plugin.enabled ? t("plugin.enabled") : t("plugin.disabled")}
                </Badge>
              </div>
              <CardDescription>{plugin.description}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-sm">
                  {plugin.enabled && (
                    plugin.healthy ? (
                      <span className="flex items-center gap-1 text-green-600">
                        <CheckCircle className="size-3.5" />
                        {t("plugin.healthy")}
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-destructive">
                        <XCircle className="size-3.5" />
                        {t("plugin.unhealthy")}
                      </span>
                    )
                  )}
                  <span className="text-muted-foreground">v{plugin.version}</span>
                </div>
                <div className="flex gap-1">
                  {plugin.enabled && (
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => healthCheck.mutate(plugin.name)}
                    >
                      <RefreshCw className="size-4" />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => handleConfigure(plugin.name)}
                  >
                    <Settings className="size-4" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Configure Dialog */}
      <Dialog open={!!editPlugin} onOpenChange={(open) => !open && setEditPlugin(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("plugin.configure")} - {editPlugin}</DialogTitle>
            <DialogDescription>{t("plugin.configureDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex items-center justify-between">
              <Label>{t("plugin.enabled")}</Label>
              <Button
                variant={editEnabled ? "default" : "outline"}
                size="sm"
                onClick={() => setEditEnabled(!editEnabled)}
              >
                {editEnabled ? t("plugin.enabled") : t("plugin.disabled")}
              </Button>
            </div>
            {editPlugin && configFields[editPlugin]?.map((field) => (
              <div key={field.key} className="space-y-2">
                <Label>{field.label}</Label>
                <Input
                  value={editConfig[field.key] ?? ""}
                  onChange={(e) =>
                    setEditConfig((prev) => ({ ...prev, [field.key]: e.target.value }))
                  }
                  placeholder={`Enter ${field.label.toLowerCase()}`}
                />
              </div>
            ))}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditPlugin(null)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleSave} disabled={configurePlugin.isPending}>
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
