import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import api from "@/lib/api"
import { AISettingsPage } from "./ai"

vi.mock("@/lib/api", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
  },
}))

vi.mock("@/stores/auth-store", () => ({
  useAuth: () => ({ user: { role: "super-admin" } }),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <AISettingsPage />
    </QueryClientProvider>,
  )
}

describe("AISettingsPage model picker", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockResolvedValue({
      enabled: true,
      baseURL: "https://models.example/v1",
      model: "old-model",
      maxTokens: 4096,
      hasApiKey: true,
    })
    vi.mocked(api.post).mockResolvedValue([{ id: "model-a" }, { id: "model-b" }])
    vi.mocked(api.put).mockResolvedValue({
      enabled: true,
      baseURL: "https://models.example/v1",
      model: "model-a",
      maxTokens: 4096,
      hasApiKey: true,
    })
  })

  it("collapses after selecting a model and after saving", async () => {
    renderPage()

    const discover = await screen.findByRole("button", { name: "ai.discoverModels" })
    fireEvent.click(discover)
    expect(await screen.findByRole("listbox", { name: "ai.availableModels" })).toBeInTheDocument()

    fireEvent.click(screen.getByRole("option", { name: "model-a" }))
    expect(screen.queryByRole("listbox", { name: "ai.availableModels" })).not.toBeInTheDocument()
    expect(screen.getByLabelText("ai.model")).toHaveValue("model-a")

    fireEvent.click(discover)
    expect(await screen.findByRole("listbox", { name: "ai.availableModels" })).toBeInTheDocument()
    fireEvent.click(screen.getByRole("button", { name: "common.save" }))

    await waitFor(() => expect(api.put).toHaveBeenCalled())
    await waitFor(() => {
      expect(screen.queryByRole("listbox", { name: "ai.availableModels" })).not.toBeInTheDocument()
    })
  })

  it("requires a new API key when the provider URL changes", async () => {
    renderPage()

    const baseURL = await screen.findByLabelText("ai.baseURL")
    fireEvent.change(baseURL, { target: { value: "https://other.example/v1" } })

    expect(screen.getByText("ai.providerKeyRequired")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "ai.discoverModels" })).toBeDisabled()
    expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled()

    fireEvent.change(screen.getByLabelText("ai.apiKey"), { target: { value: "new-key" } })
    expect(screen.getByRole("button", { name: "ai.discoverModels" })).toBeEnabled()
    expect(screen.getByRole("button", { name: "common.save" })).toBeEnabled()
  })
})
