export type HelmValuePrimitive = string | number | boolean | null

export interface HelmValueField {
  path: string[]
  label: string
  value: HelmValuePrimitive
}

export function collectHelmValueFields(
  values: Record<string, unknown>,
  query = "",
  limit = 80,
): { fields: HelmValueField[]; truncated: boolean } {
  const normalizedQuery = query.trim().toLowerCase()
  const fields: HelmValueField[] = []
  const pending = Object.entries(values).reverse().map(([key, value]) => ({ value, path: [key] }))
  let matchingCount = 0
  let scannedCount = 0

  while (pending.length > 0 && scannedCount < 50_000) {
    const current = pending.pop()
    if (!current) break
    scannedCount += 1
    const { value, path } = current
    if (value === null || typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      const label = path.join(".")
      if (!normalizedQuery || label.toLowerCase().includes(normalizedQuery)) {
        matchingCount += 1
        if (fields.length < limit) fields.push({ path, label, value })
      }
      continue
    }
    if (!Array.isArray(value) && value && typeof value === "object") {
      const entries = Object.entries(value as Record<string, unknown>)
      for (let index = entries.length - 1; index >= 0; index -= 1) {
        const [key, child] = entries[index]
        pending.push({ value: child, path: [...path, key] })
      }
    }
  }

  return { fields, truncated: matchingCount > limit || pending.length > 0 }
}

export function setHelmValue(
  values: Record<string, unknown>,
  path: string[],
  nextValue: HelmValuePrimitive,
): Record<string, unknown> {
  if (path.length === 0) return values
  const update = (current: Record<string, unknown>, index: number): Record<string, unknown> => {
    const key = path[index]
    if (index === path.length - 1) return { ...current, [key]: nextValue }
    const child = current[key]
    const childObject = child && typeof child === "object" && !Array.isArray(child)
      ? child as Record<string, unknown>
      : {}
    return { ...current, [key]: update(childObject, index + 1) }
  }
  return update(values, 0)
}

export function isSensitiveHelmValue(path: string[]): boolean {
  return /(password|passwd|token|secret|private.?key|encryption.?key|auth.?key|api.?key)/i.test(path.join("."))
}

export function generateHelmSecret(byteLength = 32): string {
  const bytes = new Uint8Array(byteLength)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")
}
