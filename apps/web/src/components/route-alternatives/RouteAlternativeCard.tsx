"use client";

import { formatMoney } from "@/entities/budget/model";
import { transportModeLabel } from "@/components/routes/route-options";
import { FeedbackChips, WhyThisFitsYou } from "@/components/personalization";
import type { RouteAlternative } from "@/types/route-alternatives";
import { RouteAlternativeMapPreview } from "./RouteAlternativeMapPreview";
import { RouteAlternativeScores } from "./RouteAlternativeScores";

type RouteAlternativeCardProps = {
  alternative: RouteAlternative;
  selected?: boolean;
  canCreateTrip?: boolean;
  canApply?: boolean;
  canCreatePoll?: boolean;
  onSelect?: (alternative: RouteAlternative) => void;
  onCreateTrip?: (alternative: RouteAlternative) => void;
  onApply?: (alternative: RouteAlternative) => void;
  onMoreLikeThis?: (alternative: RouteAlternative) => void;
};

export function RouteAlternativeCard({
  alternative,
  selected = false,
  canCreateTrip = false,
  canApply = false,
  canCreatePoll = false,
  onSelect,
  onCreateTrip,
  onApply,
  onMoreLikeThis
}: RouteAlternativeCardProps) {
  const modes = Array.from(new Set((alternative.route.legs ?? []).map((leg) => leg.mode).filter(Boolean)));
  const budget = alternative.estimatedBudget;

  return (
    <article
      className={
        selected
          ? "rounded-[18px] border border-clay bg-white p-5 shadow-[0_12px_30px_rgba(192,91,59,0.14)]"
          : "rounded-[18px] border border-sand-300 bg-white p-5 shadow-[0_1px_2px_rgba(34,26,20,0.04)]"
      }
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="font-newsreader text-[23px] font-semibold text-cocoa-900">
              {alternative.title}
            </h3>
            <span className="rounded-full bg-sand-200 px-2.5 py-1 text-[11px] font-bold uppercase tracking-[0.08em] text-cocoa-500">
              {alternative.difficulty}
            </span>
          </div>
          <p className="mt-1 text-[13.5px] leading-5 text-cocoa-500">
            {alternative.summary}
          </p>
        </div>
        <div className="text-right">
          <p className="text-[11px] font-bold uppercase tracking-[0.1em] text-[#A08D78]">
            Overall fit
          </p>
          <p className="font-newsreader text-[34px] font-semibold leading-none text-clay">
            {alternative.scores.overallFit}
          </p>
        </div>
      </div>

      <div className="mt-4">
        <RouteAlternativeMapPreview route={alternative.route} selected={selected} />
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-3">
        <Metric label="Estimated budget" value={formatMoney(budget?.amount, budget?.currency)} />
        <Metric label="Transfer time" value={formatDuration(alternative.estimatedTransferMinutes)} />
        <Metric
          label="Transport"
          value={modes.length > 0 ? modes.map(transportModeLabel).join(", ") : "Flexible"}
        />
      </div>

      <div className="mt-4">
        <RouteAlternativeScores scores={alternative.scores} compact />
      </div>

      {alternative.personalizationFit ? <div className="mt-4"><WhyThisFitsYou fit={{ score: alternative.personalizationFit.score, reasons: alternative.personalizationFit.reasons, concerns: alternative.personalizationFit.concerns }} /></div> : null}

      {alternative.bestFor.length > 0 ? (
        <div className="mt-4 flex flex-wrap gap-2">
          {alternative.bestFor.map((tag) => (
            <span
              key={tag}
              className="rounded-full border border-sand-300 bg-sand-50 px-2.5 py-1 text-[12px] font-semibold text-cocoa-500"
            >
              {tag}
            </span>
          ))}
        </div>
      ) : null}

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        <List title="Pros" items={alternative.pros} />
        <List title="Cons" items={alternative.cons} />
      </div>

      {alternative.warnings.length > 0 ? (
        <div className="mt-4 rounded-[12px] border border-amber-200 bg-amber-50 px-3 py-2 text-[12.5px] leading-5 text-amber-900">
          {alternative.warnings.join(" ")}
        </div>
      ) : null}

      <div className="mt-4 border-t border-sand-200 pt-3">
        <FeedbackChips input={{ entityType: "route_alternative", entityId: alternative.id, metadata: { transport: modes, source: "route_alternatives" } }} chips={[{ type: "too_many_transfers", label: "Too many transfers" }, { type: "prefer_trains", label: "Prefer trains" }, { type: "more_nature", label: "More nature" }, { type: "too_expensive", label: "Cheaper route" }]} />
      </div>

      <div className="mt-5 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={() => onSelect?.(alternative)}
          className="h-10 rounded-full border border-sand-300 px-4 text-[13px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay"
        >
          Compare
        </button>
        {onMoreLikeThis ? (
          <button
            type="button"
            onClick={() => onMoreLikeThis(alternative)}
            className="h-10 rounded-full border border-sand-300 px-4 text-[13px] font-semibold text-cocoa-600 transition hover:border-clay hover:text-clay"
          >
            More like this
          </button>
        ) : null}
        {canCreatePoll ? (
          <span className="h-10 rounded-full bg-sand-100 px-4 py-2.5 text-[12px] font-semibold text-cocoa-500">
            Poll-ready
          </span>
        ) : null}
        {canCreateTrip ? (
          <button
            type="button"
            onClick={() => onCreateTrip?.(alternative)}
            className="h-10 rounded-full bg-clay px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-clay-dark"
          >
            Use this route
          </button>
        ) : null}
        {canApply ? (
          <button
            type="button"
            onClick={() => onApply?.(alternative)}
            className="h-10 rounded-full bg-cocoa-900 px-4 text-[13px] font-semibold text-sand-100 transition hover:bg-cocoa-700"
          >
            Apply route
          </button>
        ) : null}
      </div>
    </article>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[12px] bg-sand-50 px-3 py-2">
      <p className="text-[11px] font-bold uppercase tracking-[0.08em] text-[#A08D78]">{label}</p>
      <p className="mt-1 text-[13px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}

function List({ title, items }: { title: string; items: string[] }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <div>
      <p className="text-[12px] font-bold uppercase tracking-[0.08em] text-[#A08D78]">{title}</p>
      <ul className="mt-1 space-y-1 text-[13px] leading-5 text-cocoa-500">
        {items.slice(0, 3).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function formatDuration(minutes: number | null | undefined) {
  if (!minutes || minutes <= 0) {
    return "—";
  }
  if (minutes < 60) {
    return `${minutes} min`;
  }
  const hours = Math.floor(minutes / 60);
  const remainder = minutes % 60;
  return remainder === 0 ? `${hours} hr` : `${hours} hr ${remainder} min`;
}
