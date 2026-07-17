import type { WhyThisFitsYou as WhyThisFitsYouModel } from "@/types/personalization";

export function WhyThisFitsYou({ fit, title = "Why this fits you" }: { fit: WhyThisFitsYouModel; title?: string }) {
  if (fit.reasons.length === 0 && (!fit.concerns || fit.concerns.length === 0)) return null;
  return (
    <section className="rounded-[14px] border border-[#D9E1D2] bg-[#F5F8F1] p-3.5">
      <div className="flex items-center justify-between gap-3">
        <h4 className="text-[12px] font-bold uppercase tracking-[0.08em] text-cocoa-500">{title}</h4>
        <span className="rounded-full bg-white px-2 py-0.5 text-[11px] font-bold text-cocoa-700">{fit.score}% match</span>
      </div>
      <ul className="mt-2 space-y-1 text-[13px] leading-5 text-cocoa-700">
        {fit.reasons.map((reason) => <li key={reason}>• {reason}</li>)}
      </ul>
      {fit.concerns?.length ? <p className="mt-2 text-[12px] leading-5 text-cocoa-500">Consider: {fit.concerns.join(" ")}</p> : null}
    </section>
  );
}
