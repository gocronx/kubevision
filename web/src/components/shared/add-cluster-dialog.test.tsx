import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { vi } from "vitest"
import { readFileAsText } from "@/lib/read-file"
import { AddClusterDialog } from "./add-cluster-dialog"

const { apiPost, toastError } = vi.hoisted(() => ({
  apiPost: vi.fn(),
  toastError: vi.fn(),
}))

vi.mock("@/lib/api", () => ({ default: { post: apiPost } }))
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))
vi.mock("sonner", () => ({
  toast: { error: toastError, success: vi.fn() },
}))

function renderDialog(open = true) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const onOpenChange = vi.fn()
  const view = render(
    <QueryClientProvider client={queryClient}>
      <AddClusterDialog open={open} onOpenChange={onOpenChange} />
    </QueryClientProvider>,
  )
  return { ...view, onOpenChange, queryClient }
}

describe("AddClusterDialog", () => {
  it("defaults to kubeconfig and requires its contents", () => {
    renderDialog()

    expect(screen.getByRole("radio", { name: "Kubeconfig" })).toBeChecked()
    expect(screen.getByLabelText(/cluster\.kubeconfig/)).toBeVisible()
    expect(screen.getByRole("button", { name: "cluster.add_submit" })).toBeDisabled()
  })

  it("shows the in-cluster warning only when explicitly selected", () => {
    renderDialog()

    fireEvent.click(screen.getByRole("radio", { name: "In-Cluster" }))

    expect(screen.getByRole("radio", { name: "In-Cluster" })).toBeChecked()
    expect(screen.queryByLabelText(/cluster\.kubeconfig/)).not.toBeInTheDocument()
    expect(screen.getByText("cluster.in_cluster_hint")).toBeVisible()
  })

  it("resets authentication to kubeconfig whenever it reopens", () => {
    const { rerender, queryClient, onOpenChange } = renderDialog()
    fireEvent.click(screen.getByRole("radio", { name: "In-Cluster" }))

    rerender(
      <QueryClientProvider client={queryClient}>
        <AddClusterDialog open={false} onOpenChange={onOpenChange} />
      </QueryClientProvider>,
    )
    rerender(
      <QueryClientProvider client={queryClient}>
        <AddClusterDialog open onOpenChange={onOpenChange} />
      </QueryClientProvider>,
    )

    expect(screen.getByRole("radio", { name: "Kubeconfig" })).toBeChecked()
    expect(screen.getByLabelText(/cluster\.kubeconfig/)).toBeVisible()
  })

  it("does not submit when Enter is pressed before kubeconfig is provided", () => {
    renderDialog()
    fireEvent.change(screen.getByLabelText(/cluster\.name/), { target: { value: "k3d" } })

    fireEvent.keyDown(screen.getByLabelText(/cluster\.name/), { key: "Enter" })

    expect(apiPost).not.toHaveBeenCalled()
  })

  it("submits a kubeconfig payload after both required fields are provided", async () => {
    apiPost.mockResolvedValueOnce({})
    renderDialog()
    fireEvent.change(screen.getByLabelText(/cluster\.name/), { target: { value: " k3d-kubevision " } })
    fireEvent.change(screen.getByLabelText(/cluster\.kubeconfig/), {
      target: { value: "apiVersion: v1\nkind: Config" },
    })

    fireEvent.click(screen.getByRole("button", { name: "cluster.add_submit" }))

    await waitFor(() => expect(apiPost).toHaveBeenCalledWith("/clusters", {
      name: "k3d-kubevision",
      authType: "kubeconfig",
      kubeconfig: "apiVersion: v1\nkind: Config",
    }))
  })
})

describe("readFileAsText", () => {
  it("loads an extensionless kubeconfig without using File.text", async () => {
    const kubeconfig = [
      "apiVersion: v1",
      "kind: Config",
      "clusters: []",
      "contexts: []",
      "users: []",
    ].join("\n")
    const file = new File([kubeconfig], "config", { type: "application/octet-stream" })
    Object.defineProperty(file, "text", { value: undefined })
    class TestFileReader {
      result: string | ArrayBuffer | null = null
      error: DOMException | null = null
      onload: ((event: ProgressEvent<FileReader>) => void) | null = null
      onerror: ((event: ProgressEvent<FileReader>) => void) | null = null
      onabort: ((event: ProgressEvent<FileReader>) => void) | null = null

      readAsText() {
        this.result = kubeconfig
        queueMicrotask(() => {
          const event = new ProgressEvent("load") as ProgressEvent<FileReader>
          this.onload?.(event)
        })
      }
    }
    vi.stubGlobal("FileReader", TestFileReader)

    await expect(readFileAsText(file)).resolves.toBe(kubeconfig)
  })
})
