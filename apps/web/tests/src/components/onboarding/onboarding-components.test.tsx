import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it } from "vitest";
import { vi } from "vitest";
import { DemoTripPageContent } from "@/_pages/demo-trip/ui/DemoTripPageContent";
import { StartOptionCard } from "@/components/onboarding/StartOptionCard";
import { createOnboardingPreferenceSchema } from "@/components/onboarding/OnboardingPreferenceWizard";
import messages from "../../../../messages/en.json";

vi.mock("next/font/google", () => ({
  Newsreader: () => ({ variable: "font-newsreader" }),
  Instrument_Sans: () => ({ variable: "font-instrument" })
}));

describe("onboarding components", () => {
  it("renders accessible existing-flow links", () => {
    const html = renderToStaticMarkup(
      <StartOptionCard href="/trips/new?mode=route" title="Create a route" description="Plan stops." bestFor="Two cities" estimatedTime="4 minutes" />
    );
    expect(html).toContain('href="/trips/new?mode=route"');
    expect(html).toContain('aria-label="Create a route. 4 minutes"');
  });

  it("validates supported language, currency, and a reasonable walking limit", () => {
    const schema = createOnboardingPreferenceSchema((key) => key);
    const result = schema.safeParse({
      homeCity: "Bratislava", homeCountry: "Slovakia", preferredCurrency: "BTC",
      preferredLanguage: "de", travelStyles: [], budgetComfort: "balanced", pace: "balanced",
      maxWalkingKmPerDay: 100, foodPreferences: [], dietaryRestrictions: [],
      preferredTransport: [], accommodationStyle: []
    });
    expect(result.success).toBe(false);
  });

  it("labels the demo as read-only and exposes no mutation controls", () => {
    const html = renderToStaticMarkup(
      <NextIntlClientProvider locale="en" messages={messages}>
        <DemoTripPageContent />
      </NextIntlClientProvider>
    );
    expect(html).toContain("This is a read-only demo");
    expect(html).toContain("Demo trip, read-only");
    expect(html).not.toContain("Save changes");
    expect(html).not.toContain("Share publicly");
  });
});
