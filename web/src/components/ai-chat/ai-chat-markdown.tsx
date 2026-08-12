import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"

interface Props {
  content: string
}

export function ChatMarkdown({ content }: Props) {
  return (
    <div className="min-w-0 text-sm leading-6">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        skipHtml
        components={{
        h1: ({ children }) => <h2 className="mt-4 mb-2 text-base font-semibold first:mt-0">{children}</h2>,
        h2: ({ children }) => <h2 className="mt-4 mb-2 text-base font-semibold first:mt-0">{children}</h2>,
        h3: ({ children }) => <h3 className="mt-3 mb-1.5 text-sm font-semibold first:mt-0">{children}</h3>,
        h4: ({ children }) => <h4 className="mt-3 mb-1 text-sm font-medium first:mt-0">{children}</h4>,
        p: ({ children }) => <p className="my-2 break-words first:mt-0 last:mb-0">{children}</p>,
        strong: ({ children }) => <strong className="font-semibold text-foreground">{children}</strong>,
        ul: ({ children }) => <ul className="my-2 list-disc space-y-1 pl-5 marker:text-muted-foreground">{children}</ul>,
        ol: ({ children }) => <ol className="my-2 list-decimal space-y-1 pl-5 marker:text-muted-foreground">{children}</ol>,
        li: ({ children }) => <li className="break-words pl-0.5">{children}</li>,
        blockquote: ({ children }) => <blockquote className="my-2 border-l-2 border-primary/40 pl-3 text-muted-foreground">{children}</blockquote>,
        a: ({ href, children }) => <a href={href} target="_blank" rel="noreferrer noopener" className="text-primary underline underline-offset-2 hover:no-underline">{children}</a>,
        pre: ({ children }) => <pre className="my-2 max-w-full overflow-x-auto rounded-md border bg-background/70 p-3 font-mono text-xs leading-relaxed">{children}</pre>,
        code: ({ className, children }) => className ? (
          <code className={`${className} font-mono`}>{children}</code>
        ) : (
          <code className="break-words rounded bg-background/70 px-1 py-0.5 font-mono text-xs">{children}</code>
        ),
        table: ({ children }) => <div className="my-3 max-w-full overflow-x-auto rounded-md border"><table className="w-full min-w-max border-collapse text-left text-xs">{children}</table></div>,
        thead: ({ children }) => <thead className="bg-background/70">{children}</thead>,
        th: ({ children }) => <th className="border-b px-3 py-2 font-medium">{children}</th>,
        td: ({ children }) => <td className="border-b px-3 py-2 align-top last:border-b-0">{children}</td>,
        hr: () => <hr className="my-3 border-border" />,
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
