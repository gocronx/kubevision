import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { vi } from "vitest"
import api from "@/lib/api"
import { DirectorySettingsPage } from "./directory"

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

describe("DirectorySettingsPage", () => {
  it("normalizes a null mappings response to an empty list", async () => {
    vi.mocked(api.get).mockResolvedValue({ mappings: null })

    render(<DirectorySettingsPage />)

    await waitFor(() => expect(api.get).toHaveBeenCalledWith("/directory/config"))
    fireEvent.click(screen.getByRole("button", { name: "directory.addMapping" }))

    expect(screen.getByPlaceholderText("directory.groupIdentifier")).toBeInTheDocument()
  })
})
