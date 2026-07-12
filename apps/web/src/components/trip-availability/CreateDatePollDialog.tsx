"use client";

import { useState } from "react";
import { Button } from "@/shared/ui/button";
import { Input } from "@/shared/ui/input";

type CreateDatePollDialogProps = {
  isPending?: boolean;
  open: boolean;
  optionCount: number;
  onCreate: (title: string) => void;
  onOpenChange: (open: boolean) => void;
};

export function CreateDatePollDialog({
  isPending = false,
  open,
  optionCount,
  onCreate,
  onOpenChange
}: CreateDatePollDialogProps) {
  const [title, setTitle] = useState("Which dates should we choose?");

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 px-4">
      <div className="w-full max-w-md rounded-[18px] border border-sand-300 bg-white p-5 shadow-xl">
        <h3 className="font-newsreader text-[24px] font-semibold text-cocoa-900">
          Create date poll
        </h3>
        <p className="mt-2 text-[14px] text-cocoa-600">
          Create a single-choice poll from {optionCount} selected date options.
        </p>
        <label className="mt-4 block space-y-1 text-[13px] font-medium text-cocoa-600">
          <span>Poll title</span>
          <Input
            disabled={isPending}
            onChange={(event) => setTitle(event.target.value)}
            value={title}
          />
        </label>
        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <Button
            disabled={isPending}
            onClick={() => onOpenChange(false)}
            type="button"
            variant="ghost"
          >
            Cancel
          </Button>
          <Button
            disabled={isPending || optionCount === 0}
            onClick={() => onCreate(title)}
            type="button"
          >
            {isPending ? "Creating..." : "Create poll"}
          </Button>
        </div>
      </div>
    </div>
  );
}
