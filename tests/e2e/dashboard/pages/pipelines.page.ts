import { type Page, type Locator, expect } from "@playwright/test"

export class PipelinesPage {
  readonly page: Page

  readonly newPipelineButton: Locator
  readonly heading: Locator

  constructor(page: Page) {
    this.page = page
    this.newPipelineButton = page.getByRole("link", { name: "New Pipeline" }).first()
    this.heading = page.getByRole("heading", { name: "Pipelines", exact: true })
  }

  async goto() {
    await this.page.goto("/pipelines")
  }

  async expectHeading() {
    await expect(this.heading).toBeVisible()
  }

  async expectPipelineList() {
    // Wait for either pipeline cards or the empty state
    await expect(
      this.page.getByText("table mappings").first()
        .or(this.page.getByText("No pipelines"))
    ).toBeVisible({ timeout: 15_000 })
  }

  async expectPipelineCard(pipelineName: string) {
    await expect(this.page.getByText(pipelineName).first()).toBeVisible({ timeout: 10_000 })
  }

  async expectPipelineStatus(status: string) {
    // Look for the pipeline status badge
    await expect(
      this.page.getByText(status, { exact: false }).first()
    ).toBeVisible({ timeout: 10_000 })
  }

  async clickViewDetails(pipelineName: string) {
    const card = this.page.locator("div").filter({ hasText: pipelineName }).first()
    await card.getByRole("link", { name: "View Details" }).click()
  }

  async clickStart(pipelineName: string) {
    const card = this.page.locator("div").filter({ hasText: pipelineName }).first()
    await card.getByRole("button", { name: "Start" }).click()
  }

  async clickStop(pipelineName: string) {
    const card = this.page.locator("div").filter({ hasText: pipelineName }).first()
    await card.getByRole("button", { name: "Stop" }).click()
  }

  async expectEmpty() {
    await expect(this.page.getByText("No pipelines")).toBeVisible()
  }
}
