// Deep mobile layout diagnostic: at 390px it measures what the eyes see,
// beyond page overflow —
//   * elements whose content is CLIPPED (scrollWidth > clientWidth) with no
//     scroll affordance (the "words cut off" failure),
//   * tables: column count, whether they scroll (overflow-x) or get squeezed,
//     and the narrowest data column,
//   * grids/cells narrower than ~110px (visually crushed cards).
//
// Run: node frontend/scripts/mobile-squeeze-audit.js
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const API = "https://schoolhub-api-98a7.onrender.com/api/v1";

const PAGES = {
  admin: ["/dashboard/admin", "/students", "/reports", "/attendance"],
  teacher: ["/portal/teacher"],
  parent: ["/portal/parent"],
};

async function demoCookie(role) {
  const res = await fetch(`${API}/auth/demo-login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ as: role }),
  });
  if (!res.ok) throw new Error(`demo-login ${role} → ${res.status}`);
  const raw = res.headers.get("set-cookie") ?? "";
  return { name: raw.slice(0, raw.indexOf("=")), value: raw.slice(raw.indexOf("=") + 1, raw.indexOf(";")) };
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });

  for (const [role, paths] of Object.entries(PAGES)) {
    const ck = await demoCookie(role);
    const page = await browser.newPage();
    await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 3, isMobile: true, hasTouch: true });
    await page.setCookie({
      name: ck.name,
      value: ck.value,
      domain: new URL(API).hostname,
      path: "/",
      httpOnly: true,
      secure: true,
    });

    for (const path of paths) {
      try {
        await page.goto(`${BASE}${path}`, { waitUntil: "networkidle2", timeout: 90000 });
        await new Promise((r) => setTimeout(r, 6000));
        const a = await page.evaluate(() => {
          const out = { clipped: [], tables: [], crushed: [] };
          const visible = (el) => {
            const r = el.getBoundingClientRect();
            return r.width > 0 && r.height > 0;
          };

          // Content clipped with no scroll container around it.
          document.querySelectorAll("body *").forEach((el) => {
            if (!visible(el) || el.children.length === 0) return;
            const sw = el.scrollWidth;
            const cw = el.clientWidth;
            const style = getComputedStyle(el);
            const scrollable = /auto|scroll/.test(style.overflowX) || el.closest('[class*="overflow-x"]');
            if (sw - cw > 24 && !scrollable && out.clipped.length < 6) {
              const cls = String(el.className).split(" ").slice(0, 5).join(" ");
              out.clipped.push(`${el.tagName.toLowerCase()}.${cls} (scrollW ${sw} > clientW ${cw})`);
            }
          });

          // Tables.
          document.querySelectorAll("table").forEach((t, i) => {
            if (!visible(t) || i > 2) return;
            const cols = t.querySelectorAll("thead th, tr:first-child > td, tr:first-child > th").length;
            const wrapper = t.closest('[class*="overflow-x"]');
            let minW = 9999;
            let squeezed = false;
            t.querySelectorAll("td, th").forEach((c) => {
              const w = c.getBoundingClientRect().width;
              if (w > 0 && w < minW) minW = w;
              const r = c.getBoundingClientRect();
              if (c.scrollWidth > Math.ceil(r.width) + 6) squeezed = true;
            });
            const cls = String(t.className).split(" ").slice(0, 4).join(" ");
            out.tables.push(
              `cols=${cols} scrollWrap=${!!wrapper} minCellW=${Math.round(minW)} squeezed=${squeezed} (${cls})`,
            );
          });

          // Visually crushed fixed cells (very narrow text cells).
          document
            .querySelectorAll("main div, main td")
            .forEach((el) => {
              if (out.crushed.length >= 5) return;
              if (!visible(el) || !(el.textContent || "").trim()) return;
              const r = el.getBoundingClientRect();
              if (r.width > 40 && r.width < 60 && el.scrollWidth > r.width + 4) {
                const cls = String(el.className).split(" ").slice(0, 4).join(" ");
                out.crushed.push(`${el.tagName.toLowerCase()}.${cls} w=${Math.round(r.width)}`);
              }
            });

          return out;
        });

        console.log(`\n=== [${role}] ${path} ===`);
        if (a.clipped.length) {
          console.log("  CLIPPED CONTENT:");
          a.clipped.forEach((t) => console.log(`    ${t}`));
        }
        a.tables.forEach((t) => console.log(`  table: ${t}`));
        if (a.crushed.length) {
          console.log("  CRUSHED CELLS:");
          a.crushed.forEach((t) => console.log(`    ${t}`));
        }
        if (!a.clipped.length && !a.crushed.length) console.log("  (no clipping/crush)");
      } catch (e) {
        console.log(`\n=== [${role}] ${path} === ERR ${String(e).slice(0, 120)}`);
      }
    }
    await page.close();
  }
  await browser.close();
})();
