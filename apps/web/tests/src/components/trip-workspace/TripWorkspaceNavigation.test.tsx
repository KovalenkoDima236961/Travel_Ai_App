// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import { axe } from "jest-axe";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it, vi } from "vitest";
import messages from "../../../../messages/en.json";
import { TripWorkspaceNavigation } from "@/components/trip-workspace/TripWorkspaceNavigation";

vi.mock("@/lib/api/alpha", () => ({ trackAlphaEvent: vi.fn() }));

describe("TripWorkspaceNavigation", () => {
  it("announces the active section and exposes all six sections", async () => {
    const { container } = render(
      <NextIntlClientProvider locale="en" messages={messages}>
        <TripWorkspaceNavigation
          activeSection="money"
          lifecycle="planning"
          role="viewer"
          tripId="trip-1"
        />
      </NextIntlClientProvider>
    );

    const currentLinks = screen.getAllByRole("link", { current: "page" });
    expect(currentLinks.some((link) => link.getAttribute("href") === "/trips/trip-1/money")).toBe(true);
    expect(container.querySelector('a[href="/trips/trip-1/group"]')).not.toBeNull();
    expect(await axe(container)).toHaveNoViolations();
  });
});
