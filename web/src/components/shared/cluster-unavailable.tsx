import { useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { RefreshCw, ServerOff, Trash2 } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { Cluster } from "@/hooks/use-cluster"
import api from "@/lib/api"

interface ClusterUnavailableProps {
  cluster: Cluster
  onCheckAgain: () => void | Promise<unknown>
  canRemove?: boolean
}

export function ClusterUnavailable({
  cluster,
  onCheckAgain,
  canRemove = true,
}: ClusterUnavailableProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [isChecking, setIsChecking] = useState(false)
  const [isRemoving, setIsRemoving] = useState(false)

  const checkAgain = async () => {
    setIsChecking(true)
    try {
      await onCheckAgain()
    } finally {
      setIsChecking(false)
    }
  }

  const removeCluster = async () => {
    setIsRemoving(true)
    try {
      await api.delete(`/clusters/${cluster.id}`)
      await queryClient.cancelQueries({ queryKey: ["overview", String(cluster.id)] })
      queryClient.removeQueries({ queryKey: ["overview", String(cluster.id)] })
      await queryClient.invalidateQueries({ queryKey: ["clusters"] })
      setConfirmOpen(false)
      toast.success(t("cluster.remove_success", { name: cluster.name }))
    } catch {
      // The shared API client already reports explicit user actions.
    } finally {
      setIsRemoving(false)
    }
  }

  return (
    <>
      <section className="flex min-h-[320px] flex-1 items-center justify-center border-y border-destructive/20 bg-destructive/[0.03] px-6 py-12 text-center">
        <div className="flex max-w-lg flex-col items-center">
          <div className="mb-4 flex size-12 items-center justify-center rounded-full bg-destructive/10">
            <ServerOff className="size-6 text-destructive" />
          </div>
          <h2 className="text-lg font-semibold">{t("cluster.unavailable_title")}</h2>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            {t("cluster.unavailable_desc", { name: cluster.name })}
          </p>
          {cluster.apiServer && (
            <code className="mt-3 max-w-full break-all rounded bg-muted px-2 py-1 text-xs text-muted-foreground">
              {cluster.apiServer}
            </code>
          )}
          <div className="mt-6 flex flex-wrap justify-center gap-2">
            <Button variant="outline" onClick={() => void checkAgain()} disabled={isChecking}>
              <RefreshCw className={`size-4 ${isChecking ? "animate-spin" : ""}`} />
              {t("cluster.check_again")}
            </Button>
            {canRemove && (
              <Button variant="destructive" onClick={() => setConfirmOpen(true)}>
                <Trash2 className="size-4" />
                {t("cluster.remove")}
              </Button>
            )}
          </div>
        </div>
      </section>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("cluster.remove_title")}</DialogTitle>
            <DialogDescription>
              {t("cluster.remove_desc", { name: cluster.name })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmOpen(false)} disabled={isRemoving}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={() => void removeCluster()} disabled={isRemoving}>
              {isRemoving ? t("common.loading") : t("cluster.remove")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
