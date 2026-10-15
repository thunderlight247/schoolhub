// Live verification of the M31 Reports Center UI against the M18 backend.
// API-driven, mirroring announcements-check.js (the browser form flow is
// covered by the pattern in block-parent-browser-check.js; /reports is a
// read-only page with no mutations to verify).
//
// Covers all four M18 endpoints the /reports page drives:
//   1. GET /reports/enrollment                     → school-wide enrollment
//   2. GET /reports/attendance/classes/:id         → per-student attendance
//   3. GET /reports/performance/classes/:id        → assessment performance
//   4. GET /reports/students/:id/report-card       → terminal report card
// plus optional-filter, invalid-input, and error paths (404 class/student,
// 401 unauthenticated, 400 bad filters).
//
// Run: node frontend/scripts/reports-check.js (backend on :8080)
const puppeteer = require("puppeteer-core");

const BASE = "http://localhost:8080/api/v1";
// Seeded IDs (demo-seed): Form 1A class, Term 1, Ama Boateng STU-2026-001.
const CLASS_FORM1A = "55555555-5555-5555-5555-555555555555";
const TERM_1 = "22222222-2222-2222-2222-222222222222";
const STUDENT_AMA = "bbbb0001-0000-0000-0000-000000000001";

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

  // 2. Enrollment report (school-wide pull the Enrollment tab does on load)
  const enr = await j("GET", "/reports/enrollment", null, adminCookie);
  const enrollment = enr.json?.data;
  check(
    "enrollment report (200, structure)",
    enr.status === 200 &&
      typeof enrollment?.total_classes === "number" &&
      typeof enrollment?.total_enrolled === "number" &&
      Array.isArray(enrollment?.classes),
    `classes=${enrollment?.total_classes} enrolled=${enrollment?.total_enrolled}`,
  );

  const form1a = (enrollment?.classes ?? []).find((c) => c.class_id === CLASS_FORM1A);
  check(
    "enrollment includes Form 1A with utilization",
    !!form1a &&
      typeof form1a.enrolled_count === "number" &&
      (form1a.utilization_rate == null || typeof form1a.utilization_rate === "number"),
    form1a ? `enrolled=${form1a.enrolled_count} util=${form1a.utilization_rate}` : "missing",
  );

  // 3. Class attendance report (Attendance tab: class + optional term/date filters)
  const att = await j("GET", `/reports/attendance/classes/${CLASS_FORM1A}?term_id=${TERM_1}`, null, adminCookie);
  const attendance = att.json?.data;
  check(
    "class attendance report (200, term filter)",
    att.status === 200 &&
      attendance?.class_id === CLASS_FORM1A &&
      attendance?.term_id === TERM_1 &&
      typeof attendance?.total_sessions === "number" &&
      Array.isArray(attendance?.students) &&
      attendance.students.length > 0 &&
      typeof attendance.students[0].attendance_rate === "number",
    `sessions=${attendance?.total_sessions} students=${attendance?.total_students} avg=${attendance?.average_attendance_rate}`,
  );

  // 4. Class performance report (Performance tab: class + optional term/subject)
  const perf = await j("GET", `/reports/performance/classes/${CLASS_FORM1A}?term_id=${TERM_1}`, null, adminCookie);
  const performance = perf.json?.data;
  check(
    "class performance report (200, term filter)",
    perf.status === 200 &&
      performance?.class_id === CLASS_FORM1A &&
      Array.isArray(performance?.assessments) &&
      performance.assessments.length > 0 &&
      typeof performance.assessments[0].average_score === "number",
    `assessments=${performance?.assessments?.length} subject=${performance?.subject_name ?? "all"}`,
  );

  // 5. Student report card (Report Card tab: student + required term_id)
  const rc = await j("GET", `/reports/students/${STUDENT_AMA}/report-card?term_id=${TERM_1}`, null, adminCookie);
  const card = rc.json?.data;
  check(
    "student report card (200, header + subjects + attendance)",
    rc.status === 200 &&
      card?.student_id === STUDENT_AMA &&
      card?.term_id === TERM_1 &&
      typeof card?.student_number === "string" &&
      card?.attendance && typeof card.attendance.attendance_rate === "number" &&
      Array.isArray(card.subjects) && card.subjects.length > 0 &&
      typeof card.subjects[0].grade === "string",
    `${card?.first_name ?? ""} ${card?.last_name ?? ""} ${card?.student_number ?? ""} subjects=${card?.subjects?.length} overall=${card?.overall_grade ?? "N/A"}`,
  );

  // 6. Missing class 404s
  const badClass = "99999999-9999-9999-9999-999999999999";
  const att404 = await j("GET", `/reports/attendance/classes/${badClass}`, null, adminCookie);
  check("attendance report unknown class → 404", att404.status === 404);
  const perf404 = await j("GET", `/reports/performance/classes/${badClass}`, null, adminCookie);
  check("performance report unknown class → 404", perf404.status === 404);

  // 7. Unknown student 404s (as nonexistent student id, valid term)
  const badStudent = "88888888-8888-8888-8888-888888888888";
  const rc404 = await j("GET", `/reports/students/${badStudent}/report-card?term_id=${TERM_1}`, null, adminCookie);
  check("report card unknown student → 404", rc404.status === 404);

  // 8. Report card without term_id → 400
  const rcNoTerm = await j("GET", `/reports/students/${STUDENT_AMA}/report-card`, null, adminCookie);
  check("report card missing term_id → 400", rcNoTerm.status === 400);

  // 9. Bad UUID inputs → 400 (class id + term filter)
  const attBad = await j("GET", "/reports/attendance/classes/not-a-uuid", null, adminCookie);
  check("attendance report bad class uuid → 400", attBad.status === 400);
  const rcBadTerm = await j("GET", `/reports/students/${STUDENT_AMA}/report-card?term_id=banana`, null, adminCookie);
  check("report card bad term uuid → 400", rcBadTerm.status === 400);

  // 10. Invalid date range → 400 (attendance with start after end)
  const badRange = await j(
    "GET",
    `/reports/attendance/classes/${CLASS_FORM1A}?start_date=2026-12-31&end_date=2026-01-01`,
    null,
    adminCookie,
  );
  check("attendance invalid date range → 400", badRange.status === 400);

  // 11. Unauthenticated requests are rejected.
  const unauth = await j("GET", "/reports/enrollment");
  check("unauthenticated report request → 401", unauth.status === 401, `got ${unauth.status}`);

  // 12. Page renders in a real browser session (classic-mode form login).
  // Requires the dev server to be in classic mode (Clerk keys out of
  // .env.local, restarted with `setsid nohup npm run dev` per remember.md) —
  // Clerk Protect headlessly flags automation and the classic-mode form
  // (#email / #password) only exists without Clerk keys. NOTE: never pkill
  // processes by name from a script on this box — the pattern can match the
  // invoking shell itself.
  if (puppeteer && (await qa.requireClassicMode("reports-check"))) {
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
      await page.goto("http://localhost:3000/reports", { waitUntil: "networkidle2", timeout: 45000 });
      // Turbopack cold-compiles the route on first hit — wait for the page
      // text to settle instead of a fixed sleep.
      let text = "";
      for (let i = 0; i < 12; i++) {
        await new Promise((r) => setTimeout(r, 2500));
        text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
        if (text.includes("report card")) break;
      }
      // Page banner, tab labels, and at least the enrollment table content
      // (Form 1A is a seed class and must appear in the Class Breakdown list).
      check(
        "/reports renders tabs + enrollment breakdown",
        text.includes("reports center") &&
          text.includes("report card") &&
          text.includes("form 1a") &&
          text.includes("total enrolled"),
        text.slice(0, 80).replace(/\n/g, " "),
      );
    } finally {
      await browser.close();
    }
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nREPORTS M31: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("REPORTS M31 failed:", e.message);
  process.exit(1);
});
