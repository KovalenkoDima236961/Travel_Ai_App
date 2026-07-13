import { useTranslations } from "next-intl";
import type { ReceiptOCRConfidence } from "@/entities/receipt/model";

export function ReceiptConfidenceBadge({ confidence }: { confidence?: ReceiptOCRConfidence | null }) {
  const t = useTranslations("receipts");
  if (!confidence) {
    return null;
  }
  const classes =
    confidence === "high"
      ? "border-emerald-200 bg-emerald-50 text-emerald-700"
      : confidence === "medium"
        ? "border-amber-200 bg-amber-50 text-amber-800"
        : "border-red-200 bg-red-50 text-red-700";
  return (
    <span className={`rounded-full border px-2 py-0.5 text-xs font-medium ${classes}`}>
      {t(`confidence.${confidence}`)}
    </span>
  );
}
