import { expect, test } from "@playwright/test"

test.describe("public-key authentication", () => {
  test.skip(!process.env.KUBEVISION_E2E_PUBLIC_KEY, "Set KUBEVISION_E2E_PUBLIC_KEY with a configured backend")

  test("registers, signs in, renames, revokes, and rejects ceremony replay", async ({ page, context }) => {
    const cdp = await context.newCDPSession(page)
    await cdp.send("WebAuthn.enable")
    await cdp.send("WebAuthn.addVirtualAuthenticator", { options: { protocol: "ctap2", transport: "internal", hasResidentKey: true, hasUserVerification: true, isUserVerified: true } })

    await page.goto("/login")
    await page.getByLabel("Username").fill(process.env.KUBEVISION_E2E_USERNAME ?? "admin")
    await page.getByLabel("Password").fill(process.env.KUBEVISION_E2E_PASSWORD ?? "admin123")
    await page.getByRole("button", { name: "Sign In", exact: true }).click()
    await page.goto("/settings/security")
    await page.getByLabel("Credential label").fill("Playwright authenticator")
    await page.getByLabel("Current password").fill(process.env.KUBEVISION_E2E_PASSWORD ?? "admin123")
    await page.getByRole("button", { name: "Register credential" }).click()
    await expect(page.getByText("Playwright authenticator", { exact: true })).toBeVisible()

    page.once("dialog", (dialog) => dialog.accept("Renamed authenticator"))
    await page.getByTitle("Rename credential").click()
    await expect(page.getByText("Renamed authenticator", { exact: true })).toBeVisible()

    await page.evaluate(() => { localStorage.clear() })
    await page.goto("/login")
    let finishBody = ""
    let ceremonyId = ""
    page.on("request", (request) => {
      if (request.url().endsWith("/api/v1/auth/public-key/login/finish")) {
        finishBody = request.postData() ?? ""
        ceremonyId = request.headers()["x-webauthn-ceremony"] ?? ""
      }
    })
    await page.getByRole("button", { name: /Sign in with a passkey/ }).click()
    await expect(page).toHaveURL(/\/overview/)
    const replay = await page.request.post("/api/v1/auth/public-key/login/finish", { data: JSON.parse(finishBody), headers: { "X-WebAuthn-Ceremony": ceremonyId } })
    expect((await replay.json()).code).toBe(40100)

    await page.goto("/settings/security")
    page.once("dialog", (dialog) => dialog.accept())
    await page.getByTitle("Revoke credential").click()
    await expect(page.getByText("Renamed authenticator", { exact: true })).not.toBeVisible()
  })
})
