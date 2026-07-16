import { renderToStaticMarkup } from "react-dom/server";
import { NextIntlClientProvider } from "next-intl";
import { describe, expect, it } from "vitest";
import { PublicShareUnavailableState } from "@/_pages/public-share/ui/PublicShareUnavailableState";
import messages from "../../../../messages/en.json";

describe("PublicShareUnavailableState", () => {
  it("renders the dedicated expired-link recovery state", () => {
    const html = renderToStaticMarkup(
      <NextIntlClientProvider locale="en" messages={messages}>
        <PublicShareUnavailableState expired />
      </NextIntlClientProvider>
    );
    expect(html).toContain("This share link has expired");
    expect(html).toContain("Ask the trip owner to create a new share link.");
    expect(html).toContain("Go to home");
  });
});
