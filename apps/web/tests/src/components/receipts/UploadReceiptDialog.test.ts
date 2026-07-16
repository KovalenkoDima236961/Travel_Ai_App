import { describe, expect, it } from "vitest";
import { validateReceiptFile } from "@/components/receipts/UploadReceiptDialog";
import { RECEIPT_MAX_FILE_SIZE_BYTES } from "@/entities/receipt/model";

const translate = (key: "unsupportedFileType" | "fileTooLarge") => key;

describe("receipt file validation", () => {
  it("rejects unsupported files before upload", () => {
    expect(
      validateReceiptFile(
        { type: "text/plain", size: 100 } as File,
        translate
      )
    ).toBe("unsupportedFileType");
  });

  it("rejects oversized files and accepts supported files", () => {
    expect(
      validateReceiptFile(
        { type: "image/jpeg", size: RECEIPT_MAX_FILE_SIZE_BYTES + 1 } as File,
        translate
      )
    ).toBe("fileTooLarge");
    expect(
      validateReceiptFile(
        { type: "application/pdf", size: 1024 } as File,
        translate
      )
    ).toBeNull();
  });
});
