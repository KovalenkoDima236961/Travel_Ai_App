"use client";

import { FormEvent, useState } from "react";
import { usePathname } from "next/navigation";
import { useMutation } from "@tanstack/react-query";
import { useAuth } from "@/components/auth/AuthProvider";
import {
  recordAnalyticsEvent,
  submitAlphaFeedback,
  type AlphaFeedbackCategory
} from "@/lib/api/alpha";
import { getApiErrorMessage } from "@/shared/api/client";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import { cn } from "@/shared/lib/cn";

const categories: Array<{ value: AlphaFeedbackCategory; label: string }> = [
  { value: "bug", label: "Bug report" },
  { value: "ai", label: "AI issue" },
  { value: "feature_request", label: "Feature request" },
  { value: "performance", label: "Performance issue" },
  { value: "ui", label: "UI issue" },
  { value: "accessibility", label: "Accessibility" },
  { value: "security", label: "Security" },
  { value: "other", label: "Other" }
];

export function FeedbackDialog() {
  const pathname = usePathname();
  const { isAuthenticated, isLoading } = useAuth();
  const [open, setOpen] = useState(false);
  const [category, setCategory] = useState<AlphaFeedbackCategory>("bug");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [message, setMessage] = useState<string | null>(null);

  const hidden =
    isLoading ||
    !isAuthenticated ||
    pathname === "/login" ||
    pathname === "/register" ||
    pathname?.startsWith("/ops") ||
    pathname?.startsWith("/share/");

  const mutation = useMutation({
    mutationFn: () =>
      submitAlphaFeedback({
        category,
        title,
        description,
        metadata: { path: pathname ?? "unknown" }
      }),
    onSuccess: (result) => {
      const eventName =
        category === "bug"
          ? "bug_report_submitted"
          : category === "feature_request"
            ? "feature_request_submitted"
            : category === "ai"
              ? "ai_feedback_submitted"
              : "feedback_submitted";
      void recordAnalyticsEvent({
        eventName,
        feature: "feedback",
        entityType: "feedback",
        entityId: result.feedback.id,
        metadata: { category }
      }).catch(() => undefined);
      setTitle("");
      setDescription("");
      setMessage("Feedback sent.");
      setOpen(false);
    },
    onError: (error) => setMessage(getApiErrorMessage(error, "Could not send feedback."))
  });

  if (hidden) {
    return null;
  }

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage(null);
    mutation.mutate();
  }

  return (
    <>
      <button
        type="button"
        className={cn(
          "fixed bottom-5 right-5 z-40 inline-flex h-11 items-center justify-center rounded-full border border-slate-300 bg-white px-4 text-sm font-semibold text-slate-800 shadow-lg shadow-slate-900/10 transition hover:border-slate-500",
          open && "border-primary-600 text-primary-700"
        )}
        onClick={() => setOpen(true)}
      >
        Feedback
      </button>
      {message && !open ? (
        <div className="fixed bottom-[74px] right-5 z-40 max-w-[280px] rounded-lg border border-slate-200 bg-white px-4 py-3 text-sm text-slate-700 shadow-lg">
          {message}
        </div>
      ) : null}
      {open ? (
        <div className="fixed inset-0 z-50 flex items-end justify-center bg-slate-950/35 p-4 sm:items-center">
          <form
            className="w-full max-w-[480px] rounded-lg border border-slate-200 bg-white p-5 shadow-2xl"
            onSubmit={onSubmit}
          >
            <div className="flex items-start justify-between gap-4">
              <div>
                <h2 className="text-lg font-semibold text-slate-950">Send feedback</h2>
                <p className="mt-1 text-sm text-slate-500">
                  Private notes, prompts, receipts, and secrets are not accepted.
                </p>
              </div>
              <button
                type="button"
                className="rounded-md px-2 py-1 text-sm text-slate-500 hover:bg-slate-100"
                onClick={() => setOpen(false)}
              >
                Close
              </button>
            </div>
            <label className="mt-5 block text-sm font-medium text-slate-700">
              Category
              <Select
                className="mt-1"
                value={category}
                onChange={(event) => setCategory(event.target.value as AlphaFeedbackCategory)}
              >
                {categories.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </Select>
            </label>
            <label className="mt-4 block text-sm font-medium text-slate-700">
              Title
              <Input
                className="mt-1"
                maxLength={160}
                required
                value={title}
                onChange={(event) => setTitle(event.target.value)}
              />
            </label>
            <label className="mt-4 block text-sm font-medium text-slate-700">
              Details
              <Textarea
                className="mt-1 min-h-[150px]"
                maxLength={4000}
                required
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>
            {message && open ? (
              <div className="mt-4 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                {message}
              </div>
            ) : null}
            <div className="mt-5 flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setOpen(false)}>
                Cancel
              </Button>
              <Button disabled={mutation.isPending || !title.trim() || !description.trim()} type="submit">
                {mutation.isPending ? "Sending..." : "Send"}
              </Button>
            </div>
          </form>
        </div>
      ) : null}
    </>
  );
}
