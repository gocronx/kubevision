export function getSubtitleColor(active: number, total: number): string {
  if (total === 0) return "text-muted-foreground"
  const ratio = active / total
  if (ratio >= 1) return "text-green-500"
  if (ratio >= 0.5) return "text-amber-500"
  return "text-red-500"
}

export function formatMemory(bytes: number): string {
  if (bytes === 0) return "0"
  const units = ["B", "Ki", "Mi", "Gi", "Ti"]
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  return `${value % 1 === 0 ? value : value.toFixed(1)} ${units[unitIndex]}`
}
