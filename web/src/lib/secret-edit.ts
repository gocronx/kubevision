type K8sObject = Record<string, unknown>

/**
 * Prepares a Kubernetes Secret for human-friendly editing. UTF-8 text values
 * move from base64-encoded data to stringData; binary values remain untouched.
 * Kubernetes encodes stringData into data when the update is applied.
 */
export function prepareSecretForEditing(resource: K8sObject): K8sObject {
  const data = asStringRecord(resource.data)
  const existingStringData = asStringRecord(resource.stringData)
  if (!data) return { ...resource }

  const remainingData = { ...data }
  const stringData = { ...(existingStringData ?? {}) }

  for (const [key, encoded] of Object.entries(data)) {
    if (key in stringData) continue
    const decoded = decodeTextSecret(encoded)
    if (decoded === null) continue
    stringData[key] = decoded
    delete remainingData[key]
  }

  const result = { ...resource }
  if (Object.keys(remainingData).length > 0) result.data = remainingData
  else delete result.data
  if (Object.keys(stringData).length > 0) result.stringData = stringData
  return result
}

function asStringRecord(value: unknown): Record<string, string> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null
  const entries = Object.entries(value)
  if (!entries.every(([, item]) => typeof item === "string")) return null
  return Object.fromEntries(entries) as Record<string, string>
}

function decodeTextSecret(encoded: string): string | null {
  try {
    const binary = atob(encoded)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    const text = new TextDecoder("utf-8", { fatal: true }).decode(bytes)
    return isHumanReadable(text) ? text : null
  } catch {
    return null
  }
}

function isHumanReadable(value: string): boolean {
  for (const char of value) {
    const code = char.codePointAt(0) ?? 0
    if (code < 0x20 && char !== "\n" && char !== "\r" && char !== "\t") return false
    if (code === 0x7f) return false
  }
  return true
}
