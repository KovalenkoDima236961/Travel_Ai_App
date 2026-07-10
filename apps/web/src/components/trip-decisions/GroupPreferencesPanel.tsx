"use client";

import { useGroupPreferences } from "@/hooks/useGroupPreferences";

type GroupPreferencesPanelProps = {
  tripId: string;
  enabled?: boolean;
};

export function GroupPreferencesPanel({ tripId, enabled = true }: GroupPreferencesPanelProps) {
  const summaryQuery = useGroupPreferences(tripId, enabled);
  const summary = summaryQuery.data;

  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h2 className="font-newsreader text-[24px] font-semibold text-cocoa-900">
            Group preferences
          </h2>
        </div>
      </div>

      {summaryQuery.isLoading ? (
        <p className="mt-4 text-[13px] text-cocoa-400">Loading group preferences...</p>
      ) : null}

      {summary ? (
        <div className="mt-4 space-y-4">
          <div className="grid gap-2 sm:grid-cols-4">
            <Metric label="Open polls" value={summary.summary.openPollCount} />
            <Metric label="Reactions" value={summary.summary.reactionCount} />
            <Metric label="Must-have" value={summary.summary.mustHaveItemCount} />
            <Metric label="Skip" value={summary.summary.skipItemCount} />
          </div>

          <div className="rounded-[14px] bg-sand-50 p-4 text-[13px] leading-5 text-cocoa-600">
            {summary.aiConstraintSummary || "There is no clear group winner yet."}
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <PreferenceList
              empty="No top poll choices yet."
              items={summary.topPollChoices.map((choice) => ({
                key: choice.pollId,
                label: choice.title,
                value: choice.winningOptions.map((option) => option.label).join(", ")
              }))}
              title="Top choices"
            />
            <PreferenceList
              empty="No preferred transport yet."
              items={summary.transportPreferences.map((item) => ({
                key: item.key,
                label: item.label,
                value: `${item.votes} votes`
              }))}
              title="Transport"
            />
            <PreferenceList
              empty="No must-have activities yet."
              items={summary.itineraryPreferences.mustHaveItems.map((item) => ({
                key: `${item.dayNumber}:${item.itemIndex}`,
                label: item.name || `Day ${item.dayNumber} item ${item.itemIndex + 1}`,
                value: `${item.count} reactions`
              }))}
              title="Must-have activities"
            />
            <PreferenceList
              empty="No skip candidates yet."
              items={summary.itineraryPreferences.mostSkippedItems.map((item) => ({
                key: `${item.dayNumber}:${item.itemIndex}`,
                label: item.name || `Day ${item.dayNumber} item ${item.itemIndex + 1}`,
                value: `${item.count} reactions`
              }))}
              title="Skip candidates"
            />
          </div>
        </div>
      ) : summaryQuery.isError ? (
        <p className="mt-4 text-[13px] text-red-700">Could not load group preferences.</p>
      ) : null}
    </section>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-[14px] bg-sand-50 p-3">
      <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        {label}
      </p>
      <p className="mt-1 font-newsreader text-[25px] font-semibold text-cocoa-900">{value}</p>
    </div>
  );
}

function PreferenceList({
  title,
  items,
  empty
}: {
  title: string;
  items: Array<{ key: string; label: string; value: string }>;
  empty: string;
}) {
  return (
    <div>
      <h3 className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
        {title}
      </h3>
      <div className="mt-2 space-y-2">
        {items.length > 0 ? (
          items.map((item) => (
            <div
              key={item.key}
              className="flex items-center justify-between gap-3 rounded-[12px] border border-sand-300 px-3 py-2 text-[13px]"
            >
              <span className="min-w-0 truncate font-semibold text-cocoa-800">{item.label}</span>
              <span className="shrink-0 text-cocoa-400">{item.value}</span>
            </div>
          ))
        ) : (
          <p className="rounded-[12px] bg-sand-50 px-3 py-2 text-[13px] text-cocoa-400">
            {empty}
          </p>
        )}
      </div>
    </div>
  );
}
