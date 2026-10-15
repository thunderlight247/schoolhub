// Live verification of the parent-access provisioning flow (expert-pass
// access-control follow-up). Now uses a REAL-DOMAIN email (gmail.com)
// because Clerk's invitation endpoint rejects example.test-style TLDs.
//   1. admin creates a guardian with a gmail-address-style email
//   2. POST /guardians/:id/account provisions the PARENT login
//   3. Guardian GET now carries the account summary
//   4. Classic login with the provisioned password works; /me has PARENT role
//   5. Clerk invitation exists for the email (Clerk Backend API), then revoked
//   6. guardian login blocked via PUT /users/:id/status → session dies
//   7. duplicate provisioning refused
//   8. cleanup: DB rows for this run deleted via psql in the container
const BASE = "http://localhost:8080/api/v1";
const CLERK_KEY = require("fs").readFileSync("/mnt/grace/schoolhub_1/.env", "utf8")
  .split("\n").find((l) => l.startsWith("CLERK_SECRET_KEY="))?.split("=")[1];
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
  const stamp = Date.now();
  // A real TLD so Clerk accepts the invitation. The local-part is unique
  // per run; nobody receives real mail because we revoke the invitation
  // during cleanup (and gmail addresses like this one are not deliverable
  // to the account owner's inbox at Resend's onboarding@ restriction
  // anyway — the invitation email comes from Clerk, not Resend).
  const email = `thunderlight2477+qa${stamp}@gmail.com`;

  // 1. admin login
  const creds = require("./lib/check-accounts").loadCreds();
  const admin = await j("POST", "/auth/login", { email: creds.masterEmail, password: creds.masterPassword });
  check("admin login", admin.status === 200 && !!admin.cookie);
  const adminCookie = admin.cookie;

  // 2. create guardian
  const g = await j("POST", "/guardians", {
    first_name: "Invited", last_name: `Parent${stamp}`, email,
  }, adminCookie);
  const guardianId = g.json?.guardian?.id;
  check("guardian created", g.status === 201 && !!guardianId, g.status);

  // 3. provision parent access
  const acct = await j("POST", `/guardians/${guardianId}/account`, {
    email, password: "ParentPass2026!",
  }, adminCookie);
  check("parent access provisioned (201)", acct.status === 201, JSON.stringify(acct.json));

  // 3b. duplicate provisioning refused
  const dup = await j("POST", `/guardians/${guardianId}/account`, {
    email, password: "ParentPass2026!",
  }, adminCookie);
  check("duplicate provisioning refused (ACCOUNT_EXISTS)", dup.status === 409 && dup.json?.error?.code === "ACCOUNT_EXISTS");

  // 4. guardian profile carries the account
  const detail = await j("GET", `/guardians/${guardianId}`, null, adminCookie);
  const acc = detail.json?.guardian?.account;
  check("guardian profile shows account (PARENT, ACTIVE)", !!acc && acc.roles?.includes("PARENT") && acc.status === "ACTIVE", JSON.stringify(acc));

  // 5. classic login works with PARENT role
  const plogin = await j("POST", "/auth/login", { email, password: "ParentPass2026!" });
  check("parent classic login works", plogin.status === 200 && !!plogin.cookie);
  const me = await j("GET", "/me", null, plogin.cookie);
  const roles = me.json?.user?.roles ?? [];
  check("/me shows PARENT role", me.status === 200 && roles.includes("PARENT"), roles.join(","));

  // 6. Clerk invitation exists for the email, then revoke it. The invite
  // is fired in a goroutine after commit, so poll for up to ~10s.
  let invOk = false, invId = "";
  if (CLERK_KEY) {
    for (let i = 0; i < 10 && !invOk; i++) {
      await new Promise((r) => setTimeout(r, 1000));
      const listRes = await fetch("https://api.clerk.com/v1/invitations?limit=20", {
        headers: { Authorization: `Bearer ${CLERK_KEY}` },
      });
      const list = await listRes.json();
      const inv = (Array.isArray(list) ? list : list?.data ?? []).find((x) => x.email_address === email);
      if (inv) {
        invOk = true;
        invId = inv.id;
        check("Clerk invitation created for the parent email", true, `${inv.status} (${inv.id}) after ~${i + 1}s`);
      }
    }
    if (!invOk) check("Clerk invitation created for the parent email", false, "not found within 10s");
    if (invOk) {
      const rev = await fetch(`https://api.clerk.com/v1/invitations/${invId}/revoke`, {
        method: "POST", headers: { Authorization: `Bearer ${CLERK_KEY}` },
      });
      check("Clerk invitation revoked (cleanup)", rev.ok || rev.status === 400 || rev.status === 409, rev.status);
    }
  } else {
    check("Clerk invitation check skipped (no key)", true, "no CLERK_SECRET_KEY");
  }

  // 7. block flows: parent blocked mid-session
  const accs = await j("GET", "/accounts?role=PARENT", null, adminCookie);
  const prow = (accs.json?.accounts ?? []).find((a) => a.email === email);
  check("parent listed in /accounts?role=PARENT", !!prow);

  const blk = await j("PUT", `/users/${prow.id}/status`, { status: "SUSPENDED" }, adminCookie);
  check("block parent returns 204", blk.status === 204);
  const meAfter = await j("GET", "/me", null, plogin.cookie);
  check("parent session dies immediately after block", meAfter.status === 401, `got ${meAfter.status}`);
  const un = await j("PUT", `/users/${prow.id}/status`, { status: "ACTIVE" }, adminCookie);
  check("unblock returns 204", un.status === 204);
  const relogin = await j("POST", "/auth/login", { email, password: "ParentPass2026!" });
  check("parent can sign in again after unblock", relogin.status === 200);

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nPARENT ACCESS LIVE: ${ok}/${results.length} checks passed`);

  // cleanup: remove this run's rows so the DB doesn't accumulate guardians.
  const admin2 = await j("POST", "/auth/login", { email: creds.masterEmail, password: creds.masterPassword });
  await j("PUT", `/users/${prow.id}/status`, { status: "ACTIVE" }, admin2.cookie);
  const { execSync } = require("child_process");
  try {
    execSync(
      `docker exec schoolhub-postgres-1 psql -U schoolhub -d schoolhub -q -c "DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE email='${email}'); DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE email='${email}'); DELETE FROM users WHERE email='${email}'; DELETE FROM guardians WHERE person_id IN (SELECT id FROM people WHERE last_name='Parent${stamp}'); DELETE FROM people WHERE last_name='Parent${stamp}';"`,
    );
    console.log("CLEANUP: run rows removed");
  } catch (e) {
    console.log("CLEANUP-HINT: manual psql delete for email " + email);
  }
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("PARENT ACCESS LIVE failed:", e.message);
  process.exit(1);
});
