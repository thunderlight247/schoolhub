// One-off M26 browser verification (run with node; needs `npm install --no-save
// puppeteer-core` and /usr/bin/chromium): academic structure tabs — create
// year/term/programme/subject/class, set-current flags, and class curriculum
// assign/unassign.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";
const STAMP = Date.now().toString().slice(-5);
const YEAR_NAME = `QA ${STAMP}/28`;

async function login(page) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
  await page.type("#email", qa.loadCreds().masterEmail);
  await page.type("#password", qa.loadCreds().masterPassword);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
}

async function clickButtonWithText(page, text) {
  const handle = await page.evaluateHandle((t) => {
    const btns = Array.from(document.querySelectorAll("button"));
    return btns.find((b) => b.textContent?.trim().includes(t));
  }, text);
  const el = handle.asElement();
  if (!el) throw new Error(`button not found: ${text}`);
  await el.click();
  return handle;
}

async function submitLastForm(page) {
  await page.evaluate(() => {
    const forms = document.querySelectorAll("form");
    forms[forms.length - 1].requestSubmit();
  });
}

// React overrides input/select value trackers — plain assignment + a bubbling
// event does not update controlled state. Use the prototype setters.
async function setReactValue(page, selector, value, index = 0) {
  await page.evaluate(
    (sel, val, idx) => {
      const els = Array.from(document.querySelectorAll("form " + sel));
      const el = els[idx];
      const proto = el instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
      setter.call(el, val);
      el.dispatchEvent(new Event("input", { bubbles: true }));
      el.dispatchEvent(new Event("change", { bubbles: true }));
    },
    selector,
    value,
    index,
  );
}

(async () => {
  if (!(await qa.requireClassicMode("m26-browser-check"))) process.exit(0);
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

    await login(page);
    await page.goto(`${BASE}/academics`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Academic Structure") && !document.body.innerText.includes("Checking session"),
      { timeout: 15000 },
    );
    let t = await text();
    check("academics page renders with tabs", t.includes("Academic Years") && t.includes("Class Curriculum"));
    check("current year flagged", t.includes("2026/2027") && t.includes("Current"));

    // --- 1. Create academic year ---
    await clickButtonWithText(page, "Add academic year");
    await page.waitForSelector("#year-name", { timeout: 5000 });
    await page.type("#year-name", YEAR_NAME);
    await setReactValue(page, "input[type=date]", "2027-09-01", 0);
    await setReactValue(page, "input[type=date]", "2028-07-31", 1);
    await submitLastForm(page);
    await page.waitForFunction(
      (n) => document.body.innerText.includes(n),
      { timeout: 10000 },
      YEAR_NAME,
    );
    check("academic year created and listed", (await text()).includes(YEAR_NAME));

    // --- 2. Set the new year current, verify the flag moves ---
    await clickButtonWithText(page, "Set current");
    await page.waitForFunction(
      () => document.body.innerText.includes("Current academic year updated."),
      { timeout: 10000 },
    );
    check("current year moved", true);
    // restore 2026/2027 as current for the demo dataset (re-scan at click
    // time: the table re-renders after the flag move)
    await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find((r) => r.textContent?.includes("2026/2027"));
      row?.querySelector("button[title='Set as current year']")?.click();
    });
    await page.waitForFunction(
      () =>
        /2026\/2027\s*\n\s*\w/.test(document.body.innerText) &&
        document.body.innerText.includes("Current academic year updated."),
      { timeout: 10000 },
    );
    // verify the flag really moved back
    const restored = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find((r) => r.textContent?.includes("2026/2027"));
      return row?.textContent?.includes("Current") ?? false;
    });
    if (!restored) throw new Error("2026/2027 is not current after restore");

    // --- 3. Create a term in the new year ---
    await page.evaluate(() => {
      const tabs = Array.from(document.querySelectorAll("button"));
      tabs.find((b) => b.textContent?.trim() === "Terms")?.click();
    });
    await page.waitForFunction(() => document.body.innerText.includes("Add term"), { timeout: 5000 });
    await clickButtonWithText(page, "Add term");
    await page.waitForSelector("form select", { timeout: 5000 });
    // pick the new year in the select (it is the last option)
    await page.evaluate((yearName) => {
      const select = document.querySelector("form select");
      const opt = Array.from(select.options).find((o) => o.textContent === yearName);
      window.__yearOptionValue = opt ? opt.value : "";
    }, YEAR_NAME);
    const yearOpt = await page.evaluate(() => window.__yearOptionValue);
    await setReactValue(page, "select", yearOpt, 0);
    await page.waitForSelector("#term-name", { timeout: 5000 });
    await page.type("#term-name", `Term 2 — QA ${STAMP}`);
    await setReactValue(page, "input[type=date]", "2028-01-10", 0);
    await setReactValue(page, "input[type=date]", "2028-04-01", 1);
    await submitLastForm(page);
    await page.waitForFunction(
      (n) => document.body.innerText.includes(n),
      { timeout: 10000 },
      `Term 2 — QA ${STAMP}`,
    );
    check("term created in selected year", true);

    // --- 4. Create programme + subject ---
    await page.evaluate(() => {
      const tabs = Array.from(document.querySelectorAll("button"));
      tabs.find((b) => b.textContent?.trim() === "Programmes")?.click();
    });
    await page.waitForFunction(() => document.body.innerText.includes("Add programme"), { timeout: 5000 });
    await clickButtonWithText(page, "Add programme");
    await page.waitForSelector("#programme-code", { timeout: 5000 });
    await page.type("#programme-code", `QA${STAMP.slice(-3)}`);
    await page.type("#programme-name", `Quality Arts ${STAMP.slice(-3)}`);
    await submitLastForm(page);
    await page.waitForFunction(
      (c) => document.body.innerText.includes(c),
      { timeout: 10000 },
      `QA${STAMP.slice(-3)}`,
    );
    check("programme created", true);

    await page.evaluate(() => {
      const tabs = Array.from(document.querySelectorAll("button"));
      tabs.find((b) => b.textContent?.trim() === "Subjects")?.click();
    });
    await page.waitForFunction(() => document.body.innerText.includes("Add subject"), { timeout: 5000 });
    await clickButtonWithText(page, "Add subject");
    await page.waitForSelector("#subject-code", { timeout: 5000 });
    await page.type("#subject-code", `SUB${STAMP.slice(-3)}`);
    await page.type("#subject-name", `Quality Studies ${STAMP.slice(-3)}`);
    await submitLastForm(page);
    await page.waitForFunction(
      (c) => document.body.innerText.includes(c),
      { timeout: 10000 },
      `SUB${STAMP.slice(-3)}`,
    );
    check("subject created", true);

    // --- 5. Curriculum: assign + unassign on Form 1A ---
    await page.evaluate(() => {
      const tabs = Array.from(document.querySelectorAll("button"));
      tabs.find((b) => b.textContent?.trim() === "Class Curriculum")?.click();
    });
    await page.waitForFunction(() => document.body.innerText.includes("Assign a subject…"), { timeout: 10000 });
    t = await text();
    check("curriculum shows Form 1A assigned subjects", t.includes("Mathematics") || t.includes("English"));

    // assign the QA subject
    await page.evaluate((code) => {
      const selects = Array.from(document.querySelectorAll("select"));
      const sel = selects.find((s) => s.textContent?.includes("Assign a subject…"));
      const opt = Array.from(sel.options).find((o) => o.textContent?.includes(code));
      if (opt) {
        sel.value = opt.value;
        sel.dispatchEvent(new Event("change", { bubbles: true }));
      }
    }, `SUB${STAMP.slice(-3)}`);
    await clickButtonWithText(page, "Assign");
    await page.waitForFunction(
      () => document.body.innerText.includes("Subject assigned to class."),
      { timeout: 10000 },
    );
    check("subject assigned to class", (await text()).includes(`SUB${STAMP.slice(-3)}`));

    // unassign it again (remove button in the row)
    await page.evaluate((code) => {
      // the remove control is the action button in the row containing the code
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find((r) => r.textContent?.includes(code));
      row?.querySelector("button")?.click();
    }, `SUB${STAMP.slice(-3)}`);
    await page.waitForFunction(
      () => document.body.innerText.includes("Subject removed from class."),
      { timeout: 10000 },
    );
    check("subject unassigned", true);

    console.log(failures === 0 ? "\nALL M26 BROWSER CHECKS PASSED" : `\n${failures} CHECK(S) FAILED`);
    process.exit(failures === 0 ? 0 : 1);
  } finally {
    await browser.close();
  }
})();
