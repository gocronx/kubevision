const STORAGE_KEY_PREFIX = "kubevision:favorites-panel-open"

function storageKey(userID: number): string {
  return `${STORAGE_KEY_PREFIX}:${userID}`
}

export function readFavoritesPanelOpen(userID: number): boolean {
  try {
    return localStorage.getItem(storageKey(userID)) !== "false"
  } catch {
    return true
  }
}

export function writeFavoritesPanelOpen(userID: number, isOpen: boolean): void {
  try {
    localStorage.setItem(storageKey(userID), String(isOpen))
  } catch {
    // A blocked or full storage should not prevent the sidebar from working.
  }
}
