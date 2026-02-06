import { type Page, type Locator, expect } from "@playwright/test"

export class QueryPage {
  readonly page: Page

  // Action buttons
  readonly runQueryButton: Locator
  readonly clearButton: Locator
  readonly templatesButton: Locator
  readonly exportCsvButton: Locator

  // Results
  readonly resultsSection: Locator

  constructor(page: Page) {
    this.page = page
    this.runQueryButton = page.getByRole("button", { name: "Run Query" })
    this.clearButton = page.getByRole("button", { name: "Clear" })
    this.templatesButton = page.getByRole("button", { name: "Templates" })
    this.exportCsvButton = page.getByRole("button", { name: "Export CSV" })
    this.resultsSection = page.getByText("Results")
  }

  async goto() {
    await this.page.goto("/query")
  }

  async expectEditorVisible() {
    await expect(this.page.getByText("SQL Editor")).toBeVisible()
  }

  async selectTemplate(templateName: string) {
    await this.templatesButton.click()
    await this.page.getByRole("menuitem", { name: new RegExp(templateName) }).click()
  }

  async executeQuery() {
    await this.runQueryButton.click()
  }

  async executeWithKeyboard() {
    // Focus the editor area and press Ctrl+Enter
    await this.page.locator(".monaco-editor").first().click()
    await this.page.keyboard.press("Control+Enter")
  }

  async waitForResults() {
    // Wait for either results table or error to appear
    await expect(
      this.page.getByText(/row/).or(this.page.getByText("Query completed"))
    ).toBeVisible({ timeout: 30_000 })
  }

  async expectResultsTable() {
    // Expect at least one row indicator
    await expect(this.page.getByText(/\d+ rows?/)).toBeVisible({ timeout: 30_000 })
  }

  async expectExportCsvVisible() {
    await expect(this.exportCsvButton).toBeVisible()
  }

  async expectErrorMessage(text?: string) {
    if (text) {
      await expect(this.page.getByText(text)).toBeVisible()
    } else {
      await expect(this.page.locator('[role="alert"]')).toBeVisible()
    }
  }

  async expectQueryRunning() {
    await expect(this.page.getByRole("button", { name: "Running..." })).toBeVisible()
  }
}
