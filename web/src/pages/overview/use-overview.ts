import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"
import type { OverviewData } from "./types"

export function useOverview(
  clusterID: string,
  enabled: boolean,
  refetchInterval: number | false,
) {
  return useQuery<OverviewData>({
    queryKey: ["overview", clusterID],
    queryFn: () => api.get<OverviewData>(`/clusters/${clusterID}/overview`),
    enabled: enabled && !!clusterID,
    refetchInterval: enabled ? refetchInterval : false,
  })
}
