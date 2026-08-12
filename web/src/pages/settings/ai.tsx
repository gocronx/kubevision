import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Bot, Check, Loader2, RefreshCw, Save, Search } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/stores/auth-store"
import {
  useAIConfig,
  useDiscoverAIModels,
  useUpdateAIConfig,
} from "@/components/ai-chat/use-ai-config"
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
import type { AIConfigView } from "@/components/ai-chat/ai-chat-types"

export function AISettingsPage() {
  const { data: config, isLoading } = useAIConfig()

  if (isLoading || !config) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="size-5 animate-spin text-muted-foreground" />
      </div>
    )
  }
  // Remount the form when the persisted config identity changes so its initial
  // state re-seeds from props without a state-syncing effect.
  return <AISettingsForm config={config} />
}

function AISettingsForm({ config }: { config: AIConfigView }) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const isAdmin = user?.role === "admin" || user?.role === "super-admin"
  const update = useUpdateAIConfig()
  const discoverModels = useDiscoverAIModels()

  const [enabled, setEnabled] = useState(config.enabled)
  const [baseURL, setBaseURL] = useState(config.baseURL)
  const [model, setModel] = useState(config.model)
  const [apiKey, setApiKey] = useState("")
  const [maxTokens, setMaxTokens] = useState(config.maxTokens)
  const [modelSearch, setModelSearch] = useState("")
  const [modelPickerOpen, setModelPickerOpen] = useState(false)

  const models = Array.from(new Set((discoverModels.data ?? []).map((item) => item.id)))
    .filter(Boolean)
    .sort((a, b) => a.localeCompare(b))
  const normalizedSearch = modelSearch.trim().toLowerCase()
  const filteredModels = normalizedSearch
    ? models.filter((item) => item.toLowerCase().includes(normalizedSearch))
    : models

  const discover = () => {
    discoverModels.mutate(
      { baseURL: baseURL.trim(), apiKey: apiKey.trim() },
      {
        onSuccess: (items) => {
          setModelPickerOpen(items.length > 0)
          setModelSearch("")
          if (items.length === 0) toast.info(t("ai.modelsEmpty"))
          else toast.success(t("ai.modelsFound", { count: items.length }))
        },
      }
    )
  }

  const save = () => {
    update.mutate(
      { enabled, baseURL, model, apiKey, maxTokens },
      {
        onSuccess: () => {
          setApiKey("")
          setModelPickerOpen(false)
          setModelSearch("")
          toast.success(t("ai.saved"))
        },
      }
    )
  }

  return (
    <div className="mx-auto w-full max-w-2xl">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Bot className="size-5 text-primary" />
            {t("ai.settingsTitle")}
          </CardTitle>
          <CardDescription>{t("ai.settingsDesc")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={enabled}
              disabled={!isAdmin}
              onChange={(e) => setEnabled(e.target.checked)}
              className="size-4 accent-primary"
            />
            <span className="text-sm font-medium">{t("ai.enable")}</span>
          </label>

          <div className="space-y-2">
            <Label htmlFor="ai-base-url">{t("ai.baseURL")}</Label>
            <Input
              id="ai-base-url"
              value={baseURL}
              disabled={!isAdmin}
              placeholder="https://api.openai.com/v1"
              onChange={(e) => setBaseURL(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("ai.baseURLHint")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ai-key">{t("ai.apiKey")}</Label>
            <Input
              id="ai-key"
              type="password"
              value={apiKey}
              disabled={!isAdmin}
              placeholder={config.hasApiKey ? "••••••••  " + t("ai.keySet") : t("ai.apiKeyPlaceholder")}
              onChange={(e) => setApiKey(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">{t("ai.apiKeyHint")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="ai-model">{t("ai.model")}</Label>
            <div className="flex gap-2">
              <Input
                id="ai-model"
                value={model}
                disabled={!isAdmin}
                placeholder="gpt-4o-mini"
                onChange={(e) => setModel(e.target.value)}
              />
              <Button
                type="button"
                variant="outline"
                disabled={!isAdmin || !baseURL.trim() || (!apiKey.trim() && !config.hasApiKey) || discoverModels.isPending}
                onClick={discover}
              >
                {discoverModels.isPending ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
                {t("ai.discoverModels")}
              </Button>
            </div>
            <p className="text-xs text-muted-foreground">{t("ai.modelHint")}</p>
            {modelPickerOpen && models.length > 0 && (
              <div className="overflow-hidden rounded-md border">
                <div className="relative border-b">
                  <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={modelSearch}
                    onChange={(event) => setModelSearch(event.target.value)}
                    placeholder={t("ai.searchModels")}
                    className="rounded-none border-0 pl-9 shadow-none focus-visible:ring-0"
                  />
                </div>
                <div className="max-h-56 overflow-y-auto p-1" role="listbox" aria-label={t("ai.availableModels")}>
                  {filteredModels.length === 0 ? (
                    <p className="p-4 text-center text-sm text-muted-foreground">{t("ai.noMatchingModels")}</p>
                  ) : filteredModels.map((item) => (
                    <button
                      key={item}
                      type="button"
                      role="option"
                      aria-selected={model === item}
                      onClick={() => {
                        setModel(item)
                        setModelPickerOpen(false)
                        setModelSearch("")
                      }}
                      className="flex w-full items-center gap-2 rounded-sm px-3 py-2 text-left text-sm hover:bg-muted"
                    >
                      <Check className={`size-4 ${model === item ? "opacity-100" : "opacity-0"}`} />
                      <span className="min-w-0 truncate font-mono text-xs">{item}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="ai-max-tokens">{t("ai.maxTokens")}</Label>
            <Input
              id="ai-max-tokens"
              type="number"
              value={maxTokens}
              disabled={!isAdmin}
              onChange={(e) => setMaxTokens(Number(e.target.value))}
            />
          </div>

          {isAdmin ? (
            <Button onClick={save} disabled={update.isPending}>
              {update.isPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Save className="size-4" />
              )}
              {t("common.save")}
            </Button>
          ) : (
            <p className="text-sm text-muted-foreground">{t("ai.adminOnly")}</p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
