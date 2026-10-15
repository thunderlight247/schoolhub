// Live verification of the M32 Audit Log UI against the M19 backend.
// API-driven + classic-mode browser step, mirroring reports-check.js:
//   login (admin + teacher) → list shape → action dropdown source →
//   detail get/404 → filters (action/resource_type/dates) → pagination →
//   validation 400s → teacher 403 (audit.view is ADMIN/MASTER_ADMIN only)
//   → unauthenticated 401 → browser render of /audit through the form login.
//
// Run: node frontend/scripts/audit-check.js (backend on :8080, frontend on
// :3000 in classic mode — Clerk keys temporarily out of .env.local).
const puppeteer = require("puppeteer-core");

const BASE = "http://localhost:8080/api/v1";

const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

async function j(method, path, body, cookie) {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(cookie ? { Cookie: cookie } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const setCookie = res.headers.getSetCookie?.() ?? [];
  const sc = setCookie.find((c) => c.startsWith("schoolhub_session="));
  let json = null;
  try { json = await res.json(); } catch {}
  return { status: res.status, json, cookie: cookie ?? (sc ? sc.split(";")[0] : undefined) };
}

(async () => {
  const qa = require("./lib/check-accounts");
  // 1. Admin + teacher logins (QA personas — see scripts/lib/check-accounts.js)
  const admin = await qa.login("master");
  check("admin login", admin.status === 200 && !!admin.cookie);
  const adminCookie = admin.cookie;

  const teacher = await qa.login("teacher");
  check("teacher login", teacher.status === 200 && !!teacher.cookie);

  // 2. List shape: { audit_logs: [...], pagination: {...} }
  const list = await j("GET", "/audit-logs", null, adminCookie);
  const logs = list.json?.audit_logs ?? [];
  const pg = list.json?.pagination;
  check(
    "list shape (audit_logs + pagination)",
    list.status === 200 &&
      Array.isArray(logs) &&
      logs.length > 0 &&
      typeof pg?.total === "number" &&
      pg.total > 0 &&
      typeof pg?.total_pages === "number",
    `total=${pg?.total} rows=${logs.length}`,
  );
  const first = logs[0];
  check(
    "entry DTO fields",
    !!first &&
      typeof first.id === "string" &&
      typeof first.action === "string" &&
      typeof first.resource_type === "string" &&
      typeof first.created_at === "string" &&
      first.created_at.includes("T") && // RFC3339, not a bare date
      typeof first.metadata === "object",
  );

  // 3. Actions dropdown source: distinct codes incl. a known one
  const actions = await j("GET", "/audit-logs/actions", null, adminCookie);
  const actionList = actions.json?.actions ?? [];
  check(
    "actions endpoint returns distinct codes",
    actions.status === 200 &&
      Array.isArray(actionList) &&
      actionList.length > 0 &&
      actionList.every((a) => typeof a === "string") &&
      sorted(actionList),
    `${actionList.length} actions`,
  );
  function sorted(xs) {
    for (let i = 1; i < xs.length; i++) if (xs[i - 1] > xs[i]) return false;
    return true;
  }

  // 4. Detail get + 404
  const detail = await j("GET", `/audit-logs/${first.id}`, null, adminCookie);
  check(
    "detail get by id",
    detail.status === 200 && detail.json?.audit_log?.id === first.id,
  );
  const gone = await j("GET", "/audit-logs/99999999-9999-9999-9999-999999999999", null, adminCookie);
  check("detail unknown id → 404", gone.status === 404);

  // 5. Filters actually filter (backend re-checks what the page's dropdowns do)
  const actionProbe = actionList[0];
  const byAction = await j("GET", `/audit-logs?action=${actionProbe}`, null, adminCookie);
  check(
    `action filter (${actionProbe}) → all rows match`,
    byAction.status === 200 &&
      (byAction.json?.audit_logs ?? []).every((r) => r.action === actionProbe) &&
      (byAction.json?.audit_logs ?? []).length > 0,
  );

  const resTypeProbe = first.resource_type;
  const byResource = await j("GET", `/audit-logs?resource_type=${resTypeProbe}`, null, adminCookie);
  check(
    `resource_type filter (${resTypeProbe}) → all rows match`,
    byResource.status === 200 &&
      (byResource.json?.audit_logs ?? []).every((r) => r.resource_type === resTypeProbe),
  );

  // Actor filter via the entry's own actor (admin wrote plenty)
  if (first.actor_user_id) {
    const byActor = await j("GET", `/audit-logs?actor_user_id=${first.actor_user_id}`, null, adminCookie);
    check(
      "actor_user_id filter → all rows match",
      byActor.status === 200 &&
        (byActor.json?.audit_logs ?? []).every((r) => r.actor_user_id === first.actor_user_id),
    );
  }

  // Date window that covers everything vs. one that covers nothing
  const wideFrom = await j(
    "GET",
    "/audit-logs?date_from=" + encodeURIComponent("2026-01-01T00:00:00Z"),
    null,
    adminCookie,
  );
  const narrow = await j(
    "GET",
    "/audit-logs?date_to=" + encodeURIComponent("2025-01-01T00:00:00Z"),
    null,
    adminCookie,
  );
  check(
    "date filters narrow correctly",
    wideFrom.status === 200 && (wideFrom.json?.pagination?.total ?? 0) > 0 &&
      narrow.status === 200 && (narrow.json?.pagination?.total ?? 0) === 0,
    `wide total=${wideFrom.json?.pagination?.total} narrow total=${narrow.json?.pagination?.total}`,
  );

  // 6. Pagination mechanics (page_size=2, page 1 vs page 2 differ)
  const p1 = await j("GET", "/audit-logs?page=1&page_size=2", null, adminCookie);
  const p2 = await j("GET", "/audit-logs?page=2&page_size=2", null, adminCookie);
  const p1rows = p1.json?.audit_logs ?? [];
  const p2rows = p2.json?.audit_logs ?? [];
  check(
    "pagination respects page/page_size",
    p1.status === 200 && p2.status === 200 &&
      p1rows.length === 2 && p2rows.length === 2 &&
      p1rows[0].id !== p2rows[0].id &&
      p1.json?.pagination?.total_pages > 1,
    `total_pages=${p1.json?.pagination?.total_pages}`,
  );

  // 7. Validation 400s (the surface the page would hit on bad input)
  const badActor = await j("GET", "/audit-logs?actor_user_id=banana", null, adminCookie);
  check("bad actor_user_id → 400", badActor.status === 400);
  const badDate = await j("GET", "/audit-logs?date_from=2026-01-01", null, adminCookie); // bare date, not RFC3339
  check("bare-date date_from → 400 (RFC3339 required)", badDate.status === 400);
  const badPage = await j("GET", "/audit-logs?page=zero", null, adminCookie);
  check("non-integer page → 400", badPage.status === 400);

  // 8. Teacher 403 — audit.view is ADMIN/MASTER_ADMIN only
  const teacherForbid = await j("GET", "/audit-logs", null, teacher.cookie);
  check("teacher without audit.view → 403", teacherForbid.status === 403, `got ${teacherForbid.status}`);

  // 9. Unauthenticated 401
  const unauth = await j("GET", "/audit-logs");
  check("unauthenticated list → 401", unauth.status === 401);

  // 10. Browser render: /audit shows table + filter dropdowns (classic mode)
  if (puppeteer && (await qa.requireClassicMode("audit-check"))) {
    const browser = await puppeteer.launch({
      executablePath: "/usr/bin/chromium",
      headless: "new",
      args: ["--no-sandbox", "--disable-gpu"],
      defaultViewport: { width: 1440, height: 900 },
    });
    try {
      const page = await browser.newPage();
      await page.goto("http://localhost:3000/login", { waitUntil: "networkidle2", timeout: 45000 });
      await new Promise((r) => setTimeout(r, 1200));
      await page.type("#email", qa.loadCreds().masterEmail);
      await page.type("#password", qa.loadCreds().masterPassword);
      await Promise.all([
        page.waitForNavigation({ waitUntil: "networkidle2", timeout: 30000 }).catch(() => {}),
        page.click("button[type=submit]"),
      ]);
      await new Promise((r) => setTimeout(r, 1500));
      await page.goto("http://localhost:3000/audit", { waitUntil: "networkidle2", timeout: 45000 });
      // Turbopack cold-compiles the route on first hit — poll for settle.
      let text = "";
      for (let i = 0; i < 12; i++) {
        await new Promise((r) => setTimeout(r, 2500));
        text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
        if (text.includes("audit log")) break;
      }
      // Page banner + filters + an action chip from the seeded log.
      const chips = actionList.slice(0, 3).map((a) => a.toLowerCase());
      const hasActionChip = chips.some((c) => text.includes(c));
      check(
        "/audit renders banner, filters, and action chips",
        text.includes("audit log") &&
          (text.includes("all actions") || text.includes("action")) &&
          hasActionChip &&
          (text.includes("total entries") || text.includes("total")),
        text.slice(0, 80).replace(/\n/g, " "),
      );
    } finally {
      await browser.close().catch(() => {});
    }
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nAUDIT M32: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("AUDIT M32 failed:", e.message);
  process.exit(1);
});
