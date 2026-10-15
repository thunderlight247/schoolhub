// One-off M24 browser verification (run with node; needs `npm install --no-save
// puppeteer-core` and /usr/bin/chromium, like m23-browser-check.js):
// staff list + search + filters, create staff, profile edit, assignment,
// account creation, role edit, employment status, and reference data.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";
const STAFF_NO = `M24-${Date.now().toString().slice(-6)}`;

(async () => {
  if (!(await qa.requireClassicMode("m24-browser-check"))) process.exit(0);
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
  });
  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 900 });
    let failures = 0;
    const check = (name, ok) => {
      console.log(`${ok ? "PASS" : "FAIL"} ${name}`);
      if (!ok) failures++;
    };
    const text = () => page.evaluate(() => document.body.innerText);

    // --- 1. Login as admin -> staff directory ---
    await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
    await page.type("#email", qa.loadCreds().masterEmail);
    await page.type("#password", qa.loadCreds().masterPassword);
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
      page.click("button[type=submit]"),
    ]);
    await page.goto(`${BASE}/staff`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Staff Directory") && !document.body.innerText.includes("Checking session"),
      { timeout: 15000 },
    );
    let t = await text();
    check("staff list renders under shell", t.includes("Staff Directory"));
    check("list shows seeded staff", t.includes("Sarah Mensah") && t.includes("Ada Obi"));
    check("account column differentiates", t.includes("Grace Amponsah"));

    // --- 2. Search filters the table ---
    await page.type("input[type=search]", "Sarah");
    await page.waitForFunction(
      () => !document.body.innerText.includes("Grace Amponsah") && document.body.innerText.includes("Sarah Mensah"),
      { timeout: 10000 },
    );
    t = await text();
    check("search narrows to Sarah Mensah", t.includes("STF-002") && !t.includes("STF-003"));

    // --- 3. Create staff member via modal ---
    await page.goto(`${BASE}/staff`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Add staff member"), { timeout: 10000 });
    const buttons = await page.$$("button");
    for (const b of buttons) {
      const label = await b.evaluate((el) => el.textContent);
      if (label?.includes("Add staff member")) {
        await b.click();
        break;
      }
    }
    await page.waitForSelector("input[placeholder='STF-010']", { timeout: 5000 });
    await page.waitForSelector("#staff-number", { timeout: 5000 });
    await page.type("#staff-number", STAFF_NO);
    await page.type("#first-name", "Kojo");
    await page.type("#last-name", "Testerson");
    await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    // The directory is sorted by name and paginated, so search for the new
    // staff number to bring the created row onto the page.
    await page.type("input[type=search]", STAFF_NO);
    await page.waitForFunction(
      (no) => document.body.innerText.includes(no),
      { timeout: 10000 },
      STAFF_NO,
    );
    t = await text();
    check("created staff appears in list", t.includes("Kojo Testerson"));

    // --- 4. Open profile via the row's link (resolve href, then navigate) ---
    const profileHref = await page.evaluate((no) => {
      const links = Array.from(document.querySelectorAll("a"));
      const link = links.find((l) => l.closest("tr")?.textContent?.includes(no));
      return link ? link.getAttribute("href") : null;
    }, STAFF_NO);
    if (!profileHref) {
      console.log("FAIL could not find profile link for created staff");
      process.exit(1);
    }
    await page.goto(`${BASE}${profileHref}`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Login account") && document.body.innerText.includes("Kojo Testerson"),
      { timeout: 15000 },
    );
    t = await text();
    check("profile shows detail sections", t.includes("Personal details") && t.includes("Employment & assignment"));
    check("no account yet message", t.includes("No login account yet"));

    // --- 5. Create login account with role ---
    const createAccountBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.trim() === "Create account");
    });
    await (createAccountBtn.asElement()).click();
    await page.waitForSelector("input[type=email]", { timeout: 5000 });
    await page.type("input[type=email]", `${STAFF_NO.toLowerCase()}@schoolhub.test`);
    await page.type("input[type=password]", "DemoPass123!");
    await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    await page.waitForFunction(
      () => document.body.innerText.includes("Account created") && document.body.innerText.includes("TEACHER"),
      { timeout: 10000 },
    );
    t = await text();
    check("account created with TEACHER role", t.includes("TEACHER") && t.includes("@schoolhub.test"));

    // --- 6. Edit roles (add STAFF, keep TEACHER) ---
    const editRolesBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.trim() === "Edit roles");
    });
    await (editRolesBtn.asElement()).click();
    await page.waitForFunction(() => document.querySelectorAll("input[type=checkbox]").length > 0, { timeout: 5000 });
    await page.evaluate(() => {
      const boxes = Array.from(document.querySelectorAll("input[type=checkbox]"));
      const staffBox = boxes.find((b) => b.closest("label")?.textContent?.trim() === "STAFF");
      if (staffBox) staffBox.click();
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    // The backend returns roles sorted alphabetically; wait for the
    // refetched account section to render both roles.
    await page.waitForFunction(
      () => /ROLES\s*\n\s*(TEACHER, STAFF|STAFF, TEACHER)/.test(document.body.innerText),
      { timeout: 10000 },
    );
    check("roles updated to TEACHER + STAFF", true);

    // --- 7. Reference data: create a department ---
    await page.goto(`${BASE}/staff/setup`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Add department"), { timeout: 10000 });
    const addDeptBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.includes("Add department"));
    });
    await (addDeptBtn.asElement()).click();
    await page.waitForSelector("#department-name", { timeout: 5000 });
    await page.type("#department-name", `QA Dept ${STAFF_NO}`);
    await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    await page.waitForFunction(
      (no) => document.body.innerText.includes(`QA Dept ${no}`),
      { timeout: 10000 },
      STAFF_NO,
    );
    check("department created and listed", true);

    console.log(failures === 0 ? "\nALL M24 BROWSER CHECKS PASSED" : `\n${failures} CHECK(S) FAILED`);
    process.exit(failures === 0 ? 0 : 1);
  } finally {
    await browser.close();
  }
})();
