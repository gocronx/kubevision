import { beforeEach, describe, expect, it, vi } from "vitest"
import api from "./api"
import { publicKeyEnabled } from "./public-key-auth"

vi.mock("./api", () => ({ default: { get: vi.fn() } }))

describe("publicKeyEnabled", () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset()
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: true })
    Object.defineProperty(window, "PublicKeyCredential", { configurable: true, value: class {} })
  })

  it("requires browser support", async () => {
    Object.defineProperty(window, "isSecureContext", { configurable: true, value: false })

    await expect(publicKeyEnabled()).resolves.toBe(false)
    expect(api.get).not.toHaveBeenCalled()
  })

  it("returns the backend capability", async () => {
    vi.mocked(api.get).mockResolvedValue({ enabled: true })

    await expect(publicKeyEnabled()).resolves.toBe(true)
    expect(api.get).toHaveBeenCalledWith("/auth/public-key/config")
  })

  it("defaults to disabled when the capability request fails", async () => {
    vi.mocked(api.get).mockRejectedValue(new Error("unavailable"))

    await expect(publicKeyEnabled()).resolves.toBe(false)
  })
})
