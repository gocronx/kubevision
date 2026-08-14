import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render, screen } from "@testing-library/react"
import { MemoryRouter, Route, Routes } from "react-router-dom"
import { vi } from "vitest"
import { PackageReleaseDetailPage } from "./detail"
import { packageResourcePath } from "./package-resource-path"

vi.mock("@/hooks/use-cluster", () => ({
  useCluster: () => ({ currentCluster: "1" }),
}))

vi.mock("@/hooks/use-package-releases", () => ({
  usePackageRelease: () => ({ data: { name: "nginx", namespace: "default", revision: 1, status: "deployed", resources: [{ apiVersion: "apps/v1", kind: "Deployment", namespace: "default", name: "nginx" }] }, isLoading: false, isError: false, refetch: vi.fn() }),
  usePackageHistory: () => ({ data: [], refetch: vi.fn() }),
  usePackageRollback: () => ({ mutate: vi.fn(), isPending: false }),
  usePackageRemove: () => ({ mutate: vi.fn(), isPending: false }),
  useCheckPackageUpgrade: () => ({ mutate: vi.fn(), isPending: false }),
  usePackageChange: () => ({ preview: { mutate: vi.fn(), reset: vi.fn(), isPending: false }, execute: { mutate: vi.fn(), reset: vi.fn(), isPending: false } }),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}))

it("links back to the package release list", () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/package-releases/default/nginx"]}>
        <Routes>
          <Route path="/package-releases/:namespace/:name" element={<PackageReleaseDetailPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )

  expect(screen.getByRole("link", { name: "packages.backToList" })).toHaveAttribute("href", "/package-releases")
  expect(screen.getByRole("button", { name: "packages.checkUpdates" })).toBeInTheDocument()
  expect(screen.getByText("packages.helmDeployed")).toBeInTheDocument()
  expect(packageResourcePath({ apiVersion: "apps/v1", kind: "Deployment", namespace: "default", name: "nginx" })).toBe("/deployments/nginx?namespace=default")
})
