"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { EmptyState } from "@/components/ui";
import { TripHealthIssueCard } from "./TripHealthIssueCard";
import { categoryLabel, severityRank } from "./health-ui";
import type { TripHealthCategory, TripHealthIssue } from "@/types/trip-health";

type FilterMode = "all" | "high" | "actionable";

export function TripHealthIssueList({ issues }: { issues: TripHealthIssue[] }) {
  const emptyT = useTranslations("emptyStates.health");
  const [mode, setMode] = useState<FilterMode>("all");
  const [category, setCategory] = useState<TripHealthCategory | "all">("all");
  const categories = useMemo(
    () => Array.from(new Set(issues.map((issue) => issue.category))).sort(),
    [issues]
  );
  const filtered = useMemo(
    () =>
      issues.filter((issue) => {
        if (category !== "all" && issue.category !== category) {
          return false;
        }
        if (mode === "high") {
          return issue.severity === "critical" || issue.severity === "high";
        }
        if (mode === "actionable") {
          return Boolean(issue.action?.href);
        }
        return true;
      }),
    [category, issues, mode]
  );
  const grouped = useMemo(() => groupIssues(filtered), [filtered]);

  return (
    <section className="rounded-[18px] border border-sand-300 bg-white p-5">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 className="font-newsreader text-[22px] font-semibold text-cocoa-900">
            Open Issues
          </h2>
          <p className="mt-1 text-[13px] text-cocoa-400">
            {filtered.length} shown from {issues.length}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <FilterButton active={mode === "all"} onClick={() => setMode("all")}>
            All
          </FilterButton>
          <FilterButton active={mode === "high"} onClick={() => setMode("high")}>
            Critical/High
          </FilterButton>
          <FilterButton
            active={mode === "actionable"}
            onClick={() => setMode("actionable")}
          >
            Actionable
          </FilterButton>
          <select
            value={category}
            onChange={(event) => setCategory(event.target.value as TripHealthCategory | "all")}
            className="h-9 rounded-full border border-sand-300 bg-white px-3 text-[13px] font-medium text-cocoa-700 outline-none transition focus:border-clay"
          >
            <option value="all">All categories</option>
            {categories.map((item) => (
              <option key={item} value={item}>
                {categoryLabel[item]}
              </option>
            ))}
          </select>
        </div>
      </div>

      {issues.length === 0 ? (
        <EmptyState
          className="mt-4 border-sand-300 bg-sand-50"
          compact
          description={emptyT("description")}
          title={emptyT("title")}
        />
      ) : filtered.length === 0 ? (
        <div className="mt-4 rounded-[14px] border border-sand-200 bg-sand-50 p-4 text-[14px] text-cocoa-500">
          No issues match this filter.
        </div>
      ) : (
        <div className="mt-5 flex flex-col gap-5">
          {grouped.map((group) => (
            <div key={group.category}>
              <h3 className="mb-3 text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
                {categoryLabel[group.category]}
              </h3>
              <div className="flex flex-col gap-3">
                {group.issues.map((issue) => (
                  <TripHealthIssueCard key={issue.id} issue={issue} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function FilterButton({
  active,
  children,
  onClick
}: {
  active: boolean;
  children: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        active
          ? "h-9 rounded-full bg-cocoa-900 px-3.5 text-[13px] font-semibold text-sand-100"
          : "h-9 rounded-full border border-sand-300 bg-white px-3.5 text-[13px] font-medium text-cocoa-600 transition hover:border-sand-500 hover:text-cocoa-900"
      }
    >
      {children}
    </button>
  );
}

function groupIssues(issues: TripHealthIssue[]) {
  const groups = new Map<TripHealthCategory, TripHealthIssue[]>();
  for (const issue of issues) {
    const current = groups.get(issue.category) ?? [];
    current.push(issue);
    groups.set(issue.category, current);
  }
  return Array.from(groups.entries())
    .map(([category, items]) => ({
      category,
      issues: items.sort((left, right) => {
        if (severityRank[left.severity] !== severityRank[right.severity]) {
          return severityRank[right.severity] - severityRank[left.severity];
        }
        return left.id.localeCompare(right.id);
      })
    }))
    .sort((left, right) => left.category.localeCompare(right.category));
}
