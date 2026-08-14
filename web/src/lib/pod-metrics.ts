export function formatCPU(milli: number | undefined): string {
  if (milli === undefined || !Number.isFinite(milli)) return "-"
  if (milli < 1000) return `${Math.round(milli)}m`
  return `${trimNumber(milli / 1000)} cores`
}

export function formatBytes(bytes: number | undefined): string {
  if (bytes === undefined || !Number.isFinite(bytes)) return "-"
  const units = ["B", "KiB", "MiB", "GiB", "TiB"]
  let value = Math.max(0, bytes)
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${trimNumber(value)} ${units[unit]}`
}

export function usagePercent(used: number, limit: number | undefined): number | undefined {
  if (!limit || limit <= 0) return undefined
  return Math.max(0, (used / limit) * 100)
}

function trimNumber(value: number): string {
  return Number(value.toFixed(value >= 10 ? 1 : 2)).toString()
}
