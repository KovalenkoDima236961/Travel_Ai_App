import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it, vi } from "vitest";
import { SortableRouteStopList } from "@/components/route-builder/SortableRouteStopList";
import messages from "../../../../messages/en.json";

describe("SortableRouteStopList", () => {
  it("provides drag ordering and keyboard fallback buttons", () => {
    const html = renderToStaticMarkup(
      <NextIntlClientProvider locale="en" messages={messages}>
        <SortableRouteStopList
          onReorder={vi.fn()}
          stops={[
            { id: "vienna", destination: "Vienna" },
            { id: "salzburg", destination: "Salzburg" }
          ]}
        />
      </NextIntlClientProvider>
    );
    expect(html).toContain('draggable="true"');
    expect(html).toContain('aria-label="Move Vienna down"');
    expect(html).toContain('aria-label="Move Salzburg up"');
    expect(html).toContain('aria-live="polite"');
  });
});
