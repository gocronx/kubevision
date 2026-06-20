import { useState, type KeyboardEvent } from "react"
import { useTranslation } from "react-i18next"
import { Send, Square } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"

interface Props {
  isLoading: boolean
  disabled?: boolean
  onSend: (text: string) => void
  onStop: () => void
}

export function ChatComposer({ isLoading, disabled, onSend, onStop }: Props) {
  const { t } = useTranslation()
  const [value, setValue] = useState("")

  const send = () => {
    const text = value.trim()
    if (!text) return
    onSend(text)
    setValue("")
  }

  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      send()
    }
  }

  return (
    <div className="flex items-end gap-2 border-t p-3">
      <Textarea
        value={value}
        disabled={disabled}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={t("ai.placeholder")}
        rows={1}
        className="max-h-32 min-h-9 resize-none"
      />
      {isLoading ? (
        <Button size="icon" variant="outline" onClick={onStop} aria-label={t("ai.stop")}>
          <Square className="size-4" />
        </Button>
      ) : (
        <Button size="icon" onClick={send} disabled={disabled || !value.trim()} aria-label={t("ai.send")}>
          <Send className="size-4" />
        </Button>
      )}
    </div>
  )
}
