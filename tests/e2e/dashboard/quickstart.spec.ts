import { test, expect } from "./fixtures"

test.describe("Quickstart Flow", () => {
  test("completes the full quickstart wizard", { timeout: 120_000 }, async ({ quickstartPage }) => {
    // Navigate to quickstart
    await quickstartPage.goto()

    // Step 1: Welcome
    await quickstartPage.expectWelcomeStep()
    await quickstartPage.clickGetStarted()

    // Step 2: Connect — fill source database credentials
    await quickstartPage.fillConnectionForm({
      name: `E2E Source ${Date.now()}`,
      host: "localhost",
      port: "5433",
      database: "source",
      username: "source",
      password: "source",
    })

    // Test connection
    await quickstartPage.testConnection()
    await quickstartPage.expectConnectionSuccess()
    await quickstartPage.clickContinue()

    // Step 3: Tables — select tables
    await quickstartPage.expectTablesStep()
    await quickstartPage.expectTableCount(1)
    // Tables should be auto-selected
    await quickstartPage.clickContinue()

    // Step 4: Verify — wait for data replication
    await quickstartPage.expectVerifyStep()
    await quickstartPage.waitForVerificationComplete()
    await quickstartPage.clickContinue()

    // Step 5: Complete
    await quickstartPage.expectCompleteStep()
  })

  test("shows health checks on welcome step", async ({ quickstartPage }) => {
    await quickstartPage.goto()
    await quickstartPage.expectWelcomeStep()

    // Should show system status section
    await expect(quickstartPage.page.getByText("System Status")).toBeVisible()

    // Should show health check items
    await expect(quickstartPage.page.getByText("API Server")).toBeVisible()
    await expect(quickstartPage.page.getByText("Buffer Database")).toBeVisible()
    await expect(quickstartPage.page.getByText("Object Storage")).toBeVisible()
    await expect(quickstartPage.page.getByText("Iceberg Catalog")).toBeVisible()
  })

  test("validates connection form before allowing test", async ({ quickstartPage }) => {
    await quickstartPage.goto()
    await quickstartPage.clickGetStarted()

    // Test Connection button should be disabled with empty form
    await expect(quickstartPage.testConnectionButton).toBeDisabled()

    // Continue button should be disabled without successful test
    await expect(quickstartPage.continueButton).toBeDisabled()
  })

  test("can navigate back from connect step", async ({ quickstartPage }) => {
    await quickstartPage.goto()
    await quickstartPage.clickGetStarted()

    // Should be on connect step
    await expect(quickstartPage.page.getByText("Connect Your Database")).toBeVisible()

    // Go back to welcome
    await quickstartPage.backButton.click()
    await quickstartPage.expectWelcomeStep()
  })
})
