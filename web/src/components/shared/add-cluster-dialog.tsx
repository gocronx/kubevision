import { useState } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { Upload } from "lucide-react"
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
import api from "@/lib/api"

interface AddClusterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function AddClusterDialog({ open, onOpenChange }: AddClusterDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
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
    <Dialog open={open} onOpenChange={(v) => !v ? resetAndClose() : onOpenChange(v)}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("cluster.add_title")}</DialogTitle>
          <DialogDescription>{t("cluster.add_desc")}</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="cluster-name">{t("cluster.name")} *</Label>
            <Input
              id="cluster-name"
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

          <div className="space-y-1.5">
            <Label>{t("cluster.auth_type")}</Label>
            <div className="flex gap-2">
              <Button
                type="button"
                size="sm"
                variant={authType === "kubeconfig" ? "default" : "outline"}
                onClick={() => setAuthType("kubeconfig")}
              >
                Kubeconfig
              </Button>
              <Button
                type="button"
                size="sm"
                variant={authType === "in-cluster" ? "default" : "outline"}
                onClick={() => setAuthType("in-cluster")}
              >
                In-Cluster
              </Button>
            </div>
          </div>

          {authType === "kubeconfig" && (
            <div className="space-y-1.5">
              <Label>{t("cluster.kubeconfig")} *</Label>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => document.getElementById("kubeconfig-file")?.click()}
                >
                  <Upload className="mr-1.5 size-3.5" />
                  {t("cluster.upload_file")}
                </Button>
                <input
                  id="kubeconfig-file"
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
              <textarea
                className="mt-1.5 w-full rounded-md border bg-transparent px-3 py-2 text-xs font-mono min-h-[120px] focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder={t("cluster.kubeconfig_placeholder")}
                value={kubeconfig}
                onChange={(e) => setKubeconfig(e.target.value)}
              />
            </div>
          )}

          <div className="flex gap-2 justify-end pt-2">
            <Button variant="outline" onClick={resetAndClose}>
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
              {mutation.isPending ? t("common.loading") : t("cluster.add_submit")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
