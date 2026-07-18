export function ReceiptExportWarning() {
  return (
    <p className="mt-3 rounded-xl border border-amber-300 bg-amber-50 px-3.5 py-3 text-sm leading-6 text-amber-950">
      Receipt files can contain sensitive purchase details. They are excluded by default; include them only when you have a secure place to store the download.
    </p>
  );
}
