import { describe, expect, it, vi } from "vitest"
import { collectHelmValueFields, generateHelmSecret, isSensitiveHelmValue, setHelmValue } from "./package-values"

describe("Helm values helpers", () => {
  it("collects nested scalar fields and skips complex arrays", () => {
    const result = collectHelmValueFields({ replicaCount: 2, db: { host: "postgres", enabled: true }, ports: [80] })
    expect(result.fields.map((field) => field.label)).toEqual(["replicaCount", "db.host", "db.enabled"])
  })

  it("updates a nested value immutably", () => {
    const values = { db: { host: "old", port: 5432 } }
    const updated = setHelmValue(values, ["db", "host"], "new")
    expect(updated).toEqual({ db: { host: "new", port: 5432 } })
    expect(values.db.host).toBe("old")
  })

  it("searches fields beyond the initial display limit", () => {
    const values = Object.fromEntries(Array.from({ length: 100 }, (_, index) => [`field${index}`, index]))
    const result = collectHelmValueFields(values, "field99")
    expect(result.fields.map((field) => field.label)).toEqual(["field99"])
    expect(result.truncated).toBe(false)
  })

  it("recognizes sensitive paths and generates cryptographic values", () => {
    vi.spyOn(crypto, "getRandomValues").mockImplementation((array) => {
      const bytes = array as Uint8Array
      bytes.fill(15)
      return array
    })
    expect(isSensitiveHelmValue(["managed", "authSecret"])).toBe(true)
    expect(isSensitiveHelmValue(["service", "port"])).toBe(false)
    expect(generateHelmSecret(4)).toBe("0f0f0f0f")
  })
})
