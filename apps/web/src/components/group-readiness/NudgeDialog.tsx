"use client";

import { useMemo, useState } from "react";
import { Button } from "@/shared/ui/button";
import { useSendGroupReadinessNudge } from "@/hooks/useSendGroupReadinessNudge";
import type { ReadinessCategory } from "@/types/group-readiness";

const CATEGORY_OPTIONS: ReadinessCategory[] = [
  "availability",
  "polls",
  "checklist",
  "reminders",
  "settlements"
];

type NudgeDialogProps = {
  tripId: string;
  open: boolean;
  targetUserIds: string[];
  targetLabel: string;
  initialCategories: ReadinessCategory[];
  onClose: () => void;
};

export function NudgeDialog({
  tripId,
  open,
  targetUserIds,
  targetLabel,
  initialCategories,
  onClose
}: NudgeDialogProps) {
  const [message, setMessage] = useState("");
  const [selectedCategories, setSelectedCategories] = useState<ReadinessCategory[]>(
    initialCategories.length > 0 ? initialCategories : ["availability"]
  );
  const mutation = useSendGroupReadinessNudge(tripId);
  const canSubmit = targetUserIds.length > 0 && selectedCategories.length > 0 && !mutation.isPending;

  const title = useMemo(() => `Remind ${targetLabel}`, [targetLabel]);

  if (!open) {
    return null;
  }

  const toggleCategory = (category: ReadinessCategory) => {
    setSelectedCategories((current) =>
      current.includes(category)
        ? current.filter((item) => item !== category)
        : [...current, category]
    );
  };

  const send = async () => {
    if (!canSubmit) {
      return;
    }
    await mutation.mutateAsync({
      targetUserIds,
      categories: selectedCategories,
      message: message.trim() || undefined,
      dedupeWindowHours: 24
    });
    onClose();
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-cocoa-900/30 px-4">
      <div
        aria-modal="true"
        className="w-full max-w-[520px] rounded-[18px] border border-sand-300 bg-white p-5 shadow-xl"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-[18px] font-semibold text-cocoa-900">{title}</h3>
            <p className="mt-1 text-[13px] leading-[1.5] text-cocoa-500">
              To avoid spam, reminders are limited.
            </p>
          </div>
          <button
            type="button"
            className="rounded-full px-2 py-1 text-[18px] leading-none text-cocoa-400 hover:bg-sand-100 hover:text-cocoa-700"
            onClick={onClose}
            aria-label="Close nudge dialog"
          >
            x
          </button>
        </div>

        <div className="mt-5 space-y-4">
          <div>
            <p className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Categories
            </p>
            <div className="mt-2 flex flex-wrap gap-2">
              {CATEGORY_OPTIONS.map((category) => (
                <label
                  key={category}
                  className="inline-flex items-center gap-2 rounded-full border border-sand-300 bg-sand-50 px-3 py-1.5 text-[13px] font-medium text-cocoa-700"
                >
                  <input
                    checked={selectedCategories.includes(category)}
                    onChange={() => toggleCategory(category)}
                    type="checkbox"
                  />
                  {category.replaceAll("_", " ")}
                </label>
              ))}
            </div>
          </div>

          <label className="block">
            <span className="text-[12px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Message
            </span>
            <textarea
              className="mt-2 min-h-[96px] w-full rounded-[12px] border border-sand-300 bg-white p-3 text-[14px] text-cocoa-800 outline-none transition focus:border-clay"
              maxLength={500}
              onChange={(event) => setMessage(event.target.value)}
              placeholder="Please update your trip readiness items when you have a moment."
              value={message}
            />
          </label>
        </div>

        {mutation.error instanceof Error ? (
          <p className="mt-3 text-[13px] text-[#B3402E]">{mutation.error.message}</p>
        ) : null}

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button disabled={!canSubmit} onClick={send}>
            {mutation.isPending ? "Sending..." : "Send reminder"}
          </Button>
        </div>
      </div>
    </div>
  );
}
