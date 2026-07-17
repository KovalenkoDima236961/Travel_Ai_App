import { WhyThisFitsYou } from "./WhyThisFitsYou";
import type { ReactNode } from "react";
import type { WhyThisFitsYou as Fit } from "@/types/personalization";

export function PersonalizedSuggestionCard({ title, fit, children }: { title: string; fit: Fit; children: ReactNode }) { return <article className="rounded-[18px] border border-sand-300 bg-white p-4"><h3 className="font-semibold text-cocoa-900">{title}</h3><div className="mt-3">{children}</div><div className="mt-3"><WhyThisFitsYou fit={fit} /></div></article>; }
