// Expert-pass browser verification: professional landing page + de-demo + auth still green.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");
const BASE = "http://localhost:3000";
const STAMP = Date.now().toString().slice(-6);

const results = [];
function check(name, ok) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}`);
}

(async () => {
  if (!(await qa.requireClassicMode("expert-pass-check"))) process.exit(0);
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
    defaultViewport: { width: 1440, height: 900 },
  });
  const page = await browser.newPage();

  try {
    // 1. Landing page renders with school identity
    await page.goto(BASE, { waitUntil: "networkidle2", timeout: 30000 });
    let text = await page.evaluate(() => document.body.innerText);
    const lower = text.toLowerCase();
    check("landing renders hero with school name", lower.includes("schoolhub academy") && lower.includes("where bright futures begin"));
    check("landing has nav + apply CTA", lower.includes("about") && lower.includes("admissions") && lower.includes("apply now"));
    check("landing has stats band", lower.includes("students enrolled"));
    check("landing has academics sections", lower.includes("lower school") && lower.includes("senior school"));
    check("landing has news + footer contact", lower.includes("news") && lower.includes("hello@schoolhub.test") && lower.includes("©"));
    check("hero image loaded", await page.evaluate(() => {
      const imgs = Array.from(document.images);
      return imgs.length > 0 && imgs.some((i) => i.complete && i.naturalWidth > 100);
    }));

    // 2. No demo/milestone artifacts
    check("no milestone console artifacts", !lower.includes("(m14)") && !lower.includes("(m15)") && !lower.includes("milestones m"));
    await page.goto(BASE + "/login", { waitUntil: "networkidle2", timeout: 30000 });
    text = await page.evaluate(() => document.body.innerText);
    check("login page clean of demo note", !text.includes("Demo accounts"));

    // 3. Legacy session auth still fully works (no clerk keys set)
    await page.goto(BASE + "/login", { waitUntil: "networkidle2", timeout: 30000 });
    const hasLegacyForm = await page.evaluate(() => !!document.querySelector("#email") && !!document.querySelector("#password"));
    check("legacy login form shown when clerk not configured", hasLegacyForm);
    await page.type("#email", qa.loadCreds().masterEmail);
    await page.type("#password", qa.loadCreds().masterPassword);
    await page.click("button[type=submit]");
    await page.waitForFunction(() => window.location.pathname.includes("dashboard"), { timeout: 25000 });
    check("admin login redirects to dashboard", (await page.url()).includes("/dashboard/admin"));
    await page.waitForFunction(() => document.body.innerText.length > 100, { timeout: 25000 });
    text = await page.evaluate(() => document.body.innerText);
    check("dashboard still functional under new branding", text.toUpperCase().includes("ADMIN"));

    // 4. Register page shows fallback when clerk not configured
    await page.goto(BASE + "/register", { waitUntil: "networkidle2", timeout: 30000 });
    text = await page.evaluate(() => document.body.innerText);
    check("register unavailable without clerk keys", text.toLowerCase().includes("registration unavailable"));

    console.log(`\nEXPERT-PASS: ${results.filter((r) => r.ok).length}/${results.length} checks passed${STAMP}`);
  } finally {
    await browser.close();
  }
})().catch((err) => {
  console.error("EXPERT-PASS failed:", err.message);
  results.forEach((r) => console.log(`${r.ok ? "PASS" : "FAIL"} — ${r.name}`));
  process.exit(1);
});
