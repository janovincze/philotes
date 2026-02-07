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
    // Radix dropdown items — click by visible text
    await this.page.getByText(templateName, { exact: true }).click()
  }

  async typeQuery(sql: string) {
    // Use the Clear button to reset the editor
    await this.clearButton.click()
    // Set the Monaco editor model value directly — keyboard.type/insertText
    // doesn't reliably update Monaco's internal model
    await this.page.evaluate((text) => {
      // Access Monaco's global editor API to find the model and set its value
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const monaco = (window as any).monaco
      const models = monaco?.editor?.getModels?.()
      if (models?.[0]) {
        models[0].setValue(text)
      }
    }, sql)
    // Wait for React state to sync with Monaco's onChange
    await this.page.waitForTimeout(200)
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
    // Wait for either results table or error alert to appear
    // Note: Next.js adds a hidden [role="alert"] route announcer, so use visible destructive alert
    await expect(
      this.page.getByTestId("results-table")
        .or(this.page.locator('[data-slot="alert"]'))
    ).toBeVisible({ timeout: 30_000 })
  }

  async expectResultsTable() {
    // Expect the results table with row count
    await expect(this.page.getByTestId("row-count")).toBeVisible({ timeout: 30_000 })
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
