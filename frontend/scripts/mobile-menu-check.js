// Verify the homepage mobile menu in a phone-viewport browser: at 390px the
// hamburger must be visible, opening it must reveal the section links and
// the portal sign-in (which were previously hidden entirely on phones), and
// tapping a link must close the menu again.
//
// Run: node frontend/scripts/mobile-menu-check.js
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";

(async () => {
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 3, isMobile: true, hasTouch: true });
  page.setDefaultNavigationTimeout(90000);
  let ok = true;
  const check = (name, pass, extra = "") => {
    ok = ok && pass;
    console.log(`${pass ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
  };

  await page.goto(`${BASE}/`, { waitUntil: "networkidle2", timeout: 90000 });

  // Hamburger visible, menu closed by default.
  const burger = await page.$('button[aria-label="Open menu"]');
  check("hamburger visible at 390px", !!burger);
  const menuBefore = await page.$("#mobile-menu");
  check("menu closed by default", !menuBefore);

  // Open it: section links + portal sign-in appear.
  await burger.click();
  await page.waitForSelector("#mobile-menu", { timeout: 10000 });
  const navText = await page.$eval("#mobile-menu", (el) => el.innerText.replace(/\n/g, " | "));
  check(
    "menu shows section links + portal sign-in",
    navText.includes("Admissions") && navText.includes("Academics") && navText.includes("Portal sign-in"),
    navText.slice(0, 80),
  );

  // No horizontal overflow while the menu is open.
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  check("no horizontal overflow with menu open", overflow <= 2, `delta=${overflow}`);

  // Tap a link: menu closes.
  await page.evaluate(() => {
    const link = Array.from(document.querySelectorAll("#mobile-menu a")).find((a) => a.textContent.includes("Academics"));
    link.click();
  });
  await new Promise((r) => setTimeout(r, 800));
  const menuAfter = await page.$("#mobile-menu");
  check("menu closes on link tap", !menuAfter);

  // Sanity: no horizontal overflow on the whole page either.
  const overflow2 = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
  check("page never scrolls sideways at 390px", overflow2 <= 2, `delta=${overflow2}`);

  await browser.close();
  console.log(ok ? "\nall mobile-menu checks passed" : "\nSOME CHECKS FAILED");
  process.exit(ok ? 0 : 1);
})();
