import { describe, expect, it } from "vitest"
import { prepareSecretForEditing } from "./secret-edit"

describe("prepareSecretForEditing", () => {
  it("moves UTF-8 data values to editable stringData", () => {
    const input = {
      apiVersion: "v1",
      kind: "Secret",
      metadata: { name: "redis-secret" },
      data: { REDIS_PASSWORD: "cm9vdDE=", CONFIG: "bGluZTEKbGluZTI=" },
    }

    expect(prepareSecretForEditing(input)).toEqual({
      apiVersion: "v1",
      kind: "Secret",
      metadata: { name: "redis-secret" },
      stringData: { REDIS_PASSWORD: "root1", CONFIG: "line1\nline2" },
    })
    expect(input.data.REDIS_PASSWORD).toBe("cm9vdDE=")
  })

  it("keeps binary values in data and preserves existing stringData", () => {
    const result = prepareSecretForEditing({
      data: { binary: "/wAB", password: "b2xk" },
      stringData: { password: "new" },
    })

    expect(result).toEqual({
      data: { binary: "/wAB", password: "b2xk" },
      stringData: { password: "new" },
    })
  })

  it("leaves malformed base64 in data so validation remains visible", () => {
    expect(prepareSecretForEditing({ data: { password: "root1" } })).toEqual({
      data: { password: "root1" },
    })
  })
})
