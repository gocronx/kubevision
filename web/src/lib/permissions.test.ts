import { canAccessAdmin, canExec, canManageUsers, canMutateResources, isReadOnly } from "./permissions"

describe("role permission helpers", () => {
  it("reserves user administration for the highest privileged role", () => {
    expect(canManageUsers("super-admin")).toBe(true)
    expect(canManageUsers("admin")).toBe(false)
  })

  it("keeps viewer and unknown roles from mutating resources or opening terminals", () => {
    for (const role of ["viewer", "custom", "unrecognized"]) {
      expect(canMutateResources(role)).toBe(false)
      expect(canExec(role)).toBe(false)
      expect(isReadOnly(role)).toBe(true)
    }
  })

  it("allows both administrator roles into the administration area", () => {
    expect(canAccessAdmin("super-admin")).toBe(true)
    expect(canAccessAdmin("admin")).toBe(true)
    expect(canAccessAdmin("editor")).toBe(false)
  })
})
