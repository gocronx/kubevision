import { useRef, useState } from "react"
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

interface AddClusterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddClusterDialog({ open, onOpenChange }: AddClusterDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState("")
  const [displayName, setDisplayName] = useState("")
  const [authType, setAuthType] = useState<"kubeconfig" | "in-cluster">("kubeconfig")
  const [kubeconfig, setKubeconfig] = useState("")

  const mutation = useMutation({
    mutationFn: async () => {
      return api.post("/clusters", {
        name,
        displayName: displayName || undefined,
        authType,
        kubeconfig: authType === "kubeconfig" ? kubeconfig : undefined,
      })
    },
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

  function resetAndClose() {
    setName("")
    setDisplayName("")
    setAuthType("kubeconfig")
    setKubeconfig("")
    onOpenChange(false)
  }

  async function handleFileUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    const text = await file.text()
    setKubeconfig(text)
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

        <div className="space-y-4">
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

          <div className="space-y-1.5">
            <Label htmlFor="cluster-display-name">{t("cluster.display_name")}</Label>
            <Input
              id="cluster-display-name"
              placeholder="e.g. Production Cluster"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <fieldset className="space-y-1.5">
            <legend className="text-sm font-medium leading-none">{t("cluster.auth_type")}</legend>
            <div className="flex gap-2" role="radiogroup" aria-label={t("cluster.auth_type")}>
              <Button
                type="button"
                size="sm"
                role="radio"
                aria-checked={authType === "kubeconfig"}
                variant={authType === "kubeconfig" ? "default" : "outline"}
                onClick={() => setAuthType("kubeconfig")}
              >
                Kubeconfig
              </Button>
              <Button
                type="button"
                size="sm"
                role="radio"
                aria-checked={authType === "in-cluster"}
                variant={authType === "in-cluster" ? "default" : "outline"}
                onClick={() => setAuthType("in-cluster")}
              >
                In-Cluster
              </Button>
            </div>
          </fieldset>

          {authType === "kubeconfig" && (
            <div className="space-y-1.5">
              <Label htmlFor="cluster-kubeconfig">
                {t("cluster.kubeconfig")}
                <span aria-hidden="true"> *</span>
              </Label>
              <div className="flex items-center gap-2">
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
                  accept=".yaml,.yml,.conf,*"
                  className="hidden"
                  onChange={handleFileUpload}
                />
                {kubeconfig && (
                  <span className="text-xs text-muted-foreground">
                    {t("cluster.file_loaded")}
                  </span>
                )}
              </div>
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

          <div className="flex gap-2 justify-end pt-2">
            <Button variant="outline" onClick={resetAndClose} disabled={mutation.isPending}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() => mutation.mutate()}
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
        </div>
      </DialogContent>
    </Dialog>
  )
}
