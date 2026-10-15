// Live verification for the expert-pass access-control layer. Uses the
// backend directly with real sessions minted by seeding a throwaway
// teacher + parent, blocking them, and asserting the session dies
// mid-conversation. Master-admin checks use the seeded admin account.
//
// Run: node frontend/scripts/access-control-check.js (backend on 8080)
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
    credentials: "omit",
    body: body ? JSON.stringify(body) : undefined,
  });
  // Collect Set-Cookie so login can hold the session like a browser.
  const setCookie = res.headers.getSetCookie?.() ?? [];
  const sc = setCookie.find((c) => c.startsWith("schoolhub_session="));
  const sessionCookie = sc ? sc.split(";")[0] : undefined;
  let json = null;
  try { json = await res.json(); } catch {}
  return { status: res.status, json, cookie: cookie ?? sessionCookie };
}

(async () => {
  const stamp = Date.now();
  const uniq = `acct${stamp}`;

  // 1. Seed admin login (classic flow, password from the seed)
  const adminLogin = await j("POST", "/auth/login", {
    email: "qa-master@schoolhub.test",
    password: require("./lib/check-accounts").loadCreds().masterPassword,
  });
  check("admin classic login works", adminLogin.status === 200 && !!adminLogin.cookie);
  const adminCookie = adminLogin.cookie;

  // 2. /admins shows 2 admins and IsMaster flags
  const admins = await j("GET", "/admins", null, adminCookie);
  const adminRows = admins.json?.admins ?? [];
  check(
    "GET /admins returns admin list",
    admins.status === 200 && adminRows.length >= 2,
    `${adminRows.length} admins`
  );
  const masterCount = adminRows.filter((a) => a.is_master).length;
  check("master admin flagged in /admins", masterCount >= 1, `${masterCount} master`);

  // 3. Seed a fresh teacher through the normal staff flow (the same
  // provisioning an admin performs from the dashboard). Parent accounts
  // are provisioned the same way via guardians flow + cmd/seed in real
  // runs; this check exercises the blocking mechanics teachers-side.
  const staffRes = await j("POST", "/staff", {
    staff_number: `STF-${stamp}`,
    first_name: "Acc",
    last_name: "Teacher",
    email: `${uniq}.s@example.test`,
  }, adminCookie);
  const staffId = staffRes.json?.staff?.id;
  const acctRes = await j("POST", `/staff/${staffId}/account`, {
    email: `${uniq}.s@example.test`,
    password: "TeacherPass2026!",
    roles: ["TEACHER"],
  }, adminCookie);
  check("teacher account created", acctRes.status === 201 || acctRes.status === 200);

  // Creating the account provisions users row; teacher logs in classically.
  const teacherCookie = (await j("POST", "/auth/login", {
    email: `${uniq}.s@example.test`,
    password: "TeacherPass2026!",
  })).cookie;
  check("teacher can log in after account creation", !!teacherCookie);

  const meBefore = await j("GET", "/me", null, teacherCookie);
  check("teacher session live pre-block", meBefore.status === 200);

  // 4. Find the teacher user id via /accounts?role=TEACHER
  const tlist = await j("GET", "/accounts?role=TEACHER", null, adminCookie);
  const tRows = tlist.json?.accounts ?? [];
  const tRow = tRows.find((r) => r.email === `${uniq}.s@example.test`);
  check("teacher listed in /accounts?role=TEACHER", !!tRow);

  // 5. Block the teacher mid-session
  const blockRes = await j("PUT", `/users/${tRow.id}/status`, { status: "SUSPENDED" }, adminCookie);
  check("block endpoint returns 204", blockRes.status === 204);

  const meAfter = await j("GET", "/me", null, teacherCookie);
  check("teacher session dies immediately after block", meAfter.status === 401, `got ${meAfter.status}`);

  // 6. Unblock, login works again
  const unblockRes = await j("PUT", `/users/${tRow.id}/status`, { status: "ACTIVE" }, adminCookie);
  check("unblock endpoint returns 204", unblockRes.status === 204);
  const relogin = await j("POST", "/auth/login", {
    email: `${uniq}.s@example.test`,
    password: "TeacherPass2026!",
  });
  check("teacher can log in again after unblock", relogin.status === 200);

  // 7. Admin self-blocking and master self-demotion are refused
  const meAdmin = await j("GET", "/me", null, adminCookie);
  const adminId = meAdmin.json?.user?.id;
  const selfBlock = await j("PUT", `/users/${adminId}/status`, { status: "SUSPENDED" }, adminCookie);
  check("admin cannot block self (SELF_BLOCK)", selfBlock.status === 409 && selfBlock.json?.error?.code === "SELF_BLOCK");

  const selfDemote = await j("PUT", `/users/${adminId}/admin-role`, {}, adminCookie);
  check("master cannot revoke own admin role (SELF_DEMOTION)", selfDemote.status === 409 && selfDemote.json?.error?.code === "SELF_DEMOTION");

  // 8. Blocking an admin is refused (ADMIN_NOT_BLOCKABLE)
  const otherAdmin = adminRows.find((a) => a.id !== adminId);
  if (otherAdmin) {
    const blockAdmin = await j("PUT", `/users/${otherAdmin.id}/status`, { status: "SUSPENDED" }, adminCookie);
    check("blocking an admin is refused (ADMIN_NOT_BLOCKABLE)", blockAdmin.status === 409 && blockAdmin.json?.error?.code === "ADMIN_NOT_BLOCKABLE");
  }

  // 9. Non-master (plain admin seeded?) — /accounts?role=PARENT works for admin
  const plist = await j("GET", "/accounts?role=PARENT", null, adminCookie);
  check("GET /accounts?role=PARENT works", plist.status === 200, `${(plist.json?.accounts ?? []).length} parents`);

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nACCESS CONTROL LIVE: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("ACCESS CONTROL LIVE failed:", e.message);
  process.exit(1);
});
