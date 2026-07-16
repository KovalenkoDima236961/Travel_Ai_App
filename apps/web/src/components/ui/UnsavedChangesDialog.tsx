"use client";

import { useTranslations } from "next-intl";
import { ConfirmDialog } from "./ConfirmDialog";

type UnsavedChangesDialogProps = {
  open: boolean;
  pending?: boolean;
  onKeepEditing: () => void;
  onDiscard: () => void;
};

export function UnsavedChangesDialog({
  open,
  pending = false,
  onKeepEditing,
  onDiscard
}: UnsavedChangesDialogProps) {
  const t = useTranslations("confirmations");
  return (
    <ConfirmDialog
      cancelLabel={t("unsaved.keepEditing")}
      confirmLabel={t("unsaved.discard")}
      description={t("unsaved.description")}
      onCancel={onKeepEditing}
      onConfirm={onDiscard}
      open={open}
      pending={pending}
      title={t("unsaved.title")}
      tone="danger"
    />
  );
}
