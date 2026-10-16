// What does each dashboard actually show at phone width? Dumps headings,
// whether data loaded (vs "..." placeholders), and table presence, so a
// "doesn't look mobile friendly" complaint can be tied to concrete state.
//
// Run: node frontend/scripts/mobile-content-probe.js
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const API = "https://schoolhub-api-98a7.onrender.com/api/v1";

const PAGES = {
  admin: ["/dashboard/admin", "/students", "/attendance", "/reports"],
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
        await page.goto(`${BASE}${path}`, { waitUntil: "networkidle2", timeout: 120000 });
        await new Promise((r) => setTimeout(r, 9000));
        const a = await page.evaluate(() => {
          const text = document.body.innerText;
          const headings = Array.from(document.querySelectorAll("main h1, main h2, main h3"))
            .slice(0, 6)
            .map((h) => (h.textContent || "").trim().slice(0, 40));
          return {
            tables: document.querySelectorAll("table").length,
            tableRows: document.querySelectorAll("tbody tr").length,
            ellipsis: (text.match(/\.\.\./g) || []).length,
            headings,
            textLen: text.length,
          };
        });
        console.log(`[${role}] ${path}`);
        console.log(`  headings: ${a.headings.join(" | ")}`);
        console.log(`  tables=${a.tables} rows=${a.tableRows} placeholders=${a.ellipsis} textChars=${a.textLen}`);
      } catch (e) {
        console.log(`[${role}] ${path} ERR ${String(e).slice(0, 120)}`);
      }
    }
    await page.close();
  }
  await browser.close();
})();
