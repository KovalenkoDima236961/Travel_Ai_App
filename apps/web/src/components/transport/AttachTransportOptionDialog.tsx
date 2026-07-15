"use client";

import type { TransportOption } from "@/types/transport";
import { TransportOptionCard } from "./TransportOptionCard";

type Props = {
  option: TransportOption | null;
  pending?: boolean;
  error?: string | null;
  onClose: () => void;
  onConfirm: () => void;
};

export function AttachTransportOptionDialog({ option, pending = false, error, onClose, onConfirm }: Props) {
  if (!option) {
    return null;
  }
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/40 p-4" role="dialog" aria-modal="true">
      <div className="w-full max-w-xl rounded-lg bg-white p-5 shadow-xl">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-[13px] font-semibold uppercase tracking-[0.08em] text-[#A08D78]">
              Select transport
            </p>
            <h3 className="mt-1 text-[18px] font-semibold text-cocoa-900">
              Use this option for the route leg
            </h3>
          </div>
          <button
            className="rounded-md px-2 py-1 text-[13px] font-semibold text-cocoa-500 transition hover:bg-sand-100"
            disabled={pending}
            onClick={onClose}
            type="button"
          >
            Close
          </button>
        </div>
        <div className="mt-4">
          <TransportOptionCard disabled selecting={pending} option={option} onSelect={onConfirm} />
        </div>
        {error ? (
          <p className="mt-3 rounded-md bg-red-50 px-3 py-2 text-[13px] font-medium text-red-700">
            {error}
          </p>
        ) : null}
        <div className="mt-4 flex justify-end gap-2">
          <button
            className="rounded-md border border-sand-300 bg-white px-3 py-2 text-[13px] font-semibold text-cocoa-600 transition hover:bg-sand-50"
            disabled={pending}
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
          <button
            className="rounded-md bg-cocoa-900 px-3 py-2 text-[13px] font-semibold text-white transition hover:bg-cocoa-700 disabled:opacity-60"
            disabled={pending}
            onClick={onConfirm}
            type="button"
          >
            {pending ? "Selecting" : "Select option"}
          </button>
        </div>
      </div>
    </div>
  );
}
