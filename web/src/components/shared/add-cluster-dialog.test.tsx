import { vi } from "vitest"
import { readFileAsText } from "@/lib/read-file"

describe("readFileAsText", () => {
  it("loads an extensionless kubeconfig without using File.text", async () => {
    const kubeconfig = [
      "apiVersion: v1",
      "kind: Config",
      "clusters: []",
      "contexts: []",
      "users: []",
    ].join("\n")
    const file = new File([kubeconfig], "config", { type: "application/octet-stream" })
    Object.defineProperty(file, "text", { value: undefined })
    class TestFileReader {
      result: string | ArrayBuffer | null = null
      error: DOMException | null = null
      onload: ((event: ProgressEvent<FileReader>) => void) | null = null
      onerror: ((event: ProgressEvent<FileReader>) => void) | null = null
      onabort: ((event: ProgressEvent<FileReader>) => void) | null = null

      readAsText() {
        this.result = kubeconfig
        queueMicrotask(() => {
          const event = new ProgressEvent("load") as ProgressEvent<FileReader>
          this.onload?.(event)
        })
      }
    }
    vi.stubGlobal("FileReader", TestFileReader)

    await expect(readFileAsText(file)).resolves.toBe(kubeconfig)
  })
})
