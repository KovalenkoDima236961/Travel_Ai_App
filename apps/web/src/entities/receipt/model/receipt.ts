import type { ExpenseCategory, ExpenseSplitType, MoneyAmount } from "@/entities/expense/model";

export type ReceiptStatus =
  | "uploaded"
  | "processing"
  | "extracted"
  | "extraction_failed"
  | "attached"
  | "deleted";

export type ReceiptOCRProvider = "mock" | "local" | "manual";

export type ReceiptOCRConfidence = "low" | "medium" | "high";

export type ReceiptOCRResult = {
  merchant?: string | null;
  expenseDate?: string | null;
  amount?: MoneyAmount | null;
  taxAmount?: MoneyAmount | null;
  category?: ExpenseCategory | null;
  suggestedTitle?: string | null;
  confidence: ReceiptOCRConfidence;
  fieldConfidence: Partial<Record<"merchant" | "date" | "amount" | "currency" | "category", ReceiptOCRConfidence>>;
  warnings: string[];
  rawText?: string | null;
};

export type ExpenseReceipt = {
  id: string;
  tripId: string;
  expenseId?: string | null;
  status: ReceiptStatus;
  originalFilename: string;
  contentType: string;
  sizeBytes: number;
  previewUrl: string;
  ocrResult?: ReceiptOCRResult | null;
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type ExpenseReceiptSummary = {
  id: string;
  originalFilename: string;
  contentType: string;
  status: ReceiptStatus;
  ocrConfidence?: ReceiptOCRConfidence | null;
  createdAt: string;
};

export type TripReceiptsResponse = {
  receipts: ExpenseReceipt[];
  nextOffset?: number | null;
};

export type ReceiptUploadInput = {
  file: File;
  expenseId?: string | null;
  runOcr?: boolean;
};

export type ListReceiptsParams = {
  expenseId?: string | null;
  status?: ReceiptStatus | null;
  unlinkedOnly?: boolean;
  limit?: number;
  offset?: number;
};

export type ExtractReceiptInput = {
  provider?: ReceiptOCRProvider;
};

export type CreateExpenseFromReceiptInput = {
  title: string;
  description?: string | null;
  amount: MoneyAmount;
  category: ExpenseCategory;
  expenseDate: string;
  paidByUserId: string;
  splitType: ExpenseSplitType;
  participantUserIds?: string[];
  notes?: string | null;
  metadata?: Record<string, unknown>;
};

export type OCRExpenseDraft = {
  title: string;
  amount: MoneyAmount;
  category: ExpenseCategory;
  expenseDate: string;
};

export const RECEIPT_ALLOWED_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp",
  "application/pdf"
] as const;

export const RECEIPT_MAX_FILE_SIZE_BYTES = 10 * 1024 * 1024;
