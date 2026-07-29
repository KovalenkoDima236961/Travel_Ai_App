import { expect, test } from "../fixtures/test";

const viewports = [
  { name: "mobile-375", width: 375, height: 812 },
  { name: "mobile-390", width: 390, height: 844 },
  { name: "tablet", width: 768, height: 1024 }
];

for (const viewport of viewports) {
  test.describe(`alpha-mobile-${viewport.name}`, () => {
    test.use({ viewport });

    test("critical pages do not horizontally overflow", async ({ page }) => {
      for (const path of ["/trips", "/trips/new", "/settings"]) {
        await page.goto(path);
        await expect(page.locator("body")).toBeVisible();
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
        expect(overflow).toBeLessThanOrEqual(2);
      }
    });
  });
}
