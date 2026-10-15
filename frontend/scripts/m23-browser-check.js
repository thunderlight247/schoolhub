// One-off M23 browser verification (run with node, not part of the build):
// admin login -> shell renders with sidebar identity -> sign out ->
// teacher login -> teacher portal + admin denial -> logout -> protected
// redirect. Also verifies the inline bad-password error.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";
async function login(page, email, password) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
  await page.type("#email", email);
  await page.type("#password", password);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
}

async function signOut(page) {
  const buttons = await page.$$("button");
  for (const b of buttons) {
    const t = await b.evaluate((el) => el.textContent);
    if (t?.trim() === "Sign out") {
      await Promise.all([
        page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
        b.click(),
      ]);
      return true;
    }
  }
  return false;
}

(async () => {
  if (!(await qa.requireClassicMode("m23-browser-check"))) process.exit(0);
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
  });
  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 900 }); // lg: shell breakpoint
    let failures = 0;
    const check = (name, ok) => {
      console.log(`${ok ? "PASS" : "FAIL"} ${name}`);
      if (!ok) failures++;
    };

    // 1. Admin login -> lands on admin dashboard, shell renders.
    await login(page, qa.loadCreds().masterEmail, qa.loadCreds().masterPassword);
    check("admin redirected to dashboard", page.url().includes("/dashboard/admin"));
    const dashText = await page.evaluate(() => document.body.innerText);
    check("dashboard renders under shell", dashText.includes("Institutional Overview"));
    check("sidebar shows user identity + role", dashText.includes("Demo Admin") && dashText.includes("ADMIN"));
    check("sign out button present", dashText.includes("Sign out"));

    // 2. Sign out via the shell.
    const signedOut = await signOut(page);
    check("sign out returns to login", signedOut && page.url().includes("/login"));

    // 3. Teacher login -> teacher portal renders.
    await login(page, qa.loadCreds().teacherEmail, qa.loadCreds().teacherPassword);
    check("teacher redirected to portal", page.url().includes("/portal/teacher"));
    const teacherText = (await page.evaluate(() => document.body.innerText)).toLowerCase();
    check("teacher portal renders for teacher", teacherText.includes("teacher workspace"));

    // 4. Role denial: teacher cannot see the admin dashboard.
    await page.goto(`${BASE}/dashboard/admin`, { waitUntil: "networkidle0" });
    const deniedText = await page.evaluate(() => document.body.innerText);
    check("teacher denied on admin dashboard", deniedText.includes("Access denied") && deniedText.includes("ADMIN"));

    // 5. Bad password shows an inline error.
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    await page.type("#email", qa.loadCreds().masterEmail);
    await page.type("#password", "wrong-password");
    await page.click("button[type=submit]");
    await page.waitForSelector("[role=alert]", { timeout: 10000 });
    const alertText = await page.evaluate(
      () => document.querySelector("[role=alert]")?.textContent ?? "",
    );
    check("bad password shows inline error", alertText.includes("Invalid email or password"));

    // 6. Protected route redirects when signed out (clear the teacher session).
    const cdp = await page.createCDPSession();
    await cdp.send("Network.clearBrowserCookies");
    await page.goto(`${BASE}/dashboard/admin`, { waitUntil: "networkidle0" });
    await page
      .waitForFunction(() => window.location.pathname === "/login", { timeout: 10000 })
      .catch(() => {});
    check("protected route redirects to login", page.url().includes("/login"));

    console.log(failures === 0 ? "\nALL BROWSER CHECKS PASSED" : `\n${failures} CHECK(S) FAILED`);
    process.exit(failures === 0 ? 0 : 1);
  } finally {
    await browser.close();
  }
})();
