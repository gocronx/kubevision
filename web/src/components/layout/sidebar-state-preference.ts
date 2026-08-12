const SIDEBAR_COOKIE_NAME = "sidebar_state"

export function readSidebarOpen(): boolean {
  if (typeof document === "undefined") return true

  const value = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(`${SIDEBAR_COOKIE_NAME}=`))
    ?.slice(SIDEBAR_COOKIE_NAME.length + 1)

  return value !== "false"
}
