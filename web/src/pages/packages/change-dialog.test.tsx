import { fireEvent, render, screen } from "@testing-library/react"
import { beforeEach, vi } from "vitest"
import { PackageChangeDialog } from "./change-dialog"

const previewMutate = vi.fn()
const executeMutate = vi.fn()
const { toastError } = vi.hoisted(() => ({ toastError: vi.fn() }))
vi.mock("@/hooks/use-package-releases", () => ({
  usePackageChange: () => ({ preview: { mutate: previewMutate, reset: vi.fn(), isPending: false }, execute: { mutate: executeMutate, reset: vi.fn(), isPending: false } }),
}))
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: (key: string) => key }) }))
vi.mock("sonner", () => ({ toast: { error: toastError, success: vi.fn() } }))

beforeEach(() => {
  vi.clearAllMocks()
})

it("does not submit an install when Enter is pressed in a field", () => {
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" />)
  fireEvent.change(screen.getByLabelText("packages.releaseName"), { target: { value: "demo" } })
  fireEvent.keyDown(screen.getByLabelText("packages.releaseName"), { key: "Enter" })
  expect(previewMutate).not.toHaveBeenCalled()
  expect(executeMutate).not.toHaveBeenCalled()
})

it("only offers preview before a confirmation token exists", () => {
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="demo" chart="oci://registry.example/demo" />)
  expect(screen.getByRole("button", { name: "packages.preview" })).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "packages.confirmInstall" })).not.toBeInTheDocument()
})

it("rejects a chart URL when a repository URL is provided", () => {
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="gocron" chart="https://charts.example/gocron" />)
  fireEvent.change(screen.getByLabelText("packages.repoUrl"), { target: { value: "https://charts.example" } })
  fireEvent.click(screen.getByRole("button", { name: "packages.preview" }))
  expect(previewMutate).not.toHaveBeenCalled()
  expect(toastError).toHaveBeenCalledWith("packages.repositoryChartNameRequired")
})

it("requires a new preview after execution fails", () => {
  previewMutate.mockImplementationOnce((_input, options) => options.onSuccess({
    operation: "install",
    chart: "demo",
    chartVersion: "1.0.0",
    digest: "digest",
    manifest: "",
    resources: [],
    risks: [],
    canExecute: true,
    confirmationToken: "one-time-token",
  }))
  executeMutate.mockImplementationOnce((_input, options) => options.onError(new Error("expired")))
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="demo" chart="oci://registry.example/demo" />)

  fireEvent.click(screen.getByRole("button", { name: "packages.preview" }))
  fireEvent.click(screen.getByRole("button", { name: "packages.confirmInstall" }))

  expect(screen.getByRole("button", { name: "packages.previewAgain" })).toBeInTheDocument()
})

it("automatically previews a trusted upgrade candidate", () => {
  render(<PackageChangeDialog open autoPreview onOpenChange={vi.fn()} cluster="1" operation="upgrade" releaseName="demo" source={{ chart: "demo", repoUrl: "https://charts.example", version: "1.1.0" }} initialValues={{}} />)

  expect(previewMutate).toHaveBeenCalledWith(expect.objectContaining({
    releaseName: "demo",
    source: { chart: "demo", repoUrl: "https://charts.example", version: "1.1.0" },
    values: {},
  }), expect.any(Object))
})

it("edits nested defaults in guided mode", () => {
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="demo" chart="demo" initialValues={{ db: { host: "old" } }} />)

  fireEvent.change(screen.getByLabelText("db.host"), { target: { value: "postgres.default.svc" } })
  fireEvent.click(screen.getByRole("button", { name: "packages.preview" }))

  expect(previewMutate).toHaveBeenCalledWith(expect.objectContaining({
    values: { db: { host: "postgres.default.svc" } },
  }), expect.any(Object))
})

it("generates a secure value for sensitive defaults", () => {
  vi.spyOn(crypto, "getRandomValues").mockImplementation((array) => {
    const bytes = array as Uint8Array
    bytes.fill(10)
    return array
  })
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="demo" chart="demo" initialValues={{ authSecret: "" }} />)

  fireEvent.click(screen.getByTitle("packages.generateSecret"))
  fireEvent.click(screen.getByRole("button", { name: "packages.preview" }))

  expect(previewMutate).toHaveBeenCalledWith(expect.objectContaining({
    values: { authSecret: "0a".repeat(32) },
  }), expect.any(Object))
})

it("validates the operation timeout before preview", () => {
  render(<PackageChangeDialog open onOpenChange={vi.fn()} cluster="1" operation="install" releaseName="demo" chart="demo" />)

  fireEvent.change(screen.getByLabelText("packages.timeoutSeconds"), { target: { value: "0" } })
  fireEvent.click(screen.getByRole("button", { name: "packages.preview" }))

  expect(previewMutate).not.toHaveBeenCalled()
  expect(toastError).toHaveBeenCalledWith("packages.invalidTimeout")
})
