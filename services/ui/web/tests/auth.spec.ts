import { test, expect } from "@playwright/test";

test.describe.skip("auth flow", () => {
  test("placeholder", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/goose’d UI/i);
  });
});
