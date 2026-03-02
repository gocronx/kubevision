import { useQuery } from "@tanstack/react-query"
import { useTranslation } from "react-i18next"
import { Box, Server, Network, Monitor, AlertCircle, Plus } from "lucide-react"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Button } from "@/components/ui/button"
import type { LucideIcon } from "lucide-react"
import { useCluster } from "@/hooks/use-cluster"
import { AddClusterDialog } from "@/components/shared/add-cluster-dialog"
import { useState } from "react"
import api from "@/lib/api"

// ---------------------------------------------------------------------------
// API types
// ---------------------------------------------------------------------------

interface OverviewData {
  pods: number
  deployments: number
  services: number
  nodes: number
}

// ---------------------------------------------------------------------------
// Data fetching hook
// ---------------------------------------------------------------------------

function useOverview(clusterID: string) {
  return useQuery<OverviewData>({
    queryKey: ["overview", clusterID],
    queryFn: async () => {
      const res = await api.get(`/clusters/${clusterID}/overview`)
      return res as unknown as OverviewData
    },
    enabled: !!clusterID,
    refetchInterval: 30_000,
  })
}

// ---------------------------------------------------------------------------
// StatCard component
// ---------------------------------------------------------------------------

interface StatCardProps {
  title: string
  icon: LucideIcon
  value?: number
  isLoading: boolean
}

function StatCard({ title, icon: Icon, value, isLoading }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {isLoading || value === undefined ? (
          <Skeleton className="h-8 w-20" />
        ) : (
          <div className="text-3xl font-bold">{value}</div>
        )}
      </CardContent>
    </Card>
  )
}

// ---------------------------------------------------------------------------
// Loading skeleton for the whole page
// ---------------------------------------------------------------------------

function OverviewSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <Card key={i}>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="size-4" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-8 w-20" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page component
// ---------------------------------------------------------------------------

export function OverviewPage() {
  const { t } = useTranslation()
  const { currentCluster, clusters, isLoading: clustersLoading } = useCluster()
  const { data, isLoading, isError, error } = useOverview(currentCluster)
  const [showAddCluster, setShowAddCluster] = useState(false)

  const noClusters = !clustersLoading && clusters.length === 0

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-bold tracking-tight">{t("overview.title")}</h1>

      {clustersLoading ? (
        <OverviewSkeleton />
      ) : noClusters ? (
        <>
          <Card>
            <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
              <div className="rounded-full bg-muted p-4">
                <Server className="size-8 text-muted-foreground" />
              </div>
              <div>
                <h2 className="text-lg font-semibold">{t("overview.no_cluster")}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{t("overview.no_cluster_desc")}</p>
              </div>
              <Button onClick={() => setShowAddCluster(true)} className="mt-2">
                <Plus className="mr-2 size-4" />
                {t("overview.add_cluster")}
              </Button>
            </CardContent>
          </Card>
          <AddClusterDialog open={showAddCluster} onOpenChange={setShowAddCluster} />
        </>
      ) : (
        <>
          {isError && (
            <div className="flex items-center gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
              <AlertCircle className="size-4 shrink-0" />
              <span>
                {error instanceof Error
                  ? error.message
                  : t("overview.error")}
              </span>
            </div>
          )}

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title={t("overview.pods")}
              icon={Box}
              value={data?.pods}
              isLoading={isLoading}
            />
            <StatCard
              title={t("overview.deployments")}
              icon={Server}
              value={data?.deployments}
              isLoading={isLoading}
            />
            <StatCard
              title={t("overview.services")}
              icon={Network}
              value={data?.services}
              isLoading={isLoading}
            />
            <StatCard
              title={t("overview.nodes")}
              icon={Monitor}
              value={data?.nodes}
              isLoading={isLoading}
            />
          </div>
        </>
      )}
    </div>
  )
}
