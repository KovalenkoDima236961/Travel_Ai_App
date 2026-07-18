# Receipts and expenses

Trip Service owns trip expenses, participants/cost splitting, settlements,
receipt metadata/files, and OCR results. Expense and receipt access always
requires trip permission; public share DTOs and public exports exclude private
financial detail.

## Flow

1. A permitted user creates/updates an expense or uploads a supported receipt
   through the trip receipt routes.
2. The service validates request/file type and size, stores it through the
   configured receipt storage provider, and creates receipt metadata.
3. OCR extraction is an explicit service workflow. Users review extracted data
   before an expense is created or attached; OCR is not authoritative.
4. Lists/summaries/settlements and CSV exports are permission-scoped. Receipt
   and expense lists use limit/offset-style pagination and `nextOffset`.

## Storage and safety

Local Compose defaults use a named receipt volume; `RECEIPT_STORAGE_*` and
`DATA_EXPORT_*` control storage and generated package retention. Do not expose
storage paths, raw OCR, files, private notes, or URLs in public data. Validate
before persistence, authorize file download separately, set bounded size/type
limits, redact logs, and keep export links authenticated/short-lived.

## Related docs

- [Data export portability](../data-export-portability.md)
- [Trips](trips.md)
- [Troubleshooting](../development/troubleshooting.md)
