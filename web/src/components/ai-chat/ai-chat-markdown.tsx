// A dependency-free renderer for assistant replies. It is intentionally small:
// it handles fenced code blocks (```), inline code (`), and preserves
// whitespace for everything else — enough for kubectl/YAML-flavored answers
// without pulling in a full Markdown engine.

interface Props {
  content: string
}

export function ChatMarkdown({ content }: Props) {
  const blocks = content.split(/```/)
  return (
    <div className="space-y-2 text-sm leading-relaxed">
      {blocks.map((block, i) =>
        i % 2 === 1 ? (
          <pre
            key={i}
            className="overflow-x-auto rounded-md bg-muted p-3 font-mono text-xs"
          >
            <code>{stripLang(block)}</code>
          </pre>
        ) : (
          <p key={i} className="whitespace-pre-wrap break-words">
            {renderInline(block)}
          </p>
        )
      )}
    </div>
  )
}

// Drops an optional leading language hint line (e.g. ```yaml).
function stripLang(block: string): string {
  const newline = block.indexOf("\n")
  if (newline === -1) return block
  const first = block.slice(0, newline).trim()
  if (/^[a-zA-Z0-9_-]+$/.test(first)) {
    return block.slice(newline + 1)
  }
  return block.replace(/^\n/, "")
}

// Renders inline code spans; everything else is plain text.
function renderInline(text: string) {
  if (!text) return null
  const parts = text.split(/`/)
  return parts.map((part, i) =>
    i % 2 === 1 ? (
      <code key={i} className="rounded bg-muted px-1 py-0.5 font-mono text-xs">
        {part}
      </code>
    ) : (
      <span key={i}>{part}</span>
    )
  )
}
