import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook } from "@testing-library/react"
import type { PropsWithChildren } from "react"
import { expect, it, vi } from "vitest"
import api from "@/lib/api"
import { useUpdateAIConfig } from "./use-ai-config"

vi.mock("@/lib/api", () => ({ default: { put: vi.fn() } }))

it("publishes saved AI availability to the shared query cache immediately", async () => {
  const client = new QueryClient()
  vi.mocked(api.put).mockResolvedValue({
    enabled: false,
    baseURL: "https://models.example/v1",
    model: "model-a",
    maxTokens: 4096,
    hasApiKey: true,
  })
  const wrapper = ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  const { result } = renderHook(() => useUpdateAIConfig(), { wrapper })

  await act(async () => {
    await result.current.mutateAsync({
      enabled: false,
      baseURL: "https://models.example/v1",
      model: "model-a",
      apiKey: "",
      maxTokens: 4096,
    })
  })

  expect(client.getQueryData(["ai", "config"])).toMatchObject({ enabled: false, hasApiKey: true })
})
