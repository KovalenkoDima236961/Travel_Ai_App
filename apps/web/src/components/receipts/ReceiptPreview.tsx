"use client";

import { useEffect, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { fetchReceiptFile, getReceiptFileUrl } from "@/lib/api/receipts";
import type { ExpenseReceipt } from "@/entities/receipt/model";

export function ReceiptPreview({ receipt }: { receipt: ExpenseReceipt }) {
  const t = useTranslations("receipts");
  const [objectUrl, setObjectUrl] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    let currentUrl: string | null = null;
    setError(null);
    setObjectUrl(null);
    fetchReceiptFile(receipt.tripId, receipt.id)
      .then((blob) => {
        if (!active) {
          return;
        }
        currentUrl = URL.createObjectURL(blob);
        setObjectUrl(currentUrl);
      })
      .catch(() => {
        if (active) {
          setError(t("previewFailed"));
        }
      });
    return () => {
      active = false;
      if (currentUrl) {
        URL.revokeObjectURL(currentUrl);
      }
    };
  }, [receipt.id, receipt.tripId, t]);

  const fileUrl = getReceiptFileUrl(receipt.tripId, receipt.id);

  if (error) {
    return (
      <div className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-600">
        <p>{error}</p>
        <Button className="mt-2" onClick={() => window.open(fileUrl, "_blank")} size="sm" type="button" variant="secondary">
          {t("openFile")}
        </Button>
      </div>
    );
  }

  if (!objectUrl) {
    return (
      <div className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-500">
        {t("loadingPreview")}
      </div>
    );
  }

  if (receipt.contentType === "application/pdf") {
    return (
      <div className="space-y-2">
        <iframe className="h-72 w-full rounded-lg border border-slate-200" src={objectUrl} title={receipt.originalFilename} />
        <Button onClick={() => window.open(objectUrl, "_blank")} size="sm" type="button" variant="secondary">
          {t("openFile")}
        </Button>
      </div>
    );
  }

  return (
    <img
      alt={receipt.originalFilename}
      className="max-h-80 w-full rounded-lg border border-slate-200 object-contain"
      src={objectUrl}
    />
  );
}
