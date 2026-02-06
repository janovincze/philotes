import { type Page, type Locator, expect } from "@playwright/test"

export class QuickstartPage {
  readonly page: Page

  // Welcome step
  readonly getStartedButton: Locator

  // Connect step
  readonly sourceNameInput: Locator
  readonly hostInput: Locator
  readonly portInput: Locator
  readonly databaseNameInput: Locator
  readonly usernameInput: Locator
  readonly passwordInput: Locator
  readonly testConnectionButton: Locator
  readonly continueButton: Locator
  readonly backButton: Locator

  // Tables step
  readonly selectAllButton: Locator
  readonly deselectAllButton: Locator

  constructor(page: Page) {
    this.page = page

    // Welcome
    this.getStartedButton = page.getByRole("button", { name: "Get Started" })

    // Connect step — form fields
    this.sourceNameInput = page.getByLabel("Source Name")
    this.hostInput = page.getByLabel("Host")
    this.portInput = page.getByLabel("Port")
    this.databaseNameInput = page.getByLabel("Database Name")
    this.usernameInput = page.getByLabel("Username")
    this.passwordInput = page.getByLabel("Password")
    this.testConnectionButton = page.getByRole("button", { name: "Test Connection" })
    this.continueButton = page.getByRole("button", { name: "Continue" })
    this.backButton = page.getByRole("button", { name: "Back" })

    // Tables step
    this.selectAllButton = page.getByRole("button", { name: "Select All" })
    this.deselectAllButton = page.getByRole("button", { name: "Deselect All" })
  }

  async goto() {
    await this.page.goto("/quickstart")
  }

  async expectWelcomeStep() {
    await expect(this.page.getByText("Welcome to Philotes")).toBeVisible()
  }

  async clickGetStarted() {
    await this.getStartedButton.click()
  }

  async fillConnectionForm(data: {
    name: string
    host: string
    port: string
    database: string
    username: string
    password: string
  }) {
    await this.sourceNameInput.fill(data.name)
    await this.hostInput.fill(data.host)
    await this.portInput.fill(data.port)
    await this.databaseNameInput.fill(data.database)
    await this.usernameInput.fill(data.username)
    await this.passwordInput.fill(data.password)
  }

  async testConnection() {
    await this.testConnectionButton.click()
    // Wait for the connection result to appear
    await expect(
      this.page.getByText("Connection successful").or(this.page.getByText("Connection failed"))
    ).toBeVisible({ timeout: 15_000 })
  }

  async expectConnectionSuccess() {
    await expect(this.page.getByText("Connection successful")).toBeVisible()
  }

  async clickContinue() {
    await this.continueButton.click()
  }

  async expectTablesStep() {
    await expect(this.page.getByText("Select Tables to Replicate")).toBeVisible()
  }

  async expectTableCount(min: number) {
    // Wait for tables to load
    await expect(this.page.getByText(/Found \d+ tables/)).toBeVisible({ timeout: 15_000 })
    const text = await this.page.getByText(/Found \d+ tables/).textContent()
    const count = parseInt(text?.match(/Found (\d+)/)?.[1] ?? "0", 10)
    expect(count).toBeGreaterThanOrEqual(min)
  }

  async expectVerifyStep() {
    await expect(this.page.getByText("Verify Data Replication")).toBeVisible()
  }

  async waitForVerificationComplete() {
    // Wait for all sub-steps to complete (max 90s for data to arrive)
    await expect(
      this.page.getByText("Pipeline created")
    ).toBeVisible({ timeout: 30_000 })
    await expect(
      this.page.getByText("Pipeline running")
    ).toBeVisible({ timeout: 30_000 })
    await expect(
      this.page.getByText(/rows replicated|Data verification complete/)
    ).toBeVisible({ timeout: 90_000 })
  }

  async expectCompleteStep() {
    await expect(this.page.getByText("You're All Set!")).toBeVisible({ timeout: 10_000 })
  }

  async clickQueryData() {
    await this.page.getByRole("button", { name: "Query Your Data" }).click()
  }

  async clickViewPipeline() {
    await this.page.getByRole("button", { name: "View Pipeline Details" }).click()
  }
}
