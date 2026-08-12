import { fireEvent, render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { vi } from "vitest"
import api from "@/lib/api"
import { ImageTagEditor } from "./image-tag-editor"
import { setContainerImage } from "@/lib/container-images"

vi.mock("@/lib/api", () => ({ default: { get: vi.fn() } }))

const workload = JSON.stringify({
  spec: { template: { spec: { containers: [{ name: "app", image: "docker.io/acme/app:v1" }] } } },
}, null, 2)

function renderEditor(onChange = vi.fn()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ImageTagEditor json={workload} onChange={onChange} />
    </QueryClientProvider>
  )
}

test("updates only the selected container image", () => {
  const updated = JSON.parse(setContainerImage(workload, ["spec", "template", "spec", "containers", "0", "image"], "acme/app:v2"))
  expect(updated.spec.template.spec.containers[0].image).toBe("acme/app:v2")
})

test("manual editing remains available when discovery fails", async () => {
  vi.mocked(api.get).mockRejectedValue(new Error("unavailable"))
  const onChange = vi.fn()
  renderEditor(onChange)
  const input = screen.getByRole("combobox", { name: "app" })
  fireEvent.change(input, { target: { value: "private.example/app:manual" } })
  expect(onChange).toHaveBeenCalledWith(expect.stringContaining("private.example/app:manual"))
  expect(input).not.toBeDisabled()
})
