// M34 governance UI verification (classic mode, dev server on :3000):
//   master sees: invitees section + admin composer + account directory
//   teacher sees: ONLY the invitees section (no composer/directory) plus
//                 the Guardians nav item (new teacher power)
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");
const results = [];
function check(name, ok, extra = "") {
  results.push({ name, ok });
  console.log(`${ok ? "PASS" : "FAIL"} — ${name}${extra ? ` (${extra})` : ""}`);
}

(async () => {
  if (!(await qa.requireClassicMode("m34-governance-ui-check"))) process.exit(0);
  const creds = qa.loadCreds();
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
    defaultViewport: { width: 1440, height: 900 },
  });

  const runAs = async (email, password, path) => {
    const page = await browser.newPage();
    await page.goto("http://localhost:3000/login", { waitUntil: "networkidle2", timeout: 60000 });
    await new Promise((r) => setTimeout(r, 1200));
    await page.type("#email", email);
    await page.type("#password", password);
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle2", timeout: 45000 }).catch(() => {}),
      page.click("button[type=submit]"),
    ]);
    await page.goto("http://localhost:3000" + path, { waitUntil: "networkidle2", timeout: 60000 });
    let text = "";
    for (let i = 0; i < 12; i++) {
      await new Promise((r) => setTimeout(r, 2500));
      text = (await page.evaluate(() => document.body.innerText)).toLowerCase();
      if (text.includes("people you invited") || text.includes("access denied")) break;
    }
    return { page, text };
  };

  try {
    // 1. Master: invitees + admin composer + account directory.
    const master = await runAs(creds.masterEmail, creds.masterPassword, "/access");
    check("master: sees People you invited section", master.text.includes("people you invited"),
      master.text.slice(0, 60).replace(/\n/g, " "));
    check("master: sees + Add admin composer", master.text.includes("add admin"));
    check("master: sees role tabs (account directory)", master.text.includes("parents") && master.text.includes("teachers"));
    const navMaster = await master.page.evaluate(() => document.body.innerText);
    check("master: Guardians nav present", navMaster.toLowerCase().includes("guardians"));
    await master.page.close();

    // 2. Teacher: ONLY the invitees section (no admin powers), Guardians nav.
    const teacher = await runAs(creds.teacherEmail, creds.teacherPassword, "/access");
    check("teacher: sees People you invited section", teacher.text.includes("people you invited"),
      teacher.text.slice(0, 60).replace(/\n/g, " "));
    check("teacher: does NOT see + Add admin composer", !teacher.text.includes("add admin"));
    check("teacher: does NOT see role-tab directory", !(teacher.text.includes("parents") && teacher.text.includes("teachers")),
      teacher.text.slice(0, 80).replace(/\n/g, " "));
    const navTeacher = await teacher.page.evaluate(() => document.body.innerText);
    check("teacher: Guardians nav present (new power)", navTeacher.toLowerCase().includes("guardians"));
    check("teacher: invitee row visible (QA parent)", teacher.text.includes("qa-parent@schoolhub.test") || teacher.text.includes("govparent"),
      teacher.text.slice(0, 120).replace(/\n/g, " "));
    await teacher.page.close();
  } finally {
    await browser.close().catch(() => {});
  }

  const ok = results.filter((r) => r.ok).length;
  console.log(`\nM34 GOVERNANCE UI: ${ok}/${results.length} checks passed`);
  process.exit(ok === results.length ? 0 : 1);
})().catch((e) => {
  console.error("M34 governance UI check failed:", e.message);
  process.exit(1);
});
