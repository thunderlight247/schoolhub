// Live verification of the M30 announcements UI against the M13 backend.
// API-driven (the browser form flow is covered by the pattern in
// block-parent-browser-check.js); exercises every composer + lifecycle
// action the /announcements page drives:
//   login → create draft → filters see it → edit → publish → feed shows it
//   for a parent → archive → delete. Unauthenticated users are rejected by
//   the mounted auth middleware. (Teachers deliberately CAN create via API —
//   M13 granted them announcement.create for class-scoped posts; the M30
//   composer UI is admin-only by product decision, not by permission.)
// Run: node frontend/scripts/announcements-check.js (backend on :8080)
const puppeteer = require("puppeteer-core");

const BASE = "http://localhost:8080/api/v1";
const stamp = Date.now();
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
  // 1. Admin login (QA persona)
  const admin = await qa.login("master");
  check("admin login", admin.status === 200 && !!admin.cookie);
  const adminCookie = admin.cookie;

  // 2. Create a draft via the composer's payload shape
  const title = `M30 Composer Draft ${stamp}`;
  const created = await j("POST", "/announcements", {
    title,
    content: "M30 verification draft — edit, publish, archive, delete.",
    target_type: "ALL",
    priority: "HIGH",
  }, adminCookie);
  const ann = created.json?.data;
  check("composer creates DRAFT", created.status === 201 && ann?.status === "DRAFT", ann?.id);

  // 3. Directory lists the draft (status filter path the page uses)
  const listed = await j("GET", `/announcements?status=DRAFT&search=${encodeURIComponent(title)}`, null, adminCookie);
  check("directory lists draft with filters", listed.status === 200 && (listed.json?.data ?? []).some((a) => a.id === ann.id));

  // 4. Edit (title via PATCH payload shape)
  const edited = await j("PATCH", `/announcements/${ann.id}`, {
    title: `${title} (edited)`,
  }, adminCookie);
  check("edit announcement (PATCH)", edited.status === 200 && edited.json?.data?.title === `${title} (edited)`);

  // 5. Publish
  const published = await j("POST", `/announcements/${ann.id}/publish`, null, adminCookie);
  check("publish announcement", published.status === 200 && published.json?.data?.status === "PUBLISHED");

  // 6. Feed visibility for a parent (audience resolution, ALL target)
  // Provision the parent the same way the guardian page does.
  const g = await j("POST", "/guardians", {
    first_name: "AnnFeed", last_name: `Parent${stamp}`, email: `annparent${stamp}@schoolhub.test`,
  }, adminCookie);
  const guardianId = g.json?.guardian?.id;
  const acc = await j("POST", `/guardians/${guardianId}/account`, {
    email: `annparent${stamp}@schoolhub.test`, password: "ParentPass2026!",
  }, adminCookie);
  const parentLogin = acc.status === 200 || acc.status === 201
    ? await j("POST", "/auth/login", { email: `annparent${stamp}@schoolhub.test`, password: "ParentPass2026!" })
    : { status: 0 };
  check("parent provisioned + logs in", parentLogin.status === 200);
  const feed = parentLogin.cookie
    ? await j("GET", "/announcements?feed=true", null, parentLogin.cookie)
    : { status: 0, json: null };
  check(
    "parent feed shows published school-wide announcement",
    feed.status === 200 && (feed.json?.data ?? []).some((a) => a.id === ann.id),
    `${(feed.json?.data ?? []).length} feed items`,
  );

  // 7. Role targeting: create a PARENT-targeted announcement; digest any provides list
  const roleAnn = await j("POST", "/announcements", {
    title: `Parents-only brief ${stamp}`,
    content: "Targeted at PARENT role.",
    target_type: "ROLE",
    target_role: "PARENT",
  }, adminCookie);
  check("role-targeted draft created", roleAnn.status === 201, roleAnn.json?.data?.id);
  const rolePublished = await j("POST", `/announcements/${roleAnn.json?.data?.id}/publish`, null, adminCookie);
  check("role-targeted publish", rolePublished.status === 200);
  const feed2 = await j("GET", "/announcements?feed=true", null, parentLogin.cookie);
  check(
    "parent feed includes role-targeted announcement",
    (feed2.json?.data ?? []).some((a) => a.id === roleAnn.json?.data?.id),
  );

  // 8. Archive
  const archived = await j("POST", `/announcements/${ann.id}/archive`, null, adminCookie);
  check("archive announcement", archived.status === 200 && archived.json?.data?.status === "ARCHIVED");

  // 9. Delete (both cleanup + endpoint proof)
  const deleted = await j("DELETE", `/announcements/${ann.id}`, null, adminCookie);
  check("delete announcement (204)", deleted.status === 204);
  const gone = await j("GET", `/announcements/${ann.id}`, null, adminCookie);
  check("deleted announcement 404s", gone.status === 404);

  // 10. Unauthenticated requests are rejected by the auth middleware.
  const denied = await j("POST", "/announcements", {
    title: "Should be denied",
    content: "No session.",
    target_type: "ALL",
  });
  check("unauthenticated create rejected (401)", denied.status === 401, `got ${denied.status}`);

  // 11. Page renders in a real browser session (adminCookie injected natively
  // is unreliable — drive the classic login form like block-parent does).
  if (puppeteer && (await qa.requireClassicMode("announcements-check"))) {
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
      await page.goto("http://localhost:3000/announcements", { waitUntil: "networkidle2", timeout: 45000 });
      // Turbopack cold-compiles the route on first hit — wait for the page
      // text to settle instead of a fixed sleep.
      let text = "";
      for (let i = 0; i < 12; i++) {
        await new Promise((r) => setTimeout(r, 2500));
        text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
        if (text.includes("new announcement")) break;
      }
      const wantTitle = title.toLowerCase().split(" ").slice(0, 3).join(" ");
      check(
        "/announcements renders composer + directory",
        text.includes("announcements") && text.includes("new announcement") && text.includes(wantTitle),
        text.slice(0, 80).replace(/\n/g, " "),
      );
    } finally {
      await browser.close();
    }
  }

  // Cleanup: parent rows + role-targeted announcement
  const cleanupAnn = await j("DELETE", `/announcements/${roleAnn.json?.data?.id}`, null, adminCookie);
  try {
    const { execSync } = require("child_process");
    execSync(
      `docker exec schoolhub-postgres-1 psql -U schoolhub -d schoolhub -q -c "DELETE FROM sessions WHERE user_id IN (SELECT u.id FROM users u JOIN people p ON p.id=u.person_id WHERE p.last_name='Parent${stamp}'); DELETE FROM user_roles WHERE user_id IN (SELECT u.id FROM users u JOIN people p ON p.id=u.person_id WHERE p.last_name='Parent${stamp}'); DELETE FROM users WHERE person_id IN (SELECT id FROM people WHERE last_name='Parent${stamp}'); DELETE FROM guardians WHERE person_id IN (SELECT id FROM people WHERE last_name='Parent${stamp}'); DELETE FROM people WHERE last_name='Parent${stamp}';"`,
    );
    console.log(`CLEANUP: parent rows removed (ann delete ${cleanupAnn.status})`);
  } catch (e) {
    console.log("CLEANUP-HINT: manual delete of AnnFeed Parent" + stamp);
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nANNOUNCEMENTS M30: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("ANNOUNCEMENTS M30 failed:", e.message);
  process.exit(1);
});
