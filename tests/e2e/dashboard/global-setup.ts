import type { FullConfig } from "@playwright/test"

async function waitForUrl(url: string, label: string, timeoutMs = 60_000) {
  const start = Date.now()
  let lastError: string | null = null

  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url)
      if (res.ok) {
        console.log(`  ${label} is ready`)
        return
      }
      lastError = `HTTP ${res.status} ${res.statusText}`
    } catch (err) {
      lastError = err instanceof Error ? err.message : String(err)
    }
    await new Promise((r) => setTimeout(r, 2_000))
  }
  throw new Error(
    `${label} did not become ready within ${timeoutMs / 1000}s (${url})\nLast error: ${lastError}`
  )
}

async function globalSetup(_config: FullConfig) {
  console.log("Waiting for services...")

  const apiUrl = process.env.API_URL || "http://localhost:8080"
  const baseUrl = process.env.BASE_URL || "http://localhost:3001"

  await Promise.all([
    waitForUrl(`${apiUrl}/health`, "API"),
    waitForUrl(baseUrl, "Dashboard"),
  ])

  console.log("All services ready — starting tests")
}

export default globalSetup
