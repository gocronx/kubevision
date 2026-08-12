export function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === "string") resolve(reader.result)
      else reject(new Error("File reader returned a non-text result"))
    }
    reader.onerror = () => reject(reader.error ?? new Error("Failed to read file"))
    reader.onabort = () => reject(new Error("File read was aborted"))
    reader.readAsText(file)
  })
}
