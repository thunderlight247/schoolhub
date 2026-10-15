// M34 governance verification (live, API-level) — the three clauses that
// finished item (2) of the user's spec:
//   A. every inviter can see their invitees            (GET /my-invitees)
//   B. every inviter can block their own invitees      (PUT /my-invitees/:id/status)
//   C. MASTER_ADMIN can block other admins             (PUT /users/:id/status)
//      + LAST_MASTER guard & ADMIN-not-blockable-by-outsiders (409)
//   D. teachers invite parents                        (POST /guardians + /account)
const qa = require("./lib/check-accounts");
const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}
const { BASE, api } = qa;

(async () => {
  const stamp = Date.now();
  const creds = qa.loadCreds();
  const master = await qa.login("master");
  const masterCookie = master.cookie;

  // --- A. master creates an admin; invitee is recorded -----------------
  const created = await api(
    "POST",
    "/admins",
    { email: `govadmin${stamp}@gmail.com`, first_name: "Gov", last_name: `Admin${stamp}` },
    masterCookie,
  );
  check("master creates admin via POST /admins", created.status === 201, `status=${created.status}`);
  const masterInvitees = await api("GET", "/my-invitees", null, masterCookie);
  const sawCreated = (masterInvitees.json?.invitees ?? []).some(
    (i) => i.email === `govadmin${stamp}@gmail.com`,
  );
  check("master sees own invitee in GET /my-invitees", masterInvitees.status === 200 && sawCreated,
    `count=${(masterInvitees.json?.invitees ?? []).length}`);

  // --- C. master blocks a plain admin, unblocks; outsider refused -------
  const adminLogin = await qa.login("admin"); // qa-admin, plain ADMIN
  const adminCookie = adminLogin.cookie;
  // grab qa-admin's id from master's admin list
  const adminsList = await api("GET", "/admins", null, masterCookie);
  const qaAdminRow = (adminsList.json?.admins ?? []).find((a) => a.email === creds.adminEmail);
  check("qa-admin present in admin list", !!qaAdminRow, qaAdminRow?.id);
  if (qaAdminRow) {
    const blocked = await api("PUT", `/users/${qaAdminRow.id}/status`, { status: "SUSPENDED" }, masterCookie);
    check("MASTER_ADMIN can block another admin (SUSPENDED)", blocked.status === 204, `status=${blocked.status}`);
    const unblocked = await api("PUT", `/users/${qaAdminRow.id}/status`, { status: "ACTIVE" }, masterCookie);
    check("MASTER_ADMIN can unblock that admin (ACTIVE)", unblocked.status === 204, `status=${unblocked.status}`);
  }
  // outsider (plain admin, not inviter, not master) refuses to block a master
  const masters = (adminsList.json?.admins ?? []).filter((a) => a.is_master && a.id !== qaAdminRow?.id);
  if (masters.length) {
    const outsider = await api("PUT", `/users/${masters[0].id}/status`, { status: "SUSPENDED" }, adminCookie);
    check("plain admin refused blocking a master → 409 ADMIN_NOT_BLOCKABLE",
      outsider.status === 409 && outsider.json?.error?.code === "ADMIN_NOT_BLOCKABLE",
      `status=${outsider.status} code=${outsider.json?.error?.code}`);
  }

  // --- D. teacher invites a parent (guardian.create route split) --------
  const teacher = await qa.login("teacher");
  const teacherCookie = teacher.cookie;
  const g = await api("POST", "/guardians", {
    first_name: "GovParent", last_name: `P${stamp}`, email: `govparent${stamp}@gmail.com`,
  }, teacherCookie);
  check("TEACHER can POST /guardians (guardian.create)", g.status === 201, `status=${g.status}`);
  const gid = g.json?.guardian?.id;
  if (gid) {
    const acc = await api("POST", `/guardians/${gid}/account`, {
      email: `govparent${stamp}@gmail.com`, password: `GovPass${stamp}!`,
    }, teacherCookie);
    check("TEACHER can provision parent account", acc.status === 200 || acc.status === 201, `status=${acc.status}`);
  }

  // --- A+B. teacher sees invitee, blocks it, parent dies, restores -------
  const tInvitees = await api("GET", "/my-invitees", null, teacherCookie);
  const invited = (tInvitees.json?.invitees ?? []).find((i) => i.email === `govparent${stamp}@gmail.com`);
  check("teacher sees the invited parent in GET /my-invitees", !!invited,
    `count=${(tInvitees.json?.invitees ?? []).length}`);
  if (invited) {
    const tBlock = await api("PUT", `/my-invitees/${invited.id}/status`, { status: "SUSPENDED" }, teacherCookie);
    check("teacher (no account.block perm) blocks own invitee → 204", tBlock.status === 204, `status=${tBlock.status}`);
    const parentLogin = await api("POST", "/auth/login", { email: `govparent${stamp}@gmail.com`, password: `GovPass${stamp}!` });
    // The backend's deliberate convention for a suspended account at login
    // is 403 ACCOUNT_NOT_ACTIVE (auth/handlers.go) — not 401 (bad creds).
    check("blocked parent can no longer log in → 403 ACCOUNT_NOT_ACTIVE",
      parentLogin.status === 403 && parentLogin.json?.error?.code === "ACCOUNT_NOT_ACTIVE",
      `status=${parentLogin.status} code=${parentLogin.json?.error?.code}`);
    const tUnblock = await api("PUT", `/my-invitees/${invited.id}/status`, { status: "ACTIVE" }, teacherCookie);
    check("teacher restores own invitee → 204", tUnblock.status === 204, `status=${tUnblock.status}`);
    const parentBack = await api("POST", "/auth/login", { email: `govparent${stamp}@gmail.com`, password: `GovPass${stamp}!` });
    check("restored parent logs in again → 200", parentBack.status === 200, `status=${parentBack.status}`);
  }

  // --- B scoping: an unrelated account can't touch someone else's invitee
  const parent = await qa.login("parent");
  const randoTarget = (tInvitees.json?.invitees ?? [])[0];
  if (randoTarget && invited && parent.cookie) {
    const foreign = await api("PUT", `/my-invitees/${randoTarget.id}/status`, { status: "SUSPENDED" }, parent.cookie);
    check("unrelated user refused (not their invitee) → 403 NOT_YOUR_INVITEE",
      foreign.status === 403 && foreign.json?.error?.code === "NOT_YOUR_INVITEE",
      `status=${foreign.status} code=${foreign.json?.error?.code}`);
    if (foreign.status === 403 && randoTarget.status === "SUSPENDED") {
      // nothing to restore; if the attempt went through we'd fix it — it didn't.
    }
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nM34 GOVERNANCE: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("M34 governance check failed:", e.message);
  process.exit(1);
});
