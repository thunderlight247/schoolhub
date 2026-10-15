// Live verification of the M33 polish pass:
//   overflow sweep (desktop 1200 + mobile 390) across every admin route and
//   both parent/student portals → tabs a11y (role=tablist, aria-selected,
//   arrow-key selection on /reports + /portal/student) → modal focus trap
//   entry/Escape-restore on /audit → label htmlFor integrity in a live modal
//   (/announcements composer).
//
// Run: node frontend/scripts/polish-check.js (backend on :8080, frontend on
// :3000 in CLASSIC mode — Clerk keys temporarily out of .env.local).
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

const VIEWPORTS = [
  { name: "desktop", width: 1200, height: 900 },
  { name: "mobile", width: 390, height: 844 },
];

const ROUTES = {
  admin: [
    "/dashboard", "/students", "/guardians", "/staff", "/academics",
    "/enrollments", "/attendance", "/assessments", "/teaching", "/access",
    "/announcements", "/reports", "/audit",
  ],
  parent: ["/portal/parent"],
  student: ["/portal/student"],
};

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function launch() {
  return puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
  });
}

// Log a role in through the real form; returns the page to reuse per context.
async function formLogin(browser, email, password) {
  const page = await browser.newPage();
  await page.setViewport({ width: 1200, height: 900 });
  await page.goto("http://localhost:3000/login", { waitUntil: "networkidle2", timeout: 45000 });
  await sleep(1200);
  await page.type("#email", email);
  await page.type("#password", password);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle2", timeout: 30000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
  await sleep(1500);
  return page;
}

async function overflowProbe(browser, url, viewport) {
  const page = await browser.newPage();
  try {
    await page.setViewport(viewport);
    // Reuse the role's session cookie by sharing storage? Simplest: new page
    // inherits nothing — so the caller pre-injects the session cookie.
    for (const c of cookies) {
      await page.setCookie({ ...c, url: "http://localhost:3000" });
    }
    await page.goto(`http://localhost:3000${url}`, { waitUntil: "networkidle2", timeout: 45000 });
    // Turbopack cold-compile: poll until interactive and settled.
    await sleep(2500);
    return await page.evaluate(() => ({
      overflow: document.documentElement.scrollWidth - window.innerWidth,
    }));
  } finally {
    await page.close().catch(() => {});
  }
}

let cookies = [];

(async () => {
  if (!(await qa.requireClassicMode("polish-check"))) process.exit(0);
  const browser = await launch();

  // ---- 1. Login each role through the form, harvest cookies for the sweep.
  const adminPage = await formLogin(browser, qa.loadCreds().masterEmail, qa.loadCreds().masterPassword);
  const adminCookies = await adminPage.cookies();
  await adminPage.close();
  cookies = adminCookies;
  check("admin form login", adminCookies.some((c) => c.name === "schoolhub_session"));

  const parentPage = await formLogin(browser, qa.loadCreds().parentEmail, qa.loadCreds().parentPassword);
  const parentCookies = await parentPage.cookies();
  await parentPage.close();
  check("parent form login", parentCookies.some((c) => c.name === "schoolhub_session"));

  const studentPage = await formLogin(browser, qa.loadCreds().studentEmail, qa.loadCreds().studentPassword);
  const studentCookies = await studentPage.cookies();
  await studentPage.close();
  check("student form login", studentCookies.some((c) => c.name === "schoolhub_session"));

  // ---- 2. Overflow sweep: every route × every viewport must not overflow.
  const roleCookies = { admin: adminCookies, parent: parentCookies, student: studentCookies };
  let offenders = [];
  for (const role of Object.keys(ROUTES)) {
    cookies = roleCookies[role];
    for (const route of ROUTES[role]) {
      for (const vp of VIEWPORTS) {
        const r = await overflowProbe(browser, route, vp);
        if (r.overflow <= 2) {
          check(`${role} ${route} @${vp.name} no overflow`, true, `+${r.overflow}px`);
        } else {
          offenders.push(`${role} ${route} @${vp.name} (+${r.overflow}px)`);
        }
      }
    }
  }
  check(
    "overflow sweep (all routes, both viewports)",
    offenders.length === 0,
    offenders.length ? `offenders: ${offenders.join(" | ")}` : "all clean",
  );
  cookies = adminCookies;

  // ---- 3. Tabs a11y on /reports (admin): tablist + aria-selected + arrow nav.
  const rpage = await browser.newPage();
  await rpage.setViewport({ width: 1200, height: 900 });
  // Explicit admin session: every browser.newPage() shares ONE cookie context,
  // whose schoolhub_session was overwritten by the LAST login (student).
  for (const c of adminCookies) {
    await rpage.setCookie({ ...c, url: "http://localhost:3000" });
  }
  await rpage.goto("http://localhost:3000/reports", { waitUntil: "networkidle2", timeout: 45000 });
  await sleep(2500);
  const tabInfo = await rpage.evaluate(() => {
    const tl = document.querySelector('[role="tablist"]');
    const tabs = tl ? [...tl.querySelectorAll('[role="tab"]')] : [];
    return {
      hasTablist: !!tl,
      count: tabs.length,
      selectedIdx: tabs.findIndex((t) => t.getAttribute("aria-selected") === "true"),
      selectedTabindex: tabs.map((t) => t.getAttribute("tabindex")),
    };
  });
  check(
    "/reports tablist + aria-selected",
    tabInfo.hasTablist && tabInfo.count === 4 && tabInfo.selectedIdx === 0 &&
      tabInfo.selectedTabindex[0] === "0",
    JSON.stringify(tabInfo),
  );
  // Keyboard users tab to the widget first — mirror that before arrowing.
  await rpage.evaluate(() =>
    document.querySelector('[role="tablist"] [role="tab"][aria-selected="true"]')?.focus(),
  );
  await sleep(200);
  await rpage.keyboard.press("ArrowRight");
  await sleep(400);
  const afterArrow = await rpage.evaluate(() => {
    const tabs = [...document.querySelectorAll('[role="tablist"] [role="tab"]')];
    const sel = tabs.findIndex((t) => t.getAttribute("aria-selected") === "true");
    return { sel, label: tabs[sel]?.textContent, focused: document.activeElement?.textContent };
  });
  check(
    "ArrowRight moves reports selection (focus follows)",
    afterArrow.sel === 1 && (afterArrow.focused || "").includes("Attendance"),
    JSON.stringify(afterArrow),
  );
  await rpage.close();

  // Keyboard nav on the student portal's five tabs.
  const spage = await browser.newPage();
  await spage.setViewport({ width: 1200, height: 900 });
  for (const c of studentCookies) {
    await spage.setCookie({ ...c, url: "http://localhost:3000" });
  }
  await spage.goto("http://localhost:3000/portal/student", { waitUntil: "networkidle2", timeout: 45000 });
  await sleep(2500);
  const sTabs = await (async () => {
    try {
      const tl = await spage.$('[role="tablist"]');
      if (!tl) return { first: false, detail: "no tablist", second: false, detail2: "skipped" };
      const info = await spage.evaluate(() => {
        const tabs = [...document.querySelectorAll('[role="tablist"] [role="tab"]')];
        return { count: tabs.length, sel: tabs.findIndex((t) => t.getAttribute("aria-selected") === "true") };
      });
      await spage.evaluate(() =>
        document.querySelector('[role="tablist"] [role="tab"][aria-selected="true"]')?.focus(),
      );
      await sleep(200);
      await spage.keyboard.press("ArrowRight");
      await sleep(400);
      const sel2 = await spage.evaluate(() => {
        const tabs = [...document.querySelectorAll('[role="tablist"] [role="tab"]')];
        return tabs.findIndex((t) => t.getAttribute("aria-selected") === "true");
      });
      return {
        first: info.count === 5 && info.sel === 0,
        detail: `tabs=${info.count} sel=${info.sel}`,
        second: sel2 === 1,
        detail2: `sel after ArrowRight=${sel2}`,
      };
    } catch (e) {
      return { first: false, detail: e.message, second: false, detail2: "error" };
    }
  })();
  check("student portal tablist + aria-selected", sTabs.first, sTabs.detail);
  check("student portal ArrowRight moves selection", sTabs.second, sTabs.detail2);
  await spage.close();

  // ---- 4. Modal focus entry + Escape restore on /audit (admin session).
  const apage = await browser.newPage();
  await apage.setViewport({ width: 1200, height: 900 });
  for (const c of adminCookies) {
    await apage.setCookie({ ...c, url: "http://localhost:3000" });
  }
  await apage.goto("http://localhost:3000/audit", { waitUntil: "networkidle2", timeout: 45000 });
  for (let i = 0; i < 12; i++) {
    await sleep(2500);
    const t = (await apage.evaluate(() => document.body.innerText)).toLowerCase();
    if (t.includes("audit log") && t.includes("view")) break;
  }
  const viewBtn = await apage.evaluateHandle(() =>
    [...document.querySelectorAll("button")].find((b) => b.textContent.trim().toLowerCase() === "view"),
  );
  const viewOk = viewBtn && (await viewBtn.asElement()) !== null;
  if (viewOk) {
    await viewBtn.asElement().click();
    await sleep(800);
    const focusInfo = await apage.evaluate(() => {
      const dlg = document.querySelector('[role="dialog"]');
      return {
        open: !!dlg,
        focusInside: dlg ? dlg.contains(document.activeElement) : false,
        focusLabel: document.activeElement?.getAttribute("aria-label") ?? document.activeElement?.textContent?.trim() ?? "",
      };
    });
    check(
      "audit detail modal focuses inside dialog",
      focusInfo.open && focusInfo.focusInside,
      `focus=${focusInfo.focusLabel}`,
    );
    await apage.keyboard.press("Escape");
    await sleep(600);
    const closed = await apage.evaluate(() => !document.querySelector('[role="dialog"]'));
    check("Escape closes audit detail modal", closed);
  } else {
    check("audit detail modal focuses inside dialog", false, "no View button found");
    check("Escape closes audit detail modal", false, "skipped");
  }

  // ---- 5. Label htmlFor integrity inside the live announcements composer modal.
  const npage = await browser.newPage();
  await npage.setViewport({ width: 1200, height: 900 });
  for (const c of adminCookies) {
    await npage.setCookie({ ...c, url: "http://localhost:3000" });
  }
  await npage.goto("http://localhost:3000/announcements", { waitUntil: "networkidle2", timeout: 45000 });
  await sleep(2500);
  const newBtn = await npage.evaluateHandle(() =>
    [...document.querySelectorAll("button")].find((b) => b.textContent.includes("New Announcement")),
  );
  if ((await newBtn.asElement()) !== null) {
    await newBtn.asElement().click();
    await sleep(800);
    const labels = await npage.evaluate(() =>
      [...document.querySelectorAll('[role="dialog"] label')].map((l) => ({
        htmlFor: l.getAttribute("for"),
        text: l.textContent.trim().slice(0, 30),
        linked: !!l.getAttribute("for") && !!document.getElementById(l.getAttribute("for")),
      })),
    );
    const allLinked = labels.length > 0 && labels.every((l) => l.linked);
    check("composer modal labels all point at real inputs", allLinked, JSON.stringify(labels));
  } else {
    check("composer modal labels all point at real inputs", false, "no New Announcement button");
  }
  await npage.close();

  await browser.close().catch(() => {});

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nPOLISH M33: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("POLISH M33 failed:", e.message);
  process.exit(1);
});
