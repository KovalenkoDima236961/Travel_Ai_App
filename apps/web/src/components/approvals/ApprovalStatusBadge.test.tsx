import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { ApprovalStatusBadge } from "@/components/approvals/ApprovalStatusBadge";
import type { ApprovalStatus } from "@/types/approval";

const STATUSES: { status: ApprovalStatus; label: string }[] = [
  { status: "not_required", label: "Not required" },
  { status: "draft", label: "Draft" },
  { status: "pending_approval", label: "Pending approval" },
  { status: "changes_requested", label: "Changes requested" },
  { status: "approved", label: "Approved" },
  { status: "cancelled", label: "Cancelled" }
];

describe("ApprovalStatusBadge", () => {
  it("renders a distinct label for every status", () => {
    for (const { status, label } of STATUSES) {
      const html = renderToStaticMarkup(<ApprovalStatusBadge status={status} />);
      expect(html).toContain(label);
    }
  });
});
