import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it, vi } from "vitest";
import {
  ConfirmDialog,
  DeepLinkScrollTarget,
  EmptyState,
  ErrorState,
  FormErrorSummary,
  PageLoadingState,
  StatusBadge,
  StickyMobileActionBar,
  UnsavedChangesDialog
} from "@/components/ui";
import messages from "../../../../messages/en.json";

function render(node: React.ReactNode) {
  return renderToStaticMarkup(
    <NextIntlClientProvider locale="en" messages={messages}>
      {node}
    </NextIntlClientProvider>
  );
}

describe("shared UX primitives", () => {
  it("renders an action-oriented and permission-aware empty state", () => {
    const html = render(
      <EmptyState
        description="Generate a useful checklist."
        primaryAction={{
          disabled: true,
          disabledReason: "Ask an editor to generate one.",
          label: "Generate checklist"
        }}
        title="No checklist yet"
      />
    );
    expect(html).toContain("No checklist yet");
    expect(html).toContain("Generate checklist");
    expect(html).toContain("Ask an editor");
    expect(html).toContain("disabled");
  });

  it("renders a contextual error with retry without raw details", () => {
    const html = render(
      <ErrorState
        description="Your route is still saved."
        developmentDetails="provider stack trace"
        retryAction={{ onRetry: vi.fn() }}
        title="We could not load transport options"
      />
    );
    expect(html).toContain('role="alert"');
    expect(html).toContain("Your route is still saved.");
    expect(html).toContain("Retry");
    expect(html).not.toContain("provider stack trace");
  });

  it("gives confirmation dialogs accessible names and safe default focus order", () => {
    const html = render(
      <ConfirmDialog
        confirmLabel="Delete receipt"
        description="The linked expense will not be deleted."
        onCancel={vi.fn()}
        onConfirm={vi.fn()}
        open
        title="Delete this receipt?"
        tone="danger"
      />
    );
    expect(html).toContain('role="dialog"');
    expect(html).toContain('aria-modal="true"');
    expect(html).toContain("Delete this receipt?");
    expect(html).toContain("The linked expense will not be deleted.");
  });

  it("uses explicit unsaved-change copy", () => {
    const html = render(
      <UnsavedChangesDialog onDiscard={vi.fn()} onKeepEditing={vi.fn()} open />
    );
    expect(html).toContain("Discard unsaved changes?");
    expect(html).toContain("Keep editing");
    expect(html).toContain("Discard changes");
  });

  it("links a long-form error summary to invalid fields", () => {
    const html = render(
      <FormErrorSummary
        errors={[
          { fieldId: "start-date", label: "Start date", message: "Choose a date." },
          { fieldId: "amount", label: "Amount", message: "Enter an amount." }
        ]}
      />
    );
    expect(html).toContain('href="#start-date"');
    expect(html).toContain('href="#amount"');
    expect(html).toContain("Please fix the following fields");
  });

  it("renders mobile actions, visible status text, deep-link target, and page skeleton semantics", () => {
    const mobile = render(
      <StickyMobileActionBar
        onCancel={vi.fn()}
        onPrimary={vi.fn()}
        primaryLabel="Save route"
      />
    );
    expect(mobile).toContain('role="group"');
    expect(mobile).toContain("Save route");

    const badge = render(<StatusBadge label="Needs attention" tone="warning" />);
    expect(badge).toContain("Needs attention");

    const target = render(
      <DeepLinkScrollTarget itemId="route-leg-1" targetId="route-leg-1">
        Route leg
      </DeepLinkScrollTarget>
    );
    expect(target).toContain('id="route-leg-1"');
    expect(target).toContain("Linked item opened and highlighted");

    const loading = render(<PageLoadingState cardCount={2} />);
    expect(loading).toContain('aria-busy="true"');
    expect(loading).toContain("Loading page");
  });
});
