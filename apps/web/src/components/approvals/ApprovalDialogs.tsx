"use client";

import { FormEvent, ReactNode, useEffect, useState } from "react";

import { ApprovalChecklist } from "@/components/approvals/ApprovalChecklist";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Textarea } from "@/components/ui/Textarea";
import type { ApprovalChecklist as ApprovalChecklistData } from "@/types/approval";

function DialogShell({
  title,
  onClose,
  children
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4">
      <Card className="max-h-[90vh] w-full max-w-lg overflow-y-auto">
        <div className="flex items-start justify-between gap-4">
          <h2 className="text-lg font-semibold text-slate-950">{title}</h2>
          <Button onClick={onClose} size="sm" type="button" variant="ghost">
            Close
          </Button>
        </div>
        {children}
      </Card>
    </div>
  );
}

function DialogError({ error }: { error?: string | null }) {
  if (!error) {
    return null;
  }
  return <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>;
}

export function SubmitForApprovalDialog({
  open,
  checklist,
  isSubmitting = false,
  error,
  onClose,
  onSubmit
}: {
  open: boolean;
  checklist?: ApprovalChecklistData;
  isSubmitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: { note?: string; acknowledgedWarnings: string[] }) => void;
}) {
  const [note, setNote] = useState("");
  const [acknowledged, setAcknowledged] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (open) {
      setNote("");
      setAcknowledged({});
    }
  }, [open]);

  if (!open) {
    return null;
  }

  const warnings = checklist?.items.filter((item) => item.status === "warning") ?? [];
  const hasBlocker = (checklist?.blockerCount ?? 0) > 0;

  function submit(event: FormEvent) {
    event.preventDefault();
    if (hasBlocker) {
      return;
    }
    const acknowledgedWarnings = warnings
      .filter((item) => acknowledged[item.key])
      .map((item) => item.key);
    onSubmit({ note: note.trim() || undefined, acknowledgedWarnings });
  }

  return (
    <DialogShell title="Submit for approval" onClose={onClose}>
      <form className="mt-4 space-y-4" onSubmit={submit}>
        {checklist ? <ApprovalChecklist checklist={checklist} /> : null}
        {hasBlocker ? (
          <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
            This trip cannot be submitted yet. Resolve the blocker above first.
          </p>
        ) : null}
        {warnings.length > 0 ? (
          <fieldset className="space-y-2">
            <legend className="text-sm font-medium text-slate-700">
              Acknowledge warnings (optional)
            </legend>
            {warnings.map((item) => (
              <label key={item.key} className="flex items-start gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  className="mt-1"
                  checked={Boolean(acknowledged[item.key])}
                  onChange={(event) =>
                    setAcknowledged((prev) => ({ ...prev, [item.key]: event.target.checked }))
                  }
                />
                <span>{item.title}</span>
              </label>
            ))}
          </fieldset>
        ) : null}
        <label className="block text-sm font-medium text-slate-700">
          Note (optional)
          <Textarea
            className="mt-2"
            maxLength={1000}
            placeholder="Anything reviewers should know?"
            value={note}
            onChange={(event) => setNote(event.target.value)}
          />
        </label>
        <DialogError error={error} />
        <div className="flex justify-end gap-2">
          <Button onClick={onClose} type="button" variant="secondary">
            Cancel
          </Button>
          <Button disabled={hasBlocker || isSubmitting} type="submit">
            {isSubmitting ? "Submitting…" : "Submit for approval"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

export function ApproveTripDialog({
  open,
  isSubmitting = false,
  error,
  onClose,
  onSubmit
}: {
  open: boolean;
  isSubmitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: { decisionNote?: string }) => void;
}) {
  const [decisionNote, setDecisionNote] = useState("");

  useEffect(() => {
    if (open) {
      setDecisionNote("");
    }
  }, [open]);

  if (!open) {
    return null;
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    onSubmit({ decisionNote: decisionNote.trim() || undefined });
  }

  return (
    <DialogShell title="Approve trip" onClose={onClose}>
      <form className="mt-4 space-y-4" onSubmit={submit}>
        <label className="block text-sm font-medium text-slate-700">
          Decision note (optional)
          <Textarea
            className="mt-2"
            maxLength={1000}
            value={decisionNote}
            onChange={(event) => setDecisionNote(event.target.value)}
          />
        </label>
        <DialogError error={error} />
        <div className="flex justify-end gap-2">
          <Button onClick={onClose} type="button" variant="secondary">
            Cancel
          </Button>
          <Button disabled={isSubmitting} type="submit">
            {isSubmitting ? "Approving…" : "Approve trip"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

export function RequestChangesDialog({
  open,
  isSubmitting = false,
  error,
  onClose,
  onSubmit
}: {
  open: boolean;
  isSubmitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: { decisionNote: string }) => void;
}) {
  const [decisionNote, setDecisionNote] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setDecisionNote("");
      setLocalError(null);
    }
  }, [open]);

  if (!open) {
    return null;
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    const trimmed = decisionNote.trim();
    if (!trimmed) {
      setLocalError("A note is required so the submitter knows what to change.");
      return;
    }
    if (trimmed.length > 1000) {
      setLocalError("Note must be at most 1000 characters.");
      return;
    }
    onSubmit({ decisionNote: trimmed });
  }

  return (
    <DialogShell title="Request changes" onClose={onClose}>
      <form className="mt-4 space-y-4" onSubmit={submit}>
        <label className="block text-sm font-medium text-slate-700">
          What needs to change?
          <Textarea
            className="mt-2"
            maxLength={1000}
            required
            value={decisionNote}
            onChange={(event) => setDecisionNote(event.target.value)}
          />
        </label>
        <DialogError error={localError ?? error} />
        <div className="flex justify-end gap-2">
          <Button onClick={onClose} type="button" variant="secondary">
            Cancel
          </Button>
          <Button disabled={isSubmitting} type="submit" variant="danger">
            {isSubmitting ? "Sending…" : "Request changes"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}

export function CancelApprovalDialog({
  open,
  isSubmitting = false,
  error,
  onClose,
  onSubmit
}: {
  open: boolean;
  isSubmitting?: boolean;
  error?: string | null;
  onClose: () => void;
  onSubmit: (input: { note?: string }) => void;
}) {
  const [note, setNote] = useState("");

  useEffect(() => {
    if (open) {
      setNote("");
    }
  }, [open]);

  if (!open) {
    return null;
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    onSubmit({ note: note.trim() || undefined });
  }

  return (
    <DialogShell title="Cancel submission" onClose={onClose}>
      <form className="mt-4 space-y-4" onSubmit={submit}>
        <p className="text-sm text-slate-600">
          This withdraws the pending submission. The trip returns to draft-like editing and can
          be resubmitted later.
        </p>
        <label className="block text-sm font-medium text-slate-700">
          Reason (optional)
          <Textarea
            className="mt-2"
            maxLength={1000}
            value={note}
            onChange={(event) => setNote(event.target.value)}
          />
        </label>
        <DialogError error={error} />
        <div className="flex justify-end gap-2">
          <Button onClick={onClose} type="button" variant="secondary">
            Keep submission
          </Button>
          <Button disabled={isSubmitting} type="submit" variant="danger">
            {isSubmitting ? "Cancelling…" : "Cancel submission"}
          </Button>
        </div>
      </form>
    </DialogShell>
  );
}
