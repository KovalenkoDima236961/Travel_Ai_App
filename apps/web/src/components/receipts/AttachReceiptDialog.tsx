"use client";

import { FormEvent, useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { Input } from "@/shared/ui/input";
import {
  RECEIPT_ALLOWED_TYPES,
  RECEIPT_MAX_FILE_SIZE_BYTES,
  type ExpenseReceipt
} from "@/entities/receipt/model";
import { useAttachReceiptToExpense } from "@/hooks/useAttachReceiptToExpense";
import { useTripReceipts } from "@/hooks/useTripReceipts";
import { useUploadReceipt } from "@/hooks/useUploadReceipt";
import { getErrorMessage } from "@/lib/utils";
import { ReceiptConfidenceBadge } from "./ReceiptConfidenceBadge";

export function AttachReceiptDialog({
  tripId,
  expenseId,
  onClose,
  onAttached
}: {
  tripId: string;
  expenseId: string;
  onClose: () => void;
  onAttached?: (receipt: ExpenseReceipt) => void;
}) {
  const t = useTranslations("receipts");
  const receiptsQuery = useTripReceipts({ tripId, params: { unlinkedOnly: true } });
  const attachMutation = useAttachReceiptToExpense(tripId);
  const uploadMutation = useUploadReceipt(tripId);
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState<string | null>(null);
  const receipts = receiptsQuery.data?.receipts ?? [];

  async function attach(receiptId: string) {
    setError(null);
    try {
      const receipt = await attachMutation.mutateAsync({ expenseId, receiptId });
      onAttached?.(receipt);
      onClose();
    } catch (err) {
      setError(getErrorMessage(err, t("attachFailed")));
    }
  }

  async function upload(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    if (!file) {
      setError(t("selectFile"));
      return;
    }
    if (!RECEIPT_ALLOWED_TYPES.includes(file.type as (typeof RECEIPT_ALLOWED_TYPES)[number])) {
      setError(t("unsupportedFileType"));
      return;
    }
    if (file.size > RECEIPT_MAX_FILE_SIZE_BYTES) {
      setError(t("fileTooLarge"));
      return;
    }
    try {
      const receipt = await uploadMutation.mutateAsync({ file, expenseId, runOcr: true });
      onAttached?.(receipt);
      onClose();
    } catch (err) {
      setError(getErrorMessage(err, t("uploadFailed")));
    }
  }

  return (
    <Card>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h3 className="text-base font-semibold text-slate-950">{t("attachReceipt")}</h3>
          <p className="mt-1 text-sm leading-6 text-slate-600">{t("attachDescription")}</p>
        </div>
        <Button onClick={onClose} size="sm" type="button" variant="ghost">
          {t("cancel")}
        </Button>
      </div>

      {error ? (
        <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-800">
          {error}
        </div>
      ) : null}

      <form className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center" onSubmit={upload}>
        <Input
          accept={RECEIPT_ALLOWED_TYPES.join(",")}
          onChange={(event) => setFile(event.target.files?.[0] ?? null)}
          type="file"
        />
        <Button disabled={uploadMutation.isPending} type="submit">
          {uploadMutation.isPending ? t("uploading") : t("uploadToExpense")}
        </Button>
      </form>

      <div className="mt-5 space-y-2">
        <p className="text-sm font-medium text-slate-950">{t("unlinkedReceipts")}</p>
        {receiptsQuery.isLoading ? (
          <p className="text-sm text-slate-500">{t("loadingReceipts")}</p>
        ) : receipts.length === 0 ? (
          <p className="text-sm text-slate-500">{t("noUnlinkedReceipts")}</p>
        ) : (
          receipts.map((receipt) => (
            <div className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-slate-200 px-3 py-2 text-sm" key={receipt.id}>
              <div className="min-w-0">
                <p className="truncate font-medium text-slate-900">{receipt.originalFilename}</p>
                <div className="mt-1 flex items-center gap-2">
                  <span className="text-xs text-slate-500">{receipt.status}</span>
                  <ReceiptConfidenceBadge confidence={receipt.ocrResult?.confidence} />
                </div>
              </div>
              <Button disabled={attachMutation.isPending} onClick={() => attach(receipt.id)} size="sm" type="button" variant="secondary">
                {t("attach")}
              </Button>
            </div>
          ))
        )}
      </div>
    </Card>
  );
}
