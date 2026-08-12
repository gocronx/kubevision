import { expect, test } from "@playwright/test"

test.describe("public authentication surface", () => {
  test("shows the login form", async ({ page }) => {
    await page.goto("/login")

    await expect(page.getByRole("heading", { name: "KubeVision" })).toBeVisible()
    await expect(page.getByLabel("Username")).toBeVisible()
    await expect(page.getByLabel("Password")).toBeVisible()
    await expect(page.getByRole("button", { name: "Sign In" })).toBeVisible()
  })

  test("reports an authentication error from the API", async ({ page }) => {
    await page.route("**/api/v1/auth/login", async (route) => {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({ code: 40001, message: "Invalid credentials", data: null }),
      })
    })
    await page.goto("/login")
    await page.getByLabel("Username").fill("smoke-user")
    await page.getByLabel("Password").fill("incorrect-password")
    await page.getByRole("button", { name: "Sign In" }).click()

    await expect(page.getByText("Invalid credentials")).toBeVisible()
  })

  test("redirects an anonymous visitor away from a private route", async ({ page }) => {
    await page.goto("/admin/users")

    await expect(page).toHaveURL(/\/login$/)
    await expect(page.getByRole("button", { name: "Sign In" })).toBeVisible()
  })
})

test.describe("cluster-backed smoke scenarios", () => {
  test.skip(!process.env.KUBEVISION_E2E_CLUSTER, "Set KUBEVISION_E2E_CLUSTER to enable cluster scenarios")

  test("cluster scenario placeholder", async ({ page }) => {
    await page.goto("/overview")
  })
})
