import Link from "next/link";
import { WhyThisFitsYou } from "./WhyThisFitsYou";
import type { WhyThisFitsYou as Fit } from "@/types/personalization";

type Template = { id: string; title: string; durationDays: number };
export function RecommendedTemplatesSection({ items }: { items: Array<{ template: Template; fitScore: number; whyThisFitsYou: Fit }> }) {
  if (!items.length) return null;
  return <section><div className="flex items-center justify-between"><h2 className="font-newsreader text-2xl text-cocoa-900">Recommended templates</h2><Link href="/templates" className="text-sm font-semibold text-clay-deep">View all</Link></div><div className="mt-4 grid gap-3 md:grid-cols-2">{items.map(({ template, fitScore, whyThisFitsYou }) => <Link href={`/templates/${template.id}`} key={template.id} className="rounded-[18px] border border-sand-300 bg-white p-4 hover:border-sand-500"><div className="flex justify-between gap-3"><h3 className="font-semibold text-cocoa-900">{template.title}</h3><span className="text-xs font-bold text-cocoa-600">{fitScore}% fit</span></div><p className="mt-1 text-sm text-cocoa-500">{template.durationDays} days</p><div className="mt-3"><WhyThisFitsYou fit={whyThisFitsYou} title="Why this template fits" /></div></Link>)}</div></section>;
}
