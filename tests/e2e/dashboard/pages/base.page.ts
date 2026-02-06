import { type Page, type Locator, expect } from "@playwright/test"

export class BasePage {
  readonly page: Page

  // Navigation
  readonly navDashboard: Locator
  readonly navSources: Locator
  readonly navPipelines: Locator
  readonly navQuery: Locator

  constructor(page: Page) {
    this.page = page
    this.navDashboard = page.getByRole("link", { name: "Dashboard" })
    this.navSources = page.getByRole("link", { name: "Sources" })
    this.navPipelines = page.getByRole("link", { name: "Pipelines" })
    this.navQuery = page.getByRole("link", { name: "Query" })
  }

  async goto(path: string) {
    await this.page.goto(path)
  }

  async navigateTo(name: "Dashboard" | "Sources" | "Pipelines" | "Query") {
    await this.page.getByRole("link", { name }).click()
  }

  async expectHeading(text: string) {
    await expect(this.page.getByRole("heading", { name: text }).first()).toBeVisible()
  }

  async expectToast(text: string) {
    await expect(this.page.getByText(text)).toBeVisible({ timeout: 10_000 })
  }

  async expectUrl(path: string) {
    await expect(this.page).toHaveURL(new RegExp(path))
  }
}
