import { test, expect } from "@playwright/test";

// Key user journeys in DEMO mode (deterministic, no backend). Covers nav,
// overview, the release evidence drill-down, and the honest live-disconnected
// failure state (DASHBOARD_AND_NETLIFY_WORKFLOW.md §7 D2).

test("overview renders with the DEMO banner and release decisions", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByTestId("demo-banner")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Overview" })).toBeVisible();
  await expect(page.getByText("Release decisions")).toBeVisible();
  // At least one source badge present.
  await expect(page.getByTestId("source-badge").first()).toBeVisible();
});

test("primary navigation reaches all sections", async ({ page }) => {
  await page.goto("/");
  for (const label of ["Agent Twin Lab", "Simulation Center", "Trace Explorer", "Release Assurance", "Settings"]) {
    await page.getByRole("link", { name: label }).click();
    await expect(page.getByRole("heading").first()).toBeVisible();
  }
});

test("release evidence drill-down shows the BLOCK decision and its rule", async ({ page }) => {
  await page.goto("/release-assurance");
  // The 1.3.0 candidate row is BLOCK; open its evidence.
  await page.getByRole("row", { name: /1\.3\.0/ }).getByRole("link", { name: /Evidence/ }).click();
  await expect(page.getByText("Combined gate decision")).toBeVisible();
  await expect(page.getByText("BLOCK", { exact: true })).toBeVisible();
  await expect(page.getByText("AGENT_CRITICAL_SCENARIO")).toBeVisible();
  // The two real critical scenarios are listed.
  await expect(page.getByText("refund-tool-success-lie")).toBeVisible();
});

test("switching to Live mode (no backend) shows an honest not-connected state", async ({ page }) => {
  await page.goto("/");
  await page.getByRole("button", { name: "Live" }).click();
  await expect(page.getByTestId("state-disconnected").first()).toBeVisible();
  await expect(page.getByTestId("demo-banner")).toHaveCount(0);
});
