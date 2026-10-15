// M34 browser verification:
//   1. master admin opens /access and sees the "+ Add admin" composer button
//   2. the Add-admin modal opens with email/name fields (no password field)
//   3. plain admin does NOT see the Add-admin composer (master-only)
//   4. flagged (must_change_password) session hitting a protected page
//      renders the "Set your own password" gate instead of the page
//
// Clerk mode is enabled, so the login PAGE shows Clerk's SignIn — these
// checks sign in through the still-live classic API and seed the session
// cookie directly, exactly what the Clerk bridge would hand the browser.
//
// Run: node frontend/scripts/access-admin-browser-check.js
// (frontend dev server on :3000, backend on :8080)
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");
const BASE = qa.BASE;
const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

async function j(method, path, body, cookie) {
  const res = await fetch(BASE + path, {
    method,
    headers: {
      "content-type": "application/json",
      ...(cookie ? { cookie } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const setCookie = res.headers.get("set-cookie");
  const cookieName = setCookie
    ? setCookie.slice(0, setCookie.indexOf("="))
    : null;
  return {
    status: res.status,
    json: await res.json().catch(() => null),
    sessionCookie: setCookie
      ? setCookie.slice(0, setCookie.indexOf(";"))
      : null,
    cookieName,
  };
}

async function login(email, password) {
  const res = await j("POST", "/auth/login", { email, password });
  if (res.status !== 200) throw new Error(`${email} login ${res.status}`);
  return res.sessionCookie;
}

(async () => {
  if (!(await qa.requireClassicMode("access-admin-browser-check"))) process.exit(0);
  // M34 demo accounts removed — QA personas (gitignored creds file).
  const masterLogin = await qa.login("master");
  const cookieName = masterLogin.cookieName;
  const masterCookie = masterLogin.sessionCookie;
  check("master classic login works (Clerk mode still permits API login)", !!masterCookie, masterCookie ? masterCookie.slice(0, 12) + "…" : "missing");

  const plainLogin = await qa.login("admin");
  const adminCookie = plainLogin.sessionCookie;

  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
    defaultViewport: { width: 1440, height: 900 },
  });

  const textOf = async (page) =>
    (await page.evaluate(() => document.body.innerText)).toLowerCase();

  const openAs = async (cookieVal, path) => {
    const page = await browser.newPage();
    if (cookieVal && cookieName) {
      await page.setCookie({ name: cookieName, value: cookieVal.split("=")[1], domain: "localhost", path: "/" });
    }
    await page.goto("http://localhost:3000" + path, {
      waitUntil: "networkidle2",
      timeout: 60000,
    });
    // Turbopack cold-compiles — poll until the text settles.
    let text = "";
    for (let i = 0; i < 12; i++) {
      await new Promise((r) => setTimeout(r, 2500));
      text = await textOf(page);
      if (["access denied", "add admin", "set your own password", "staff directory", "access control"].some((k) => text.includes(k))) break;
    }
    return { page, text };
  };

  try {
    // 1+2: master sees the composer, and the modal has no password field.
    const master = await openAs(masterCookie, "/access");
    if (master.text.includes("set your own password")) {
      // master itself is flagged? Then flag a check separately — but with
      // the direct API cookie this shouldn't happen.
      await master.page.close();
      throw new Error("master unexpectedly hit the password gate");
    }
    check("master sees /access", master.text.includes("access control") || master.text.includes("admins"), master.text.slice(0, 80).replace(/\n/g, " "));
    check("master sees + Add admin composer button", master.text.includes("add admin"), master.text.slice(0, 120).replace(/\n/g, " "));
    if (master.text.includes("add admin")) {
      await master.page.evaluate(() => {
        const btn = [...document.querySelectorAll("button")].find((b) =>
          b.innerText.toLowerCase().includes("add admin"),
        );
        if (btn) btn.click();
      });
      await new Promise((r) => setTimeout(r, 800));
      const modalText = await textOf(master.page);
      const passwordInputs = await master.page.evaluate(
        () => document.querySelectorAll('input[type="password"]').length,
      );
      check(
        "Add-admin modal opens (email field present)",
        modalText.includes("email"),
        modalText.slice(0, 90).replace(/\n/g, " "),
      );
      check(
        "Add-admin modal has no password input (invite-only)",
        passwordInputs === 0,
        `password inputs: ${passwordInputs}`,
      );
    }
    await master.page.close();

    // 3: plain admin must NOT see the composer.
    const plain = await openAs(adminCookie, "/access");
    check(
      "plain admin does not see Add admin composer",
      !(await textOf(plain.page)).includes("add admin"),
    );
    await plain.page.close();

    // 4: flagged session hits the password gate on a protected page.
    // Flag the QA teacher account server-side (test scaffolding only),
    // then restore it afterwards.
    const { execSync } = require("child_process");
    const teacherLogin = await qa.login("teacher");
    execSync(
      `docker exec schoolhub-postgres-1 psql -U schoolhub -d schoolhub -c "update users set must_change_password = true where email = '${qa.loadCreds().teacherEmail}'"`,
    );
    const fresh = await j("POST", "/auth/login", { email: qa.loadCreds().teacherEmail, password: qa.loadCreds().teacherPassword });
    check("flagged account still classic-logins", fresh.status === 200);
    const gated = await openAs(fresh.sessionCookie, "/staff");
    check(
      "flagged session sees password gate instead of page",
      gated.text.includes("set your own password") &&
        !gated.text.includes("staff directory"),
      gated.text.slice(0, 80).replace(/\n/g, " "),
    );
    // Un-flag through the very endpoint being tested: change the password
    // in the browser? Simpler: reset the flag in DB afterwards.
    execSync(
      `docker exec schoolhub-postgres-1 psql -U schoolhub -d schoolhub -c "update users set must_change_password = false where email = '${qa.loadCreds().teacherEmail}'"`,
    );
    void teacherLogin;
    await gated.page.close();
  } finally {
    await browser.close().catch(() => {});
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nM34 ACCESS/GATE: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("M34 access/gate check failed:", e.message);
  process.exit(1);
});
