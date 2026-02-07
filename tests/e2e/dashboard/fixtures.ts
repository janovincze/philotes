import { test as base } from "@playwright/test"
import { BasePage } from "./pages/base.page"
import { QuickstartPage } from "./pages/quickstart.page"
import { QueryPage } from "./pages/query.page"
import { PipelinesPage } from "./pages/pipelines.page"

type Fixtures = {
  basePage: BasePage
  quickstartPage: QuickstartPage
  queryPage: QueryPage
  pipelinesPage: PipelinesPage
}

export const test = base.extend<Fixtures>({
  basePage: async ({ page }, use) => {
    await use(new BasePage(page))
  },
  quickstartPage: async ({ page }, use) => {
    await use(new QuickstartPage(page))
  },
  queryPage: async ({ page }, use) => {
    await use(new QueryPage(page))
  },
  pipelinesPage: async ({ page }, use) => {
    await use(new PipelinesPage(page))
  },
})

export { expect } from "@playwright/test"
