import { chromium } from "playwright";

const BASE = "http://localhost:5174";
const OUT = "docs/assets";
const CREDS = { username: "admin", password: "admin123" };

async function main() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    colorScheme: "light",
  });
  const page = await context.newPage();

  // --- Login ---
  console.log("Logging in...");
  await page.goto(`${BASE}/login`);
  await page.fill("#username", CREDS.username);
  await page.fill("#password", CREDS.password);
  await page.click('button[type="submit"]');
  await page.waitForURL("**/overview", { timeout: 10000 });
  await page.waitForTimeout(2000); // let data load

  // 1. Terminal - navigate to pods, open a terminal
  console.log("Taking screenshot: terminal...");
  await page.goto(`${BASE}/pods`);
  await page.waitForTimeout(2000);
  // Try to click the first pod's terminal action
  try {
    const firstRow = page.locator("table tbody tr").first();
    await firstRow.waitFor({ timeout: 5000 });
    // Look for terminal/exec button in actions
    const terminalBtn = firstRow.locator('button:has([class*="terminal"]), button:has([class*="Terminal"]), [title*="Terminal"], [title*="terminal"], [aria-label*="Terminal"], [aria-label*="terminal"]').first();
    if (await terminalBtn.isVisible({ timeout: 3000 })) {
      await terminalBtn.click();
      await page.waitForTimeout(3000);
    }
  } catch {
    console.log("  Could not open terminal, taking pods page screenshot instead");
  }
  await page.screenshot({ path: `${OUT}/screenshot-terminal.png` });

  // 2. Topology
  console.log("Taking screenshot: topology...");
  await page.goto(`${BASE}/topology`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${OUT}/screenshot-topology.png` });

  // 3. Dry-Run Diff (compare page)
  console.log("Taking screenshot: diff...");
  await page.goto(`${BASE}/compare`);
  await page.waitForTimeout(2000);
  await page.screenshot({ path: `${OUT}/screenshot-diff.png` });

  // 4. Global Search (Cmd+K)
  console.log("Taking screenshot: search...");
  await page.goto(`${BASE}/overview`);
  await page.waitForTimeout(1500);
  await page.keyboard.press("Meta+k");
  await page.waitForTimeout(1000);
  await page.screenshot({ path: `${OUT}/screenshot-search.png` });
  await page.keyboard.press("Escape");

  // 5. Terminal audit/sessions
  console.log("Taking screenshot: audit...");
  await page.goto(`${BASE}/admin/terminal-sessions`);
  await page.waitForTimeout(2000);
  await page.screenshot({ path: `${OUT}/screenshot-audit.png` });

  // 6. Dark mode
  console.log("Taking screenshot: dark mode...");
  await page.goto(`${BASE}/overview`);
  await page.waitForTimeout(1500);
  // Toggle dark mode via the theme button
  try {
    const themeBtn = page.locator('button:has([class*="moon"]), button:has([class*="Moon"]), button:has([class*="sun"]), button:has([class*="Sun"]), [aria-label*="theme"], [aria-label*="Theme"], [title*="theme"], [title*="Theme"]').first();
    if (await themeBtn.isVisible({ timeout: 2000 })) {
      await themeBtn.click();
      await page.waitForTimeout(500);
      // If it opens a dropdown, select dark
      const darkOption = page.locator('text=Dark, text=dark, [data-value="dark"]').first();
      if (await darkOption.isVisible({ timeout: 1000 })) {
        await darkOption.click();
      }
      await page.waitForTimeout(1000);
    }
  } catch {
    // Fallback: inject dark mode
    await page.evaluate(() => {
      document.documentElement.classList.add("dark");
    });
    await page.waitForTimeout(1000);
  }
  await page.screenshot({ path: `${OUT}/screenshot-dark.png` });

  console.log("All screenshots saved to", OUT);
  await browser.close();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
