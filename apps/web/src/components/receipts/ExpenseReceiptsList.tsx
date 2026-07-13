import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import type { ExpenseReceiptSummary } from "@/entities/receipt/model";
import { ReceiptConfidenceBadge } from "./ReceiptConfidenceBadge";

export function ExpenseReceiptsList({
  receipts,
  onDelete,
  onView,
  deleting
}: {
  receipts: ExpenseReceiptSummary[];
  onDelete?: (receiptId: string) => void;
  onView?: (receiptId: string) => void;
  deleting?: boolean;
}) {
  const t = useTranslations("receipts");
  if (receipts.length === 0) {
    return null;
  }
  return (
    <div className="mt-3 space-y-2">
      {receipts.map((receipt) => (
        <div
          className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600"
          key={receipt.id}
        >
          <div className="flex min-w-0 items-center gap-2">
            <span aria-hidden="true">▣</span>
            <button
              className="truncate text-left font-medium text-slate-800 underline-offset-2 hover:underline"
              onClick={() => onView?.(receipt.id)}
              type="button"
            >
              {receipt.originalFilename}
            </button>
            <ReceiptConfidenceBadge confidence={receipt.ocrConfidence} />
          </div>
          {onDelete ? (
            <Button disabled={deleting} onClick={() => onDelete(receipt.id)} size="sm" type="button" variant="ghost">
              {t("deleteReceipt")}
            </Button>
          ) : null}
        </div>
      ))}
    </div>
  );
}
