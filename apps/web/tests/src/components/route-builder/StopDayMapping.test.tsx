import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it } from "vitest";
import { StopDayMapping } from "@/components/route-builder/StopDayMapping";
import messages from "../../../../messages/en.json";

describe("StopDayMapping", () => {
  it("maps itinerary days and activity counts to route stops", () => {
    const html = renderToStaticMarkup(
      <NextIntlClientProvider locale="en" messages={messages}>
        <StopDayMapping
          itinerary={{
            days: [
              {
                day: 2,
                title: "Vienna",
                primaryStopId: "vienna",
                items: [
                  { time: "09:00", type: "place", name: "Belvedere" },
                  { time: "14:00", type: "food", name: "Naschmarkt" }
                ]
              }
            ]
          }}
          route={{ stops: [{ id: "vienna", destination: "Vienna" }], legs: [] }}
        />
      </NextIntlClientProvider>
    );
    expect(html).toContain("Vienna");
    expect(html).toContain("Day 2");
    expect(html).toContain("2 itinerary items");
  });
});
