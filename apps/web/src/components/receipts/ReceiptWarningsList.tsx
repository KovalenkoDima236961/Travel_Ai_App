import { useTranslations } from "next-intl";

export function ReceiptWarningsList({ warnings }: { warnings?: string[] }) {
  const t = useTranslations("receipts");
  if (!warnings?.length) {
    return null;
  }
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
      <p className="font-medium">{t("warnings")}</p>
      <ul className="mt-2 list-disc space-y-1 pl-5">
        {warnings.map((warning, index) => (
          <li key={`${warning}-${index}`}>{warning}</li>
        ))}
      </ul>
    </div>
  );
}
