import { test, expect } from "./fixtures"

test.describe("Pipelines Page", () => {
  test("shows the pipelines page", async ({ pipelinesPage }) => {
    await pipelinesPage.goto()
    await pipelinesPage.expectHeading()
  })

  test("displays pipeline list or empty state", async ({ pipelinesPage }) => {
    await pipelinesPage.goto()
    await pipelinesPage.expectPipelineList()
  })

  test("has a New Pipeline button", async ({ pipelinesPage }) => {
    await pipelinesPage.goto()
    await expect(pipelinesPage.newPipelineButton).toBeVisible()
  })

  test("shows pipeline status when pipelines exist", async ({ pipelinesPage }) => {
    await pipelinesPage.goto()
    await pipelinesPage.expectPipelineList()

    // If pipelines exist, check that status badges are present
    const hasPipelines = await pipelinesPage.page.getByText("table mappings").first().isVisible().catch(() => false)
    if (hasPipelines) {
      // Should have at least one status badge (running, stopped, etc.)
      await expect(
        pipelinesPage.page.getByText(/running|stopped|starting|stopping|error/i).first()
      ).toBeVisible()
    }
  })

  test("navigates to pipeline details", async ({ pipelinesPage }) => {
    await pipelinesPage.goto()
    await pipelinesPage.expectPipelineList()

    // Only test if there are pipelines
    const hasPipelines = await pipelinesPage.page.getByText("table mappings").first().isVisible().catch(() => false)
    if (hasPipelines) {
      const detailsLink = pipelinesPage.page.getByRole("link", { name: "View Details" }).first()
      await detailsLink.click()
      await expect(pipelinesPage.page).toHaveURL(/\/pipelines\//)
    }
  })
})
