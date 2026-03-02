import { useTheme as useNextTheme } from "next-themes"

export function useTheme() {
  const { theme, setTheme, resolvedTheme } = useNextTheme()

  return {
    theme: (theme ?? "system") as "dark" | "light" | "system",
    setTheme,
    resolvedTheme: resolvedTheme as "dark" | "light" | undefined,
  } as const
}
