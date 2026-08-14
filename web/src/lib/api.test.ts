import { describe, expect, it } from "vitest"
import { TokenRefreshQueue } from "./api"

describe("TokenRefreshQueue", () => {
  it("resolves every queued request and releases its waiters", async () => {
    const queue = new TokenRefreshQueue()
    const first = queue.wait()
    const second = queue.wait()

    expect(queue.size).toBe(2)
    queue.resolve("new-token")

    await expect(first).resolves.toBe("new-token")
    await expect(second).resolves.toBe("new-token")
    expect(queue.size).toBe(0)
  })

  it("rejects every queued request and releases its waiters", async () => {
    const queue = new TokenRefreshQueue()
    const first = queue.wait()
    const second = queue.wait()
    const error = new Error("refresh failed")

    queue.reject(error)

    await expect(first).rejects.toBe(error)
    await expect(second).rejects.toBe(error)
    expect(queue.size).toBe(0)
  })
})
