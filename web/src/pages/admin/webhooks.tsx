import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Plus, Pencil, Trash2, FlaskConical, Webhook } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import {
  useWebhooks,
  useCreateWebhook,
  useUpdateWebhook,
  useDeleteWebhook,
  useTestWebhook,
  type Webhook as WebhookModel,
  type WebhookPayload,
} from "@/hooks/use-webhooks"

// --------------------------------------------------------------------------
// Available event types (matches informer.ResourceEvent.Type)
// --------------------------------------------------------------------------

const EVENT_OPTIONS = ["ADDED", "MODIFIED", "DELETED", "test"]
const RESOURCE_OPTIONS = [
  "pods", "deployments", "statefulsets", "daemonsets",
  "services", "ingresses", "configmaps", "secrets",
  "persistentvolumeclaims", "nodes", "namespaces",
]

// --------------------------------------------------------------------------
// Webhook form dialog
// --------------------------------------------------------------------------

interface WebhookFormProps {
  open: boolean
  onClose: () => void
  initial?: WebhookModel | null
}

const DEFAULT_FORM: WebhookPayload = {
  name: "",
  url: "",
  secret: "",
  events: [],
  clusters: [],
  resources: [],
  isActive: true,
}

function WebhookFormDialog({ open, onClose, initial }: WebhookFormProps) {
  const { t } = useTranslation()
  const createMutation = useCreateWebhook()
  const updateMutation = useUpdateWebhook()

  const [form, setForm] = useState<WebhookPayload>(() =>
    initial
      ? {
          name: initial.name,
          url: initial.url,
          secret: "",
          events: initial.events,
          clusters: initial.clusters,
          resources: initial.resources,
          isActive: initial.isActive,
        }
      : DEFAULT_FORM
  )

  // Reset when opening.
  const handleOpenChange = (o: boolean) => {
    if (o) {
      setForm(
        initial
          ? {
              name: initial.name,
              url: initial.url,
              secret: "",
              events: initial.events,
              clusters: initial.clusters,
              resources: initial.resources,
              isActive: initial.isActive,
            }
          : DEFAULT_FORM
      )
    } else {
      onClose()
    }
  }

  function toggleItem(field: "events" | "resources", item: string) {
    setForm((prev) => {
      const list = prev[field] as string[]
      return {
        ...prev,
        [field]: list.includes(item) ? list.filter((x) => x !== item) : [...list, item],
      }
    })
  }

  const handleSubmit = async () => {
    if (initial) {
      await updateMutation.mutateAsync({ id: initial.id, payload: form })
    } else {
      await createMutation.mutateAsync(form)
    }
    onClose()
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {initial ? t("webhook.edit") : t("webhook.create")}
          </DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {/* Name */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="wh-name">{t("common.name")}</Label>
            <Input
              id="wh-name"
              value={form.name}
              onChange={(e) => setForm((p) => ({ ...p, name: e.target.value }))}
              placeholder="My Webhook"
            />
          </div>

          {/* URL */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="wh-url">{t("webhook.url")}</Label>
            <Input
              id="wh-url"
              value={form.url}
              onChange={(e) => setForm((p) => ({ ...p, url: e.target.value }))}
              placeholder="https://example.com/hook"
            />
          </div>

          {/* Secret */}
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="wh-secret">{t("webhook.secret")}</Label>
            <Input
              id="wh-secret"
              type="password"
              value={form.secret}
              onChange={(e) => setForm((p) => ({ ...p, secret: e.target.value }))}
              placeholder={initial ? t("webhook.secretPlaceholder") : ""}
            />
          </div>

          {/* Events */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("webhook.events")}</Label>
            <div className="flex flex-wrap gap-2">
              {EVENT_OPTIONS.map((ev) => (
                <Badge
                  key={ev}
                  variant={form.events.includes(ev) ? "default" : "outline"}
                  className="cursor-pointer select-none"
                  onClick={() => toggleItem("events", ev)}
                >
                  {ev}
                </Badge>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">{t("webhook.eventsHint")}</p>
          </div>

          {/* Resources */}
          <div className="flex flex-col gap-1.5">
            <Label>{t("webhook.resources")}</Label>
            <div className="flex flex-wrap gap-1.5">
              {RESOURCE_OPTIONS.map((r) => (
                <Badge
                  key={r}
                  variant={form.resources.includes(r) ? "default" : "outline"}
                  className="cursor-pointer select-none text-xs"
                  onClick={() => toggleItem("resources", r)}
                >
                  {r}
                </Badge>
              ))}
            </div>
            <p className="text-xs text-muted-foreground">{t("webhook.resourcesHint")}</p>
          </div>

          {/* Active toggle */}
          <div className="flex items-center gap-2">
            <input
              id="wh-active"
              type="checkbox"
              checked={form.isActive}
              onChange={(e) => setForm((p) => ({ ...p, isActive: e.target.checked }))}
              className="size-4 accent-primary"
            />
            <Label htmlFor="wh-active">{t("webhook.active")}</Label>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={isPending || !form.name || !form.url}>
            {t("common.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// --------------------------------------------------------------------------
// Webhooks page
// --------------------------------------------------------------------------

export function WebhooksPage() {
  const { t } = useTranslation()
  const { data: webhooks = [], isLoading } = useWebhooks()
  const deleteMutation = useDeleteWebhook()
  const testMutation = useTestWebhook()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<WebhookModel | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<WebhookModel | null>(null)

  const handleEdit = (wh: WebhookModel) => {
    setEditing(wh)
    setFormOpen(true)
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    await deleteMutation.mutateAsync(deleteTarget.id)
    setDeleteTarget(null)
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("webhook.title")}</h1>
          <p className="text-muted-foreground text-sm">{t("webhook.description")}</p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setEditing(null)
            setFormOpen(true)
          }}
          className="self-start gap-1.5"
        >
          <Plus className="size-4" />
          {t("webhook.create")}
        </Button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      ) : webhooks.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-muted-foreground gap-2">
          <Webhook className="size-10 opacity-30" />
          <p>{t("webhook.empty")}</p>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {webhooks.map((wh) => (
            <Card key={wh.id}>
              <CardHeader className="pb-2">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <CardTitle className="text-base">{wh.name}</CardTitle>
                    <Badge variant={wh.isActive ? "default" : "secondary"}>
                      {wh.isActive ? t("webhook.active") : t("webhook.inactive")}
                    </Badge>
                  </div>
                  <div className="flex flex-wrap items-center gap-1.5">
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 gap-1 text-xs"
                      onClick={() => testMutation.mutate(wh.id)}
                      disabled={testMutation.isPending}
                    >
                      <FlaskConical className="size-3" />
                      {t("webhook.test")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 gap-1 text-xs"
                      onClick={() => handleEdit(wh)}
                    >
                      <Pencil className="size-3" />
                      {t("common.edit")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      className="h-7 gap-1 text-xs text-destructive hover:text-destructive"
                      onClick={() => setDeleteTarget(wh)}
                    >
                      <Trash2 className="size-3" />
                      {t("common.delete")}
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pb-3">
                <p className="font-mono text-xs text-muted-foreground truncate">{wh.url}</p>
                <div className="mt-2 flex flex-wrap gap-1">
                  {wh.events.map((ev) => (
                    <Badge key={ev} variant="outline" className="text-xs">
                      {ev}
                    </Badge>
                  ))}
                  {wh.resources.map((r) => (
                    <Badge key={r} variant="secondary" className="text-xs">
                      {r}
                    </Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create / Edit dialog */}
      <WebhookFormDialog
        open={formOpen}
        onClose={() => {
          setFormOpen(false)
          setEditing(null)
        }}
        initial={editing}
      />

      {/* Delete confirmation dialog */}
      <Dialog open={!!deleteTarget} onOpenChange={(o) => !o && setDeleteTarget(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>{t("webhook.deleteTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("webhook.deleteConfirm", { name: deleteTarget?.name })}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteMutation.isPending}
            >
              {t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
