// M34 LAST_MASTER guard — staged live proof of the demoted-inviter edge
// case the guard exists for:
//
//   1. qa-master invites govlm (inviter recorded)
//   2. govlm promoted to MASTER_ADMIN          (SQL, test scaffolding)
//   3. the two real masters get SUSPENDED      (SQL → only ACTIVE master = govlm)
//   4. qa-master's admin roles revoked         (SQL → demoted, but still the INVITER)
//   5. qa-master (roleless, ACTIVE) tries to suspend govlm via the
//      inviter-scoped route → expect 409 LAST_MASTER (not 204!)
//
// Everything is restored in `finally` (roles back, masters ACTIVE, test
// admin removed) so a mid-run failure never leaves the dev DB locked out.
const qa = require("./lib/check-accounts");
const { execSync } = require("child_process");
const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}
const psql = (sql) =>
  execSync(
    `docker exec schoolhub-postgres-1 psql -U schoolhub -d schoolhub -tAc ${JSON.stringify(sql)}`,
    { encoding: "utf8" },
  ).trim();

const stamp = Date.now();
const lmEmail = `govlm${stamp}@gmail.com`;
let lmId = null;
let ok = false;

(async () => {
  try {
    const master = await qa.login("master");

    // 1. Inviter relationship.
    const created = await qa.api("POST", "/admins", {
      email: lmEmail, first_name: "GovLM", last_name: `LM${stamp}`,
    }, master.cookie);
    check("setup: master creates test admin", created.status === 201, `status=${created.status}`);
    lmId = created.json?.admin?.id;
    if (!lmId) throw new Error("no test admin id — cannot stage");

    // 2–4. SQL scaffolding (test-only, restored in finally).
    psql(`insert into user_roles (user_id, role_id) select u.id, r.id from users u, roles r where u.email='${lmEmail}' and r.name='MASTER_ADMIN' on conflict do nothing`);
    psql(`update users set status='SUSPENDED' where email in ('thunderlight2477@gmail.com','thunderlight2477+admin@gmail.com')`);
    psql(`delete from user_roles where user_id = (select id from users where email='qa-master@schoolhub.test')`);

    // Sanity: exactly one ACTIVE master remains (the target).
    const actives = psql(
      `select count(distinct u.id) from users u join user_roles ur on ur.user_id=u.id join roles r on r.id=ur.role_id where r.name='MASTER_ADMIN' and u.status='ACTIVE'`,
    );
    check("setup: only one ACTIVE master remains (the target)", actives === "1", `actives=${actives}`);

    // 5. The demoted inviter attacks the last master.
    const qaLogin = await qa.login("master"); // same creds; roles now gone
    const blocked = await qa.api("PUT", `/my-invitees/${lmId}/status`, { status: "SUSPENDED" }, qaLogin.cookie);
    check("demoted inviter suspending the LAST master → 409 LAST_MASTER",
      blocked.status === 409 && blocked.json?.error?.code === "LAST_MASTER",
      `status=${blocked.status} code=${blocked.json?.error?.code}`);

    // The target must still be ACTIVE (nothing partially applied).
    const stillActive = psql(`select status from users where email='${lmEmail}'`);
    check("target master is untouched (still ACTIVE)", stillActive === "ACTIVE", `status=${stillActive}`);

    ok = true;
  } finally {
    // --- restore everything, unconditionally ---
    try {
      if (lmId) psql(`delete from users where id='${lmId}'`); // cascade: roles, sessions
      psql(`update users set status='ACTIVE' where email in ('thunderlight2477@gmail.com','thunderlight2477+admin@gmail.com')`);
      psql(
        `insert into user_roles (user_id, role_id) select u.id, r.id from users u, roles r where u.email='qa-master@schoolhub.test' and r.name in ('ADMIN','MASTER_ADMIN') on conflict do nothing`,
      );
      const back = psql(
        `select count(distinct u.id) from users u join user_roles ur on ur.user_id=u.id join roles r on r.id=ur.role_id where r.name='MASTER_ADMIN' and u.status='ACTIVE'`,
      );
      console.log(`restored: ACTIVE masters=${back}, test admin removed=${psql(`select count(*) from users where email='${lmEmail}'`) === "0"}`);
    } catch (e) {
      console.error("RESTORE FAILED — check the dev DB manually:", e.message);
      ok = false;
    }
  }

  const p = results.filter((r) => r.ok).length;
  console.log(`\nM34 LAST_MASTER: ${p}/${results.length} checks passed`);
  process.exit(p === results.length ? 0 : 1);
})().catch((e) => {
  console.error("M34 last-master check failed:", e.message);
  process.exit(1);
});
