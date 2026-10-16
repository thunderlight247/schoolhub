// Mobile-friendliness audit: loads key pages at a phone viewport (390x844,
// DPR 3) and reports (1) document-level horizontal overflow (the classic
// "pinch-zoom sideways" bug), (2) scrollable wide elements, (3) viewport
// meta presence. Also captures screenshots to mobile-audit-*.png.
//
// Run: node frontend/scripts/mobile-audit.js [url ...]
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const PATHS = process.argv.slice(2).length
  ? process.argv.slice(2)
  : ["/", "/login"];
const CHROME = "/usr/bin/chromium";

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  const page = await browser.newPage();
  await page.setViewport({ width: 390, height: 844, deviceScaleFactor: 3, isMobile: true, hasTouch: true });
  page.setDefaultNavigationTimeout(90000);

  for (const path of PATHS) {
    const url = `${BASE}${path}`;
    try {
      await page.goto(url, { waitUntil: "networkidle2", timeout: 90000 });
      await new Promise((r) => setTimeout(r, 4000));
      const audit = await page.evaluate(() => {
        const doc = document.documentElement;
        const innerW = window.innerWidth;
        const overflowY =
          doc.scrollWidth - innerW > 2 || document.body.scrollWidth - innerW > 2;
        // Find the widest culprits: elements extending past the right edge.
        const wide = [];
        document.querySelectorAll("body *").forEach((el) => {
          const r = el.getBoundingClientRect();
          if (r.width > 40 && (r.right > innerW + 3 || r.left < -3) && !el.closest("nav[aria-label]")) {
            const cls = (el.className && String(el.className).slice(0, 40)) || "";
            if (wide.length < 6) wide.push(`${el.tagName.toLowerCase()}.${cls} right=${Math.round(r.right)}`);
          }
        });
        const viewportMeta = document.querySelector('meta[name="viewport"]')?.getAttribute("content") ?? null;
        // Tiny tap targets (interactive elements < ~40px tall)
        const tiny = [];
        document.querySelectorAll("button, a[href]").forEach((el) => {
          const r = el.getBoundingClientRect();
          if (r.height > 0 && r.height < 24 && tiny.length < 5) {
            tiny.push(`${el.tagName.toLowerCase()}:${(el.textContent || "").trim().slice(0, 20)}`);
          }
        });
        return { overflowY, innerW, scrollW: doc.scrollWidth, bodyScrollW: document.body.scrollWidth, wide, viewportMeta, tiny };
      });
      console.log(`=== ${path} ===`);
      console.log(`  viewport meta: ${audit.viewportMeta ?? "MISSING"}`);
      console.log(`  horizontal overflow: ${audit.overflowY ? "YES" : "no"} (window ${audit.innerW}, doc scrollWidth ${audit.scrollW}, body ${audit.bodyScrollW})`);
      if (audit.wide.length) console.log("  wide elements:\n    " + audit.wide.join("\n    "));
      if (audit.tiny.length) console.log("  tiny tap targets:\n    " + audit.tiny.join("\n    "));
      const name = path === "/" ? "home" : path.replace(/\//g, "-").replace(/^-/, "") || "root";
      await page.screenshot({ path: `/tmp/mobile-${name}.png`, fullPage: false });
      console.log(`  screenshot: /tmp/mobile-${name}.png`);
    } catch (e) {
      console.log(`=== ${path} ===\n  ERROR ${String(e).slice(0, 160)}\n`);
    }
  }
  await browser.close();
})();
