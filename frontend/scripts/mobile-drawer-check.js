// Verify the app-shell slide-out drawer on a phone viewport: the hamburger
// opens the drawer, the persona's real navigation items appear with 36px+
// tap targets, tapping a link navigates and closes the drawer, and Escape
// also closes it. Signing in goes through the real persona buttons.
//
// Run: node frontend/scripts/mobile-drawer-check.js "Sign in as Teacher"
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const LABEL = process.argv[2] ?? "Sign in as Teacher";

(async () => {
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 3, isMobile: true, hasTouch: true });
  page.setDefaultNavigationTimeout(120000);
  let ok = true;
  const check = (name, pass, extra = "") => {
    ok = ok && pass;
    console.log(`${pass ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
  };

  // Sign in through the real flow.
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle2", timeout: 120000 });
  await page.waitForFunction(
    () => document.body && document.body.innerText.includes("Explore the school"),
    { timeout: 90000 },
  );
  await page.evaluate((l) => {
    const b = Array.from(document.querySelectorAll("button")).find((x) => x.innerText.includes(l));
    b.click();
  }, LABEL);
  await page.waitForFunction(
    () => /\/(portal|dashboard)\//.test(window.location.pathname),
    { timeout: 120000, polling: 800 },
  );
  await new Promise((r) => setTimeout(r, 8000));

  // Stale mobile nav strip must be gone.
  const strip = await page.$('nav[aria-label="Primary (mobile)"]');
  check("old scrolling nav strip removed", !strip);

  // Hamburger visible; drawer closed by default; no horizontal overflow.
  const burger = await page.$('button[aria-label="Open menu"]');
  check("hamburger visible", !!burger);
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  check("no horizontal overflow", overflow <= 2, `delta=${overflow}`);

  // Open the drawer (Tailwind translates via classes → check computed style;
  // translate-x-0 resolves to matrix(1,0,0,1,0,0), closed is x<0).
  await burger.click();
  await page.waitForFunction(() => {
    const d = document.getElementById("app-drawer");
    if (!d) return false;
    const x = new DOMMatrixReadOnly(getComputedStyle(d).transform).m41;
    return x > -5;
  }, { timeout: 8000 });
  const drawerText = await page.$eval("#app-drawer", (el) => el.innerText.replace(/\n/g, " | "));
  check(
    "drawer shows navigation + user card",
    drawerText.includes("Sign out") &&
      (drawerText.includes("Reports") || drawerText.includes("Attendance") || drawerText.includes("Portal")),
    drawerText.slice(0, 90),
  );
  const tapOk = await page.evaluate(() =>
    Array.from(document.querySelectorAll("#app-drawer nav a")).every(
      (a) => a.getBoundingClientRect().height >= 40,
    ),
  );
  check("drawer links ≥40px tap targets", tapOk);

  // Navigate via the drawer: URL changes and drawer closes. The Admin
  // persona has an Admin Dashboard button (~"Attendance") — pick a link
  // that exists for every persona instead.
  await page.evaluate(() => {
    const links = Array.from(document.querySelectorAll("#app-drawer nav a"));
    const link = links.find((a) => a.textContent.includes("Reports")) ?? links[1];
    link.click();
  });
  await page.waitForFunction(() => window.location.pathname.startsWith("/report"), {
    timeout: 60000,
    polling: 500,
  });
  await new Promise((r) => setTimeout(r, 3000));
  const closed = await page.evaluate(() => {
    const d = document.getElementById("app-drawer");
    if (!d) return false;
    const x = new DOMMatrixReadOnly(getComputedStyle(d).transform).m41;
    return x < -5; // fully off-canvas
  });
  check("drawer navigated and closed", closed, `url=${page.url()}`);
  const active = await page.evaluate(() =>
    !!document.querySelector('#app-drawer nav a[aria-current="page"]'),
  );
  check("active item highlighted inside drawer", active);

  // Escape closes the drawer too.
  const burger2 = await page.$('button[aria-label="Open menu"]');
  await burger2.click();
  await page.waitForFunction(() => {
    const d = document.getElementById("app-drawer");
    if (!d) return false;
    const x = new DOMMatrixReadOnly(getComputedStyle(d).transform).m41;
    return x > -5;
  }, { timeout: 8000 });
  await page.keyboard.press("Escape");
  await page.waitForFunction(() => {
    const d = document.getElementById("app-drawer");
    if (!d) return false;
    const x = new DOMMatrixReadOnly(getComputedStyle(d).transform).m41;
    return x < -5;
  }, { timeout: 8000 });
  check("Escape closes the drawer", true);

  console.log(ok ? "\nall drawer checks passed" : "\nSOME CHECKS FAILED");
  await browser.close();
  process.exit(ok ? 0 : 1);
})();
