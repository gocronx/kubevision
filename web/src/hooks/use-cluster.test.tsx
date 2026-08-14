import { act, renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query"
import type { ReactNode } from "react"
import { vi } from "vitest"
import { ClusterProvider, useCluster, type Cluster } from "./use-cluster"

const { apiGet } = vi.hoisted(() => ({ apiGet: vi.fn() }))
vi.mock("@/lib/api", () => ({ default: { get: apiGet } }))

const clusters: Cluster[] = [
  { id: 1, name: "one", status: "healthy" },
  { id: 2, name: "two", status: "healthy" },
]

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ClusterProvider>{children}</ClusterProvider>
      </QueryClientProvider>
    )
  }
}

describe("useCluster", () => {
  it("shares cluster changes between independent consumers", async () => {
    localStorage.setItem("kubevision-current-cluster", "1")
    const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } })
    queryClient.setQueryData(["clusters"], clusters)
    const { result } = renderHook(() => ({
      first: useCluster(),
      second: useCluster(),
    }), { wrapper: createWrapper(queryClient) })

    act(() => result.current.first.setCurrentCluster("2"))

    await waitFor(() => expect(result.current.second.currentCluster).toBe("2"))
    expect(result.current.second.selectedCluster?.name).toBe("two")
  })

  it("refetches fresh cached data when switching back to a cluster", async () => {
    localStorage.setItem("kubevision-current-cluster", "1")
    const queryClient = new QueryClient({ defaultOptions: { queries: { staleTime: Infinity } } })
    queryClient.setQueryData(["clusters"], clusters)
    queryClient.setQueryData(["overview", "2"], "cached")
    apiGet.mockResolvedValue("fresh")

    const { result } = renderHook(() => {
      const cluster = useCluster()
      const overview = useQuery({
        queryKey: ["overview", cluster.currentCluster],
        queryFn: () => apiGet(`/clusters/${cluster.currentCluster}/overview`),
      })
      return { cluster, overview }
    }, { wrapper: createWrapper(queryClient) })

    act(() => result.current.cluster.setCurrentCluster("2"))

    await waitFor(() => expect(apiGet).toHaveBeenCalledWith("/clusters/2/overview"))
    await waitFor(() => expect(result.current.overview.data).toBe("fresh"))
  })
})
