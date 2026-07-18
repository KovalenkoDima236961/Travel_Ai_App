import type { TravelWarning } from "@/types/travel-day";

export function TravelWarningsCard({ warnings }: { warnings: TravelWarning[] }) {
  if (!warnings.length) return null;
  return <section className="rounded-2xl border border-[#EAD9B8] bg-[#FDF7E8] p-4"><h2 className="font-semibold text-[#7A5727]">Needs attention</h2><ul className="mt-2 space-y-2 text-sm text-[#7A5727]">{warnings.slice(0, 3).map((warning) => <li key={`${warning.scope}-${warning.entityId}`}>{warning.title}: {warning.message}</li>)}</ul></section>;
}
