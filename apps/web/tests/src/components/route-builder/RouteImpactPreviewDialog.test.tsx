import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it, vi } from "vitest";
import { RouteImpactPreviewDialog } from "@/components/route-builder/RouteImpactPreviewDialog";
import messages from "../../../../messages/en.json";

describe("RouteImpactPreviewDialog", () => {
  it("shows destructive impacts and confirm/cancel controls", () => {
    const html = renderToStaticMarkup(
      <NextIntlClientProvider locale="en" messages={messages}>
        <RouteImpactPreviewDialog
          impact={{
            affectedLegIds: ["leg_1"],
            removedTransportOptionCount: 2,
            staleTransportOptionCount: 1,
            itineraryImpact: true,
            budgetImpact: true,
            reminderImpact: true,
            approvalMayReset: true,
            stopOrderChanged: true,
            legCountChanged: false
          }}
          onCancel={vi.fn()}
          onConfirm={vi.fn()}
          open
        />
      </NextIntlClientProvider>
    );
    expect(html).toContain("Route impact preview");
    expect(html).toContain("remove 2 selected transport options");
    expect(html).toContain("Keep editing");
    expect(html).toContain("Save route changes");
  });
});
