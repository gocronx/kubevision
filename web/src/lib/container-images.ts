export function setContainerImage(json: string, path: string[], image: string): string {
  const document = JSON.parse(json) as Record<string, unknown>
  let current: unknown = document
  for (const segment of path.slice(0, -1)) {
    if (Array.isArray(current)) current = current[Number(segment)]
    else current = (current as Record<string, unknown>)[segment]
  }
  ;(current as Record<string, unknown>)[path[path.length - 1]] = image
  return JSON.stringify(document, null, 2)
}
