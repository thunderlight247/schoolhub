// Live deployment browser check for the one-click demo sign-in (DEMO_MODE).
// Verifies in a real Chrome (not curl) that:
//   1. /login on the Cloudflare frontend shows the "Explore the school"
//      persona picker (the /auth/demo-mode probe returned true)
//   2. clicking "Sign in as Teacher" opens a real cross-origin session
//      cookie and lands on /portal/teacher
//   3. the teacher workspace renders (auth middleware accepted the cookie)
//   4. the parent persona lands on /portal/parent
//
// Run: node frontend/scripts/live-demo-browser-check.js
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const CHROME = "/usr/bin/chromium";

const results = [];
function report(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

async function personaSignIn(page, expectedHome, buttonLabel) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle2", timeout: 90000 });
  await page.waitForFunction(
    () => document.body && document.body.innerText.includes("Explore the school"),
    { timeout: 60000 },
  );
  report(`persona picker visible (probe true)`, true);

  // Click the persona button by its visible label.
  await page.evaluate((label) => {
    const btns = Array.from(document.querySelectorAll("button"));
    const b = btns.find((el) => el.innerText.includes(label));
    if (!b) throw new Error(`button ${label} not found`);
    b.click();
  }, buttonLabel);

  await page.waitForFunction(
    (want) => window.location.pathname.startsWith(want),
    { timeout: 60000, polling: 500 },
    expectedHome,
  );
  report(`landed on ${expectedHome}`, true, `url=${page.url()}`);
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  try {
    const page = await browser.newPage();
    page.setDefaultNavigationTimeout(90000);

    // Teacher persona
    await personaSignIn(page, "/portal/teacher", "Sign in as Teacher");
    // Give the portal's data fetches time to resolve, then confirm the
  // signed-in persona's real data is on screen.
  await new Promise((r) => setTimeout(r, 8000));
  const text = await page.evaluate(() => document.body.innerText);
  report(
    `teacher workspace shows Ada's data`,
    text.includes("Ada"),
    `chars=${text.length}`,
  );
  console.log(`  portal text head: ${text.slice(0, 160).replace(/\n/g, " | ")}`);

    // Parent persona (fresh, cookie-free context)
    const ctx = await browser.createBrowserContext();
    const p2 = await ctx.newPage();
    p2.setDefaultNavigationTimeout(90000);
    await personaSignIn.call(null, p2, "/portal/parent", "Sign in as Parent");
    await new Promise((r) => setTimeout(r, 8000));
    const text2 = await p2.evaluate(() => document.body.innerText);
    report(
      "parent workspace shows Abena's children",
      text2.includes("Abena"),
      `chars=${text2.length}`,
    );
    console.log(`  portal text head: ${text2.slice(0, 160).replace(/\n/g, " | ")}`);
  } catch (err) {
    report("flow completed without errors", false, String(err).slice(0, 200));
  } finally {
    await browser.close();
  }

  const failed = results.filter((r) => !r.ok);
  console.log(`\n${results.length - failed.length}/${results.length} checks passed`);
  process.exit(failed.length ? 1 : 0);
})();
