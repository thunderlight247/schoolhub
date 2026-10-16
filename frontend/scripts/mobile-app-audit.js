// Mobile-friendliness audit of the authenticated app: signs in through the
// demo API (same one-click flow a visitor uses), injects the session cookie
// into a phone-viewport Chrome, and checks each app page for horizontal
// overflow, wide element culprits, and tiny tap targets.
//
// Run: node frontend/scripts/mobile-app-audit.js [api-base-url]
const puppeteer = require("puppeteer-core");

const BASE = "https://schoolhub-frontend.schoolhub-demo.workers.dev";
const API = process.argv[2] ?? "https://schoolhub-api-98a7.onrender.com/api/v1";

// Pages grouped by the persona that can see them.
const PAGES = {
  admin: [
    "/dashboard/admin",
    "/students",
    "/academics",
    "/attendance",
    "/assessments",
    "/reports",
    "/announcements",
  ],
  teacher: ["/portal/teacher"],
  parent: ["/portal/parent"],
};
const CHROME = "/usr/bin/chromium";

async function demoCookie(role) {
  const res = await fetch(`${API}/auth/demo-login`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ as: role }),
  });
  if (!res.ok) throw new Error(`demo-login ${role} → ${res.status}`);
  const raw = res.headers.get("set-cookie") ?? "";
  const value = raw.slice(raw.indexOf("=") + 1, raw.indexOf(";"));
  return { name: raw.slice(0, raw.indexOf("=")), value };
}

async function auditPage(page, path, label) {
  try {
    await page.goto(`${BASE}${path}`, { waitUntil: "networkidle2", timeout: 90000 });
    await new Promise((r) => setTimeout(r, 6000));
    const a = await page.evaluate(() => {
      const doc = document.documentElement;
      const innerW = window.innerWidth;
      const overflow = doc.scrollWidth - innerW > 2 || document.body.scrollWidth - innerW > 2;
      const wide = [];
      document.querySelectorAll("body *").forEach((el) => {
        const r = el.getBoundingClientRect();
        if (r.width > 60 && (r.right > innerW + 3 || r.left < -3) && wide.length < 5) {
          const cls = (el.className && String(el.className).split(" ").slice(0, 4).join(" ")) || "";
          wide.push(`${el.tagName.toLowerCase()}.${cls} right=${Math.round(r.right)} left=${Math.round(r.left)}`);
        }
      });
      const tiny = [];
      document.querySelectorAll("button, a[href]").forEach((el) => {
        const r = el.getBoundingClientRect();
        if (r.height > 0 && r.height < 24 && tiny.length < 4) {
          tiny.push(`${el.tagName.toLowerCase()}:${(el.textContent || "").trim().slice(0, 18)}`);
        }
      });
      const bodyText = document.body.innerText.slice(0, 80).replace(/\n/g, " | ");
      return { overflow, w: doc.scrollWidth, wide, tiny, bodyText };
    });
    const issues = [];
    if (a.overflow) issues.push("OVERFLOW");
    if (a.tiny.length) issues.push("tiny-taps");
    console.log(`${issues.length ? "⚠️ " : "✓ "} ${label} ${path}${issues.length ? ` — ${issues.join(", ")}` : ""}`);
    if (a.overflow) {
      console.log(`   window 390 / doc scrollWidth ${a.w}`);
      a.wide.forEach((t) => console.log(`   wide: ${t}`));
    }
    a.tiny.forEach((t) => console.log(`   tiny: ${t}`));
    const name = path.replace(/\//g, "-").replace(/^-/, "");
    await page.screenshot({ path: `/tmp/mobile-app-${name}.png` });
    return { path, overflow: a.overflow, tiny: a.tiny };
  } catch (e) {
    console.log(`ERR ${label} ${path}: ${String(e).slice(0, 140)}`);
    return null;
  }
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath: CHROME,
    headless: "new",
    args: ["--no-sandbox", "--disable-dev-shm-usage"],
  });
  let overflowCount = 0;

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
      const r = await auditPage(page, path, `[${role}]`);
      if (r) overflowCount += r.overflow ? 1 : 0;
    }
    await page.close();
  }

  console.log(`\n${overflowCount} page(s) with horizontal overflow`);
  await browser.close();
  process.exit(0);
})();
