import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Upload, Loader2 } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import api from "@/lib/api"
import { preventSubmitWhileComposing } from "@/lib/form-events"
import { readFileAsText } from "@/lib/read-file"

interface AddClusterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type ClusterAuthType = "kubeconfig" | "in-cluster"

interface AddClusterPayload {
  name: string
  authType: ClusterAuthType
  kubeconfig?: string
}

export function AddClusterDialog({ open, onOpenChange }: AddClusterDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState("")
  const [authType, setAuthType] = useState<ClusterAuthType>("kubeconfig")
  const [kubeconfig, setKubeconfig] = useState("")
  const [kubeconfigFileName, setKubeconfigFileName] = useState("")

  const mutation = useMutation({
    mutationFn: (payload: AddClusterPayload) => api.post("/clusters", payload),
    onSuccess: () => {
      toast.success(t("cluster.add_success"))
      queryClient.invalidateQueries({ queryKey: ["clusters"] })
      queryClient.invalidateQueries({ queryKey: ["overview"] })
      resetAndClose()
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : t("cluster.add_error"))
    },
  })

  useEffect(() => {
    if (open) resetForm()
  }, [open])

  function resetForm() {
    setName("")
    setAuthType("kubeconfig")
    setKubeconfig("")
    setKubeconfigFileName("")
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

  function resetAndClose() {
    resetForm()
    onOpenChange(false)
  }

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const trimmedName = name.trim()
    if (!trimmedName) return
    if (authType === "kubeconfig" && !kubeconfig.trim()) {
      toast.error(t("cluster.kubeconfig_required"))
      return
    }

    mutation.mutate({
      name: trimmedName,
      authType,
      kubeconfig: authType === "kubeconfig" ? kubeconfig : undefined,
    })
  }

  async function handleFileUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    try {
      const text = await readFileAsText(file)
      if (!text.trim()) {
        toast.error(t("cluster.file_empty"))
        return
      }
      setKubeconfig(text)
      setKubeconfigFileName(file.name)
    } catch {
      toast.error(t("cluster.file_read_error"))
    } finally {
      e.target.value = ""
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => {
      if (!v && !mutation.isPending) resetAndClose()
      else if (v) onOpenChange(v)
    }}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("cluster.add_title")}</DialogTitle>
          <DialogDescription>{t("cluster.add_desc")}</DialogDescription>
        </DialogHeader>

        <form
          className="space-y-4"
          onSubmit={handleSubmit}
          onKeyDownCapture={preventSubmitWhileComposing}
        >
          <div className="space-y-1.5">
            <Label htmlFor="cluster-name">
              {t("cluster.name")}
              <span aria-hidden="true"> *</span>
            </Label>
            <Input
              id="cluster-name"
              required
              aria-required="true"
              placeholder="e.g. production"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>

          <fieldset className="space-y-1.5">
            <legend className="text-sm font-medium leading-none">{t("cluster.auth_type")}</legend>
            <div className="grid gap-2 sm:grid-cols-2">
              {(["kubeconfig", "in-cluster"] as const).map((value) => (
                <label
                  key={value}
                  className={`cursor-pointer rounded-md border px-3 py-2 text-sm transition-colors ${
                    authType === value
                      ? "border-primary bg-primary/5 text-foreground"
                      : "border-input text-muted-foreground hover:bg-accent"
                  }`}
                >
                  <span className="flex items-center gap-2 font-medium">
                    <input
                      type="radio"
                      name="cluster-auth-type"
                      value={value}
                      checked={authType === value}
                      onChange={() => setAuthType(value)}
                      className="size-4 accent-primary"
                    />
                    {value === "kubeconfig" ? "Kubeconfig" : "In-Cluster"}
                  </span>
                </label>
              ))}
            </div>
            {authType === "in-cluster" && (
              <p className="text-xs text-muted-foreground">{t("cluster.in_cluster_hint")}</p>
            )}
          </fieldset>

          {authType === "kubeconfig" && (
            <div className="space-y-1.5">
              <Label htmlFor="cluster-kubeconfig">
                {t("cluster.kubeconfig")}
                <span aria-hidden="true"> *</span>
              </Label>
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => fileInputRef.current?.click()}
                >
                  <Upload className="mr-1.5 size-3.5" />
                  {t("cluster.upload_file")}
                </Button>
                <input
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  aria-label={t("cluster.upload_file")}
                  onChange={handleFileUpload}
                />
                {kubeconfigFileName && (
                  <span className="min-w-0 truncate text-xs text-muted-foreground" title={kubeconfigFileName}>
                    {t("cluster.file_loaded", { name: kubeconfigFileName })}
                  </span>
                )}
              </div>
              <p className="text-xs text-muted-foreground">{t("cluster.file_hint")}</p>
              <Textarea
                id="cluster-kubeconfig"
                className="min-h-[120px] font-mono text-xs"
                placeholder={t("cluster.kubeconfig_placeholder")}
                value={kubeconfig}
                onChange={(e) => setKubeconfig(e.target.value)}
                required
                aria-required="true"
              />
            </div>
          )}

          <div className="flex flex-col-reverse gap-2 pt-2 sm:flex-row sm:justify-end">
            <Button variant="outline" onClick={resetAndClose} disabled={mutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button
              type="submit"
              disabled={
                !name.trim() ||
                (authType === "kubeconfig" && !kubeconfig.trim()) ||
                mutation.isPending
              }
            >
              {mutation.isPending && <Loader2 className="mr-2 size-4 animate-spin" />}
              {mutation.isPending ? t("common.loading") : t("cluster.add_submit")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
