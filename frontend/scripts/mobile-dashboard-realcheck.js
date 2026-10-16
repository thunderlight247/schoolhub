// Mobile dashboard experience check through the REAL flow: opens /login at
// phone width, taps the persona button (cookies land exactly as a visitor's
// would), waits for the workspace to load, then measures the layout —
// headings, table density ("..." placeholders), ICLIPPED content, crushed
// cells, tiny text, and narrow stat cards.
//
// Run: node frontend/scripts/mobile-dashboard-realcheck.js
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";

const PERSONAS = [
  { role: "admin", label: "Sign in as Admin", home: "/dashboard/admin" },
  { role: "teacher", label: "Sign in as Teacher", home: "/portal/teacher" },
  { role: "parent", label: "Sign in as Parent", home: "/portal/parent" },
];

(async () => {
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });

  for (const p of PERSONAS) {
    const page = await browser.newPage();
    await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 3, isMobile: true, hasTouch: true });
    page.setDefaultNavigationTimeout(120000);

    try {
      await page.goto(`${BASE}/login`, { waitUntil: "networkidle2", timeout: 120000 });
      await page.waitForFunction(
        () => document.body && document.body.innerText.includes("Explore the school"),
        { timeout: 90000 },
      );
      await page.evaluate((label) => {
        const btn = Array.from(document.querySelectorAll("button")).find((b) => b.innerText.includes(label));
        btn.click();
      }, p.label);

      await page.waitForFunction(
        (want) => window.location.pathname.startsWith(want),
        { timeout: 120000, polling: 800 },
        p.home,
      );
      // Let the workspace's data fetches resolve.
      await new Promise((r) => setTimeout(r, 10000));

      const a = await page.evaluate(() => {
        const visible = (el) => {
          const r = el.getBoundingClientRect();
          return r.width > 0 && r.height > 0;
        };
        const text = document.body.innerText;
        const headings = Array.from(document.querySelectorAll("h1,h2,h3"))
          .slice(0, 7)
          .map((h) => (h.textContent || "").trim().slice(0, 32));
        const clipped = [];
        document.querySelectorAll("main *").forEach((el) => {
          if (!visible(el) || el.children.length === 0 || clipped.length >= 5) return;
          const style = getComputedStyle(el);
          const wrapped = /auto|scroll/.test(style.overflowX);
          if (el.scrollWidth - el.clientWidth > 24 && !wrapped && !el.closest('[class*="overflow-x"]')) {
            const cls = String(el.className).split(" ").slice(0, 5).join(" ");
            clipped.push(`${el.tagName.toLowerCase()}.${cls}`);
          }
        });
        const tables = Array.from(document.querySelectorAll("table")).map((t) => {
          const cols = t.querySelectorAll("thead th").length || (t.rows[0] ? t.rows[0].cells.length : 0);
          const wrapper = !!t.closest('[class*="overflow-x"]');
          let min = 9999;
          t.querySelectorAll("th,td").forEach((c) => {
            const w = c.getBoundingClientRect().width;
            if (w > 0 && w < min) min = w;
          });
          return { cols, wrapper, min: Math.round(min) };
        });
        // Cells narrower than 60px with content (visually crushed).
        let crushed = 0;
        document.querySelectorAll("td,th").forEach((c) => {
          const w = c.getBoundingClientRect().width;
          if (w > 0 && w < 60) crushed += 1;
        });
        // Topbar button heights (tap targets).
        const topTaps = Array.from(document.querySelectorAll("header button, header a")).slice(0, 6)
          .map((b) => Math.round(b.getBoundingClientRect().height));
        return {
          headings,
          tables,
          crushed,
          clipped,
          placeholders: (text.match(/\.\.\./g) || []).length,
          textLen: text.length,
          topTaps: topTaps.join(","),
        };
      });

      console.log(`\n=== ${p.role} → ${p.home} ===`);
      console.log(`  headings: ${a.headings.join(" | ") || "NONE"}`);
      console.log(`  textChars=${a.textLen} placeholders=${a.placeholders}`);
      console.log(`  tables: ${JSON.stringify(a.tables)}`);
      console.log(`  crushed cells: ${a.crushed} · clipped elements: ${a.clipped.length}`);
      a.clipped.forEach((c) => console.log(`    clipped: ${c}`));
      console.log(`  topbar tap heights: ${a.topTaps}`);
      await page.screenshot({ path: `/tmp/mobile-real-${p.role}.png` });
    } catch (e) {
      console.log(`\n=== ${p.role} === ERR ${String(e).slice(0, 150)}`);
    } finally {
      await page.close();
    }
  }
  await browser.close();
})();
