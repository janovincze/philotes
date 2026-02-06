import { test, expect } from "./fixtures"

test.describe("Query Page", () => {
  test("shows the SQL editor", async ({ queryPage }) => {
    await queryPage.goto()
    await queryPage.expectEditorVisible()

    // Run Query and Clear buttons should be visible
    await expect(queryPage.runQueryButton).toBeVisible()
    await expect(queryPage.clearButton).toBeVisible()
    await expect(queryPage.templatesButton).toBeVisible()
  })

  test("can select a query template", async ({ queryPage }) => {
    await queryPage.goto()
    await queryPage.expectEditorVisible()

    // Open templates and select one (use SELECT-based template; SHOW statements
    // break when the backend appends LIMIT)
    await queryPage.selectTemplate("Sample Customers")

    // Execute the template query
    await queryPage.executeQuery()
    await queryPage.waitForResults()
  })

  test("executes a query and shows results", async ({ queryPage }) => {
    await queryPage.goto()
    await queryPage.expectEditorVisible()

    // Use "Test Query" template (SELECT 1) — always returns data
    await queryPage.selectTemplate("Test Query")
    await queryPage.executeQuery()
    await queryPage.waitForResults()
    await queryPage.expectResultsTable()
  })

  test("shows Export CSV button when results are present", async ({ queryPage }) => {
    await queryPage.goto()

    await queryPage.selectTemplate("Test Query")
    await queryPage.executeQuery()
    await queryPage.waitForResults()
    await queryPage.expectResultsTable()
    await queryPage.expectExportCsvVisible()
  })

  test("pre-fills SQL from table query parameter", async ({ page }) => {
    await page.goto("/query?table=iceberg.public.customers")

    // Editor should contain the pre-filled query
    await expect(page.getByText("SQL Editor")).toBeVisible()
    // The Monaco editor should have the pre-filled content
    await expect(page.locator(".monaco-editor")).toBeVisible()
  })
})
