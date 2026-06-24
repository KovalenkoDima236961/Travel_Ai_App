export function downloadBlob(blob: Blob, filename: string): void {
  if (typeof window === "undefined" || typeof document === "undefined") {
    return;
  }

  const objectUrl = window.URL.createObjectURL(blob);
  const link = document.createElement("a");

  link.href = objectUrl;
  link.download = filename;
  link.style.display = "none";
  document.body.appendChild(link);
  link.click();
  link.remove();

  window.setTimeout(() => {
    window.URL.revokeObjectURL(objectUrl);
  }, 1000);
}

export function downloadTextFile(
  content: string,
  filename: string,
  mimeType: string
): void {
  downloadBlob(new Blob([content], { type: mimeType }), filename);
}
