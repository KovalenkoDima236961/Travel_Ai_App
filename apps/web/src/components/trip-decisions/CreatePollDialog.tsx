"use client";

import { FormEvent, useMemo, useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Textarea } from "@/shared/ui/textarea";
import type { CreateTripPollInput, TripPollType } from "@/types/trip-decisions";

const QUICK_TEMPLATES: Array<{
  label: string;
  input: Pick<CreateTripPollInput, "title" | "pollType" | "metadata" | "options">;
}> = [
  {
    label: "Destination choice",
    input: {
      title: "Which destination should we choose?",
      pollType: "single_choice",
      metadata: { category: "destination" },
      options: [
        { label: "Vienna", metadata: { category: "destination", destination: "Vienna" } },
        { label: "Ljubljana", metadata: { category: "destination", destination: "Ljubljana" } }
      ]
    }
  },
  {
    label: "Transport mode",
    input: {
      title: "Which transport mode should we prefer?",
      pollType: "single_choice",
      metadata: { category: "transport" },
      options: [
        { label: "Train", optionKey: "train", metadata: { category: "transport", mode: "train" } },
        { label: "Car", optionKey: "car", metadata: { category: "transport", mode: "car" } },
        { label: "Bus", optionKey: "bus", metadata: { category: "transport", mode: "bus" } }
      ]
    }
  },
  {
    label: "Activities",
    input: {
      title: "Which activities are must-haves?",
      pollType: "multiple_choice",
      options: [{ label: "Food tour" }, { label: "Museum visit" }, { label: "Viewpoint walk" }]
    }
  },
  {
    label: "Date choice",
    input: {
      title: "Which dates work best?",
      pollType: "date_choice",
      metadata: { category: "date" },
      options: [{ label: "Sep 10-14" }, { label: "Sep 17-21" }, { label: "Flexible" }]
    }
  },
  {
    label: "Accommodation",
    input: {
      title: "Which accommodation style should we book?",
      pollType: "single_choice",
      metadata: { category: "accommodation" },
      options: [{ label: "Central hotel" }, { label: "Apartment" }, { label: "Budget hostel" }]
    }
  },
  {
    label: "Budget",
    input: {
      title: "Which budget level feels right?",
      pollType: "single_choice",
      metadata: { category: "budget" },
      options: [{ label: "Keep it lean" }, { label: "Balanced" }, { label: "Comfort upgrade" }]
    }
  }
];

type CreatePollDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (input: CreateTripPollInput) => Promise<void>;
  isPending?: boolean;
  error?: string | null;
};

export function CreatePollDialog({
  open,
  onOpenChange,
  onCreate,
  isPending = false,
  error = null
}: CreatePollDialogProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [pollType, setPollType] = useState<TripPollType>("single_choice");
  const [optionsText, setOptionsText] = useState("Option 1\nOption 2");
  const [closesAt, setClosesAt] = useState("");
  const [allowMultipleVotes, setAllowMultipleVotes] = useState(false);
  const [metadata, setMetadata] = useState<Record<string, unknown> | undefined>(undefined);

  const options = useMemo(
    () =>
      optionsText
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean)
        .map((label) => ({ label })),
    [optionsText]
  );

  if (!open) {
    return null;
  }

  function applyTemplate(index: number) {
    const template = QUICK_TEMPLATES[index];
    if (!template) {
      return;
    }
    setTitle(template.input.title);
    setPollType(template.input.pollType);
    setOptionsText(template.input.options.map((option) => option.label).join("\n"));
    setAllowMultipleVotes(template.input.pollType === "multiple_choice");
    setMetadata(template.input.metadata);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onCreate({
      title,
      description,
      pollType,
      allowMultipleVotes,
      closesAt: closesAt ? new Date(closesAt).toISOString() : undefined,
      metadata,
      options
    });
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-slate-950/40 px-4 py-10">
      <div className="w-full max-w-2xl rounded-[18px] bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-950">Create poll</h2>
          </div>
          <Button onClick={() => onOpenChange(false)} type="button" variant="ghost">
            Close
          </Button>
        </div>

        <div className="mt-4 flex flex-wrap gap-2">
          {QUICK_TEMPLATES.map((template, index) => (
            <button
              key={template.label}
              className="rounded-full border border-sand-300 px-3 py-1.5 text-[12px] font-semibold text-cocoa-500 transition hover:border-clay hover:text-clay"
              onClick={() => applyTemplate(index)}
              type="button"
            >
              {template.label}
            </button>
          ))}
        </div>

        <form className="mt-5 space-y-4" onSubmit={submit}>
          <label className="block text-sm font-semibold text-slate-700">
            Title
            <Input value={title} onChange={(event) => setTitle(event.target.value)} required />
          </label>
          <label className="block text-sm font-semibold text-slate-700">
            Description
            <Textarea
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </label>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="block text-sm font-semibold text-slate-700">
              Type
              <Select
                value={pollType}
                onChange={(event) => setPollType(event.target.value as TripPollType)}
              >
                <option value="single_choice">Single choice</option>
                <option value="multiple_choice">Multiple choice</option>
                <option value="yes_no">Yes/No</option>
                <option value="date_choice">Date choice</option>
                <option value="rating">Rating</option>
              </Select>
            </label>
            <label className="block text-sm font-semibold text-slate-700">
              Closes at
              <Input
                type="datetime-local"
                value={closesAt}
                onChange={(event) => setClosesAt(event.target.value)}
              />
            </label>
          </div>
          <label className="block text-sm font-semibold text-slate-700">
            Options
            <Textarea
              value={optionsText}
              onChange={(event) => setOptionsText(event.target.value)}
              placeholder="One option per line"
            />
          </label>
          {pollType === "multiple_choice" ? (
            <label className="flex items-center gap-2 text-sm font-medium text-slate-700">
              <input
                checked={allowMultipleVotes}
                onChange={(event) => setAllowMultipleVotes(event.target.checked)}
                type="checkbox"
              />
              Allow multiple selected options
            </label>
          ) : null}
          {error ? (
            <div className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
              {error}
            </div>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button onClick={() => onOpenChange(false)} type="button" variant="secondary">
              Cancel
            </Button>
            <Button disabled={isPending || options.length === 0 || title.trim().length < 2} type="submit">
              {isPending ? "Creating..." : "Create poll"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
