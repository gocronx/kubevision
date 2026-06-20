// SSE streaming client for the AI chat endpoints. We use fetch + ReadableStream
// rather than axios/EventSource so we can stream POST responses while still
// sending the Authorization header.

export interface StreamCallbacks {
  onEvent: (event: string, data: Record<string, unknown>) => void
  signal?: AbortSignal
}

/** POST a JSON body to an SSE endpoint and dispatch parsed events. Resolves when
 *  the stream closes; rejects on transport errors (aborts resolve quietly). */
export async function streamSSE(
  url: string,
  body: unknown,
  { onEvent, signal }: StreamCallbacks
): Promise<void> {
  const token = localStorage.getItem("token")
  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
    signal,
  })

  if (!res.ok || !res.body) {
    throw new Error(`AI request failed (${res.status})`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  try {
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })

      // Events are separated by a blank line.
      let sep: number
      while ((sep = buffer.indexOf("\n\n")) !== -1) {
        const raw = buffer.slice(0, sep)
        buffer = buffer.slice(sep + 2)
        dispatch(raw, onEvent)
      }
    }
    if (buffer.trim()) dispatch(buffer, onEvent)
  } catch (err) {
    if ((err as Error)?.name === "AbortError") return
    throw err
  }
}

function dispatch(raw: string, onEvent: StreamCallbacks["onEvent"]) {
  let event = "message"
  const dataLines: string[] = []
  for (const line of raw.split("\n")) {
    if (line.startsWith("event:")) {
      event = line.slice(6).trim()
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trim())
    }
  }
  if (dataLines.length === 0) return
  try {
    const data = JSON.parse(dataLines.join("\n")) as Record<string, unknown>
    onEvent(event, data)
  } catch {
    // Ignore malformed frames.
  }
}
