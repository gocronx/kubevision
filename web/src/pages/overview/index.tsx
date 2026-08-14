import { useState } from "react"
import { useTranslation } from "react-i18next"
import { AlertCircle, Box, Layers, Monitor, Plus, RefreshCw, Server } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { AddClusterDialog } from "@/components/shared/add-cluster-dialog"
import { ClusterUnavailable } from "@/components/shared/cluster-unavailable"
import { useCluster } from "@/hooks/use-cluster"
import { useAuth } from "@/stores/auth-store"
import { canAccessAdmin } from "@/lib/permissions"
import { RecentEvents, StorageOverview } from "./activity-cards"
import { ResourceUtilization, StatCard } from "./resource-cards"
import { EMPTY_POD_STATUS, EMPTY_RESOURCES, type OverviewData } from "./types"
import { useOverview } from "./use-overview"
import { getSubtitleColor } from "./utils"
import { OverviewSkeleton, PodStatusBar, WorkloadSummary } from "./workload-cards"

const REFRESH_OPTIONS = [
  { value: "0", label: "Off" },
  { value: "5000", label: "5s" },
  { value: "10000", label: "10s" },
  { value: "30000", label: "30s" },
  { value: "60000", label: "60s" },
] as const

export function OverviewPage() {
  const { t } = useTranslation()
  const {
    currentCluster,
    clusters,
    selectedCluster,
    isClusterHealthy,
    isLoading: clustersLoading,
    refetchClusters,
  } = useCluster()
  const { user } = useAuth()
  const [refreshInterval, setRefreshInterval] = useState("30000")
  const [showAddCluster, setShowAddCluster] = useState(false)
  const interval = refreshInterval === "0" ? false : Number(refreshInterval)
  const { data, isLoading, isError, error, refetch, isFetching } = useOverview(
    currentCluster,
    isClusterHealthy,
    interval,
  )
  const noClusters = !clustersLoading && clusters.length === 0

  return (
    <div className="flex h-full flex-col gap-2">
      <div className="flex shrink-0 items-center justify-between">
        <h1 className="text-lg font-bold tracking-tight">{t("overview.title")}</h1>
        {!noClusters && !clustersLoading && isClusterHealthy && (
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" className="size-8" onClick={() => refetch()} disabled={isFetching}>
              <RefreshCw className={`size-4 ${isFetching ? "animate-spin" : ""}`} />
            </Button>
            <Select value={refreshInterval} onValueChange={setRefreshInterval}>
              <SelectTrigger className="h-8 w-[80px]"><SelectValue /></SelectTrigger>
              <SelectContent>
                {REFRESH_OPTIONS.map((option) => <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>)}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      {clustersLoading ? (
        <OverviewSkeleton />
      ) : noClusters ? (
        <NoClusters open={showAddCluster} onOpenChange={setShowAddCluster} />
      ) : selectedCluster?.status === "unhealthy" ? (
        <ClusterUnavailable cluster={selectedCluster} onCheckAgain={refetchClusters} canRemove={canAccessAdmin(user?.role ?? "")} />
      ) : (
        <OverviewContent data={data} isLoading={isLoading} error={isError ? error : null} />
      )}
    </div>
  )
}

function NoClusters({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useTranslation()
  return (
    <>
      <Card>
        <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
          <div className="rounded-full bg-muted p-4"><Server className="size-8 text-muted-foreground" /></div>
          <div><h2 className="text-lg font-semibold">{t("overview.no_cluster")}</h2><p className="mt-1 text-sm text-muted-foreground">{t("overview.no_cluster_desc")}</p></div>
          <Button onClick={() => onOpenChange(true)} className="mt-2"><Plus className="mr-2 size-4" />{t("overview.add_cluster")}</Button>
        </CardContent>
      </Card>
      <AddClusterDialog open={open} onOpenChange={onOpenChange} />
    </>
  )
}

function OverviewContent({ data, isLoading, error }: { data?: OverviewData; isLoading: boolean; error: unknown }) {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden pb-4">
      {error !== null && (
        <div className="flex shrink-0 items-center gap-3 rounded-xl border border-destructive/20 bg-destructive/10 p-5 text-sm font-medium text-destructive">
          <AlertCircle className="size-5 shrink-0" />
          <span>{error instanceof Error ? error.message : t("overview.error")}</span>
        </div>
      )}
      <OverviewStats data={data} isLoading={isLoading} />
      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-12 lg:grid-rows-2">
        <div className="h-full min-h-0 lg:col-span-4"><ResourceUtilization resources={data?.resources ?? EMPTY_RESOURCES} isLoading={isLoading} /></div>
        <div className="h-full min-h-0 lg:col-span-4"><StorageOverview data={data} isLoading={isLoading} /></div>
        <div className="h-full min-h-0 lg:col-span-4 lg:row-span-2"><RecentEvents events={data?.recentEvents ?? []} isLoading={isLoading} /></div>
        <div className="h-full min-h-0 lg:col-span-4"><PodStatusBar dist={data?.podStatusDistribution ?? EMPTY_POD_STATUS} total={data?.pods ?? 0} isLoading={isLoading} /></div>
        <div className="h-full min-h-0 lg:col-span-4"><WorkloadSummary data={data} isLoading={isLoading} /></div>
      </div>
    </div>
  )
}

function OverviewStats({ data, isLoading }: { data?: OverviewData; isLoading: boolean }) {
  const { t } = useTranslation()
  const stats = [
    {
      title: t("overview.nodes"), icon: Monitor, value: data?.nodes,
      subtitle: readySubtitle(data?.readyNodes, data?.nodes, t("overview.all_ready"), (ready, total) => t("overview.ready_count", { ready, total })),
      color: data ? getSubtitleColor(data.readyNodes, data.nodes) : undefined,
      iconClass: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
    },
    {
      title: t("overview.pods"), icon: Box, value: data?.pods,
      subtitle: readySubtitle(data?.runningPods, data?.pods, t("overview.all_running"), (running, total) => t("overview.running_count", { running, total })),
      color: data ? getSubtitleColor(data.runningPods, data.pods) : undefined,
      iconClass: "bg-green-500/10 text-green-600 dark:text-green-400",
    },
    {
      title: t("overview.deployments"), icon: Server, value: data?.deployments,
      subtitle: readySubtitle(data?.readyDeployments, data?.deployments, t("overview.all_available"), (available, total) => t("overview.available_count", { available, total })),
      color: data ? getSubtitleColor(data.readyDeployments, data.deployments) : undefined,
      iconClass: "bg-purple-500/10 text-purple-600 dark:text-purple-400",
    },
    {
      title: t("overview.namespaces"), icon: Layers, value: data?.namespaces,
      subtitle: readySubtitle(data?.activeNamespaces, data?.namespaces, t("overview.all_active"), (active, total) => t("overview.active_count", { active, total })),
      color: data ? getSubtitleColor(data.activeNamespaces, data.namespaces) : undefined,
      iconClass: "bg-teal-500/10 text-teal-600 dark:text-teal-400",
    },
  ]
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => <StatCard key={stat.title} title={stat.title} icon={stat.icon} value={stat.value} subtitle={stat.subtitle} subtitleColor={stat.color} isLoading={isLoading} iconClass={stat.iconClass} />)}
    </div>
  )
}

function readySubtitle(active: number | undefined, total: number | undefined, allReady: string, partial: (active: number, total: number) => string) {
  if (active === undefined || total === undefined || total === 0) return undefined
  return active >= total ? allReady : partial(active, total)
}
