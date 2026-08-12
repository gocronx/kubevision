import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen, waitFor } from "@testing-library/react"
import { vi } from "vitest"
import { OverviewPage } from "./index"
import api from "@/lib/api"

vi.mock("@/hooks/use-cluster", () => ({
  useCluster: () => ({
    currentCluster: "1",
    clusters: [{ id: 1, name: "local", status: "unhealthy" }],
    selectedCluster: { id: 1, name: "local", status: "unhealthy" },
    isClusterHealthy: false,
    isLoading: false,
    isFetchingClusters: false,
    refetchClusters: vi.fn(),
    setCurrentCluster: vi.fn(),
  }),
}))

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock("@/stores/auth-store", () => ({
  useAuth: () => ({ user: { role: "admin" } }),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, values?: { name?: string }) =>
      key === "cluster.unavailable_title"
        ? "Cluster unavailable"
        : values?.name ?? key,
  }),
}))

describe("OverviewPage", () => {
  it("does not request overview data for an unhealthy cluster", async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })

    render(
      <QueryClientProvider client={client}>
        <OverviewPage />
      </QueryClientProvider>
    )

    expect(screen.getByText("Cluster unavailable")).toBeInTheDocument()
    await waitFor(() => expect(api.get).not.toHaveBeenCalled())
  })
})
