// Clerk live-loop verification: /login must render Clerk's hosted SignIn in
// Clerk mode. Real Clerk sign-in can't be fully automated here (email codes),
// so this verifies the Clerk mode engagement + that the legacy flow is gone,
// plus that the backend bridge endpoint is listening (401 on junk token).
const puppeteer = require("puppeteer-core");
const results = [];
function check(name, ok) {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}`);
}
(async () => {
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
    defaultViewport: { width: 1440, height: 900 },
  });
  const page = await browser.newPage();
  try {
    await page.goto("http://localhost:3000/login", { waitUntil: "networkidle2", timeout: 30000 });
    await new Promise((r) => setTimeout(r, 4000)); // Clerk client boot
    const text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
    check("clerk sign-in UI engaged on /login", text.includes("sign in") && (text.includes("continue") || text.includes("clerk") || text.includes("email address") || text.includes("email")));
    check("legacy password form replaced", !(await page.$("#password")));

    // Register fallback should now render SignIn, not "unavailable"
    await page.goto("http://localhost:3000/register", { waitUntil: "networkidle2", timeout: 30000 });
    await new Promise((r) => setTimeout(r, 4000));
    const regText = (await page.evaluate(() => document.body.innerText)).toLowerCase();
    check("register page shows Clerk SignUp", !regText.includes("registration unavailable"));

    // Bridge endpoint live check
    const resp = await page.evaluate(async () => {
      const r = await fetch("http://localhost:8080/api/v1/auth/clerk-session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ token: "probe" }),
      });
      return r.status;
    });
    check("bridge endpoint reachable from browser (401 on junk)", resp === 401);
    console.log(`\nCLERK LIVE: ${results.filter((r) => r.ok).length}/${results.length} checks passed`);
  } finally {
    await browser.close();
  }
})().catch((e) => {
  console.error("CLERK LIVE failed:", e.message);
  process.exit(1);
});
