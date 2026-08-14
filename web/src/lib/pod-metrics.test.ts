import { describe, expect, it } from "vitest"
import { formatBytes, formatCPU, usagePercent } from "./pod-metrics"

describe("pod metrics formatting", () => {
  it("formats CPU and binary memory units", () => {
    expect(formatCPU(2)).toBe("2m")
    expect(formatCPU(1250)).toBe("1.25 cores")
    expect(formatBytes(10 * 1024 * 1024)).toBe("10 MiB")
  })

  it("only computes utilization when a limit exists", () => {
    expect(usagePercent(50, 200)).toBe(25)
    expect(usagePercent(50, 0)).toBeUndefined()
  })
})
