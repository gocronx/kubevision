import { useTranslation } from "react-i18next"
import { Send, Square } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"

interface Props {
  isLoading: boolean
  disabled?: boolean
  value: string
  onChange: (value: string) => void
  onSend: (text: string) => void
  onStop: () => void
}

export function ChatComposer({ isLoading, disabled, value, onChange, onSend, onStop }: Props) {
  const { t } = useTranslation()

  const send = () => {
    const text = value.trim()
    if (!text) return
    onSend(text)
  }

  return (
    <div className="flex items-end gap-2 border-t p-3">
      <Textarea
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
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
