// Reproduce the M34 cross-role bug: admin signs out, then logs in as a
// parent/teacher; report any NetworkError / Next dev-overlay / fetch failures.
// Run in classic mode (Clerk key commented out) with dev server on :3000.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");
const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

(async () => {
  if (!(await qa.requireClassicMode("crossrole-repro-check"))) process.exit(0);
  const creds = qa.loadCreds();
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
    defaultViewport: { width: 1440, height: 900 },
  });
  try {
    const page = await browser.newPage();
    const consoleErrors = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") consoleErrors.push(msg.text().slice(0, 200));
    });
    page.on("pageerror", (err) =>
      consoleErrors.push("PAGEERROR: " + String(err).slice(0, 200)),
    );

    // 1. Admin login
    await page.goto("http://localhost:3000/login", { waitUntil: "networkidle2", timeout: 60000 });
    await new Promise((r) => setTimeout(r, 1500));
    await page.type("#email", creds.masterEmail);
    await page.type("#password", creds.masterPassword);
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle2", timeout: 45000 }).catch(() => {}),
      page.click("button[type=submit]"),
    ]);
    await new Promise((r) => setTimeout(r, 3000));
    check("admin login reached dashboard", page.url().includes("dashboard"), page.url());

    // 2. Sign out through the shell
    await page.evaluate(() => {
      const btn = [...document.querySelectorAll("button, a")]
        .find((b) => /sign out|log ?out/i.test(b.innerText));
      if (btn) btn.click();
      return btn ? btn.tagName : null;
    });
    await new Promise((r) => setTimeout(r, 3500));
    check("sign out returned to login", page.url().includes("/login"), page.url());
    consoleErrors.length = 0; // only the NEXT leg matters

    // 3. Log in as teacher WITHOUT a fresh page load — the SPA login form is
    // the same DOM; this is the sequence the bug was reported against.
    await new Promise((r) => setTimeout(r, 1500));
    await page.type("#email", creds.teacherEmail);
    await page.type("#password", creds.teacherPassword);
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle2", timeout: 45000 }).catch(() => {}),
      page.click("button[type=submit]"),
    ]);
    await new Promise((r) => setTimeout(r, 4000));
    const text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
    const overlayText = await page.evaluate(() => {
      const overlay = document.querySelector("nextjs-portal");
      return overlay ? shadowText(overlay) : "";
      function shadowText(el) {
        if (el.shadowRoot) return el.shadowRoot.textContent || "";
        return "";
      }
    }).catch(() => "");
    check("teacher login lands on portal", page.url().includes("portal"), page.url());
    check(
      "no NetworkError/overlay after cross-role login",
      !text.includes("networkerror") &&
        !text.includes("application error") &&
        !consoleErrors.some((m) => /networkerror|failed to fetch/i.test(m)) &&
        !overlayText.toLowerCase().includes("networkerror"),
      `overlay=${(overlayText || "").slice(0, 80)} console=${consoleErrors.length}`,
    );
    if (consoleErrors.length) console.log("console errors:", consoleErrors.slice(0, 6));
  } finally {
    await browser.close().catch(() => {});
  }
  const ok = results.filter((r) => r.ok).length;
  console.log(`\nM34 CROSS-ROLE: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("cross-role repro failed:", e.message);
  process.exit(1);
});
