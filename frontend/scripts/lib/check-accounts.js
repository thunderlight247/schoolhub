// Shared credential source for the milestone check scripts.
//
// M34 removed all seeded demo accounts (admin@/teacher@/ada@/parent@/student@
// …@schoolhub.test with the shared "ChangeMe123!" password) from the DB. The
// check scripts still need real role personas, so they now log in through
// THIS module, which reads the gitignored frontend/.check-credentials.json
// file. Nothing about these QA accounts is seed data: they are local script
// tooling, the passwords are strong random values, and the file never leaves
// the machine (see .gitignore).
//
// Personas live in frontend/.check-credentials.json:
//   qa-master   ADMIN+MASTER_ADMIN  — master-level flows (admins, access)
//   qa-admin    ADMIN               — plain-admin 403 paths
//   qa-teacher  TEACHER             — Ada Obi's staff row, with assignments
//   qa-parent   PARENT              — Abena Boateng's guardian
//   qa-student  STUDENT             — Ama Boateng (STU-2026-001)
//
// Scripts use e.g.:
//   const { login } = require("./lib/check-accounts");
//   const admin = await login("master");   // { cookie, res }
//   const teacher = await login("teacher");

const fs = require("fs");
const path = require("path");

const BASE = process.env.CHECK_API_BASE || "http://localhost:8080/api/v1";
const FRONTEND_ORIGIN = process.env.CHECK_FRONTEND || "http://localhost:3000";
const CREDS_PATH = path.join(__dirname, "..", "..", ".check-credentials.json");

/**
 * True when the dev server renders the classic email/password login form
 * (Clerk keys commented out). Browser legs that drive #email/#password can
 * only work in that mode — with Clerk keys active, /login is Clerk's hosted
 * sign-in and every form login 404s its selector. Detection greps the
 * served /login HTML for Clerk's dev-domain marker (remember.md workflow).
 */
async function isClassicMode() {
  try {
    const res = await fetch(FRONTEND_ORIGIN + "/login");
    const html = await res.text();
    return !html.includes("clerk.accounts.dev");
  } catch {
    return false; // frontend down → nothing can run either way
  }
}

/**
 * Gate for browser legs: returns true when the script may run its browser
 * section (classic mode), otherwise logs an explicit SKIPPED line and
 * returns false. Skipped legs are NOT counted as passed — they simply do
 * not run, mirroring the existing optional-`puppeteer` pattern.
 */
async function requireClassicMode(scriptName) {
  if (await isClassicMode()) return true;
  console.log(
    `SKIPPED — ${scriptName} browser leg requires classic mode; ` +
      `Clerk keys are active on ${FRONTEND_ORIGIN}. Comment out ` +
      `NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY and re-run (remember.md M34 workflow).`,
  );
  return false;
}

function loadCreds() {
  if (!fs.existsSync(CREDS_PATH)) {
    throw new Error(
      `.check-credentials.json not found at ${CREDS_PATH}. QA persona ` +
        `credentials are machine-local (never committed); ask the developer ` +
        `for the file or recreate the accounts per remember.md's M34 note.`,
    );
  }
  return JSON.parse(fs.readFileSync(CREDS_PATH, "utf8"));
}

/** POST an endpoint; returns { status, json, cookie } as the scripts expect. */
async function api(method, endpoint, body, cookie) {
  const res = await fetch(BASE + endpoint, {
    method,
    headers: {
      "content-type": "application/json",
      ...(cookie ? { cookie } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  const setCookie = res.headers.get("set-cookie");
  return {
    status: res.status,
    json: await res.json().catch(() => null),
    cookie: setCookie ? setCookie.slice(0, setCookie.indexOf(";")) : null,
  };
}

/**
 * Log in one of the QA personas by short name ("master", "admin", "teacher",
 * "parent", "student"). Throws on failure — checks should fail loudly.
 */
async function login(persona) {
  const creds = loadCreds();
  const emailKey = `${persona}Email`;
  const passwordKey = `${persona}Password`;
  if (!creds[emailKey]) throw new Error(`no ${persona} persona in .check-credentials.json`);
  const res = await api("POST", "/auth/login", {
    email: creds[emailKey],
    password: creds[passwordKey],
  });
  if (res.status !== 200 || !res.cookie) {
    throw new Error(
      `${persona} login failed (${res.status}). The local password may have ` +
        `rotated — fix frontend/.check-credentials.json.`,
    );
  }
  return res;
}

module.exports = { BASE, FRONTEND_ORIGIN, loadCreds, api, login, isClassicMode, requireClassicMode };
