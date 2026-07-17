import Link from "next/link";

export function PreferenceReviewPrompt({ compact = false }: { compact?: boolean }) {
  return <div className={`rounded-[16px] border border-sand-300 bg-sand-50 ${compact ? "p-3" : "p-4"}`}><p className="text-sm font-semibold text-cocoa-800">Make suggestions more personal</p><p className="mt-1 text-[13px] text-cocoa-500">Add a few travel preferences — you can change or override them any time.</p><Link href="/settings?section=preferences" className="mt-2 inline-block text-[13px] font-semibold text-clay-deep hover:underline">Review preferences</Link></div>;
}
