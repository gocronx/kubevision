import type { KeyboardEvent as ReactKeyboardEvent } from "react"

// IME candidate confirmation can emit Enter before composition fully ends.
// Prevent that keystroke from bubbling into a form submit while preserving
// ordinary Enter submission for accessible, low-risk forms.
export function preventSubmitWhileComposing(event: ReactKeyboardEvent<HTMLElement>) {
  if (event.key !== "Enter") return
  if (event.nativeEvent.isComposing || event.nativeEvent.keyCode === 229) {
    event.preventDefault()
  }
}
