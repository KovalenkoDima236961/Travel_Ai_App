"use client";

import { ConfirmDialog } from "@/components/ui";

export function DataCleanupConfirmDialog({ open, title, description, confirmLabel, pending, onCancel, onConfirm }: { open: boolean; title: string; description: string; confirmLabel: string; pending?: boolean; onCancel: () => void; onConfirm: () => void }) {
  return <ConfirmDialog confirmLabel={confirmLabel} description={description} onCancel={onCancel} onConfirm={onConfirm} open={open} pending={pending} title={title} tone="danger" />;
}
