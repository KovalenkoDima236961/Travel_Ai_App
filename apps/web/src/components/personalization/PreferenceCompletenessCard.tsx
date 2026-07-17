import Link from "next/link";
import type { PreferenceCompleteness } from "@/types/personalization";

export function PreferenceCompletenessCard({ value }: { value: PreferenceCompleteness }) {
  const tone = value.score >= 70 ? "bg-[#EDF4E8]" : "bg-[#FFF4E9]";
  return (
    <section className={`rounded-[20px] border border-sand-300 p-6 ${tone}`}>
      <div className="flex items-start justify-between gap-4">
        <div><h2 className="font-newsreader text-2xl font-medium text-cocoa-900">Personalization</h2><p className="mt-1 text-sm text-cocoa-600">Uses your preferences, trip history, and feedback inside this app — never for advertising.</p></div>
        <div className="rounded-full bg-white px-3 py-1.5 text-sm font-bold text-cocoa-800">{value.score}%</div>
      </div>
      {value.missingFields.length ? <p className="mt-4 text-sm text-cocoa-700">Add {value.missingFields.slice(0, 2).map((field) => field.label).join(" and ")} to improve your suggestions.</p> : <p className="mt-4 text-sm text-cocoa-700">Your saved preferences are ready to guide recommendations.</p>}
      {value.recommendedActions[0] ? <Link className="mt-4 inline-flex rounded-full bg-cocoa-900 px-4 py-2 text-[13px] font-semibold text-sand-100" href={value.recommendedActions[0].href}>{value.recommendedActions[0].label}</Link> : null}
    </section>
  );
}
