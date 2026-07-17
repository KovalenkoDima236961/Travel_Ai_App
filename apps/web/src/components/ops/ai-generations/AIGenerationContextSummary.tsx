export function AIGenerationContextSummary({ title, value }: { title: string; value?: Record<string, unknown> | null }) {
  if (!value) return null;
  return <section><h3 className="text-[14px] font-semibold text-cocoa-800">{title}</h3><dl className="mt-3 grid gap-x-5 gap-y-2 sm:grid-cols-2">{Object.entries(value).map(([key, entry]) => <div key={key} className="flex justify-between gap-4 border-b border-sand-200 pb-1.5 text-[12.5px]"><dt className="text-cocoa-400">{key}</dt><dd className="max-w-[65%] break-words text-right text-cocoa-700">{typeof entry === "object" ? JSON.stringify(entry) : String(entry)}</dd></div>)}</dl></section>;
}
