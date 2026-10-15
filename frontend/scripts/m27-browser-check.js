// One-off M27 browser verification (run with node; needs `npm install --no-save
// puppeteer-core` and /usr/bin/chromium): enrollments directory + filters,
// enroll modal (search-pick), status transitions, student-profile enrollment
// history + enroll-in-class, teaching assignments create/delete.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";
const STAMP = Date.now().toString().slice(-5);
const STU_NO = `STU-M27-${STAMP}`;
const STU_LAST = `M27-${STAMP}`;

async function login(page) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
  await page.type("#email", qa.loadCreds().masterEmail);
  await page.type("#password", qa.loadCreds().masterPassword);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
}

// React overrides input/select value trackers — plain assignment + a bubbling
// event does not update controlled state. Use the prototype setters.
async function setReactValue(page, selector, value, index = 0) {
  await page.evaluate(
    (sel, val, idx) => {
      const el = Array.from(document.querySelectorAll(sel))[idx];
      const proto =
        el instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
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

async function clickButtonWithText(page, text) {
  const handle = await page.evaluateHandle((t) => {
    const btns = Array.from(document.querySelectorAll("button"));
    return btns.find((b) => b.textContent?.trim().includes(t));
  }, text);
  const el = handle.asElement();
  if (!el) throw new Error(`button not found: ${text}`);
  await el.click();
}

async function submitLastForm(p) {
  await p.evaluate(() => {
    const forms = document.querySelectorAll("form");
    forms[forms.length - 1].requestSubmit();
  });
}

async function debugDialog(page) {
  console.log(
    await page.evaluate(() => {
      const d = document.querySelector("[role=dialog]");
      return d ? d.textContent?.slice(0, 800) : "NO DIALOG";
    }),
  );
}

// Pick an option whose text contains `needle` inside the first select whose
// option list contains `probe`, firing React-compatible events.
function pickSelectOption(page, probe, needle) {
  return page.evaluate(
    (prb, ned) => {
      const sel = Array.from(document.querySelectorAll("form select")).find((s) =>
        Array.from(s.options).some((o) => o.textContent?.includes(prb)),
      );
      if (!sel) throw new Error(`select with option ${prb} not found`);
      const opt = Array.from(sel.options).find((o) => o.textContent?.includes(ned));
      if (!opt) throw new Error(`option ${ned} not found`);
      const proto = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set;
      proto.call(sel, opt.value);
      sel.dispatchEvent(new Event("change", { bubbles: true }));
      return opt.textContent;
    },
    probe,
    needle,
  );
}

(async () => {
  if (!(await qa.requireClassicMode("m27-browser-check"))) process.exit(0);
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

    // --- 1. Enrollments directory ---
    await page.goto(`${BASE}/enrollments`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () =>
        document.body.innerText.includes("Enrollment Register") &&
        !document.body.innerText.includes("Checking session"),
      { timeout: 20000 },
    );
    let t = await text();
    check("enrollments directory renders", t.includes("Enrollment Register"));
    check("seeded enrollments listed", t.includes("Fiifi Ackon"));
    check("status badges render", /Enrolled/.test(t));

    // --- 2. Status filter narrows (WITHDRAWN shows an empty register) ---
    await setReactValue(page, "select[aria-label='Filter by status']", "WITHDRAWN");
    await page.waitForFunction(
      () => !document.body.innerText.includes("Fiifi Ackon"),
      { timeout: 10000 },
    );
    t = await text();
    check("status filter narrows to WITHDRAWN", t.includes("No records found"));
    await setReactValue(page, "select[aria-label='Filter by status']", "");
    await page.waitForFunction(() => document.body.innerText.includes("Fiifi Ackon"), {
      timeout: 10000,
    });

    // --- 3. Create a fresh student, then enroll them via the register modal ---
    await page.goto(`${BASE}/students`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Student Records"),
      { timeout: 20000 },
    );
    await clickButtonWithText(page, "Enroll student");
    await page.waitForSelector("#student-number", { timeout: 5000 });
    await page.type("#student-number", STU_NO);
    await page.type("#student-first-name", "Zuri");
    await page.type("#student-last-name", STU_LAST);
    await submitLastForm(page);
    await page.waitForFunction(
      () => !document.querySelector("[role=dialog]"),
      { timeout: 10000 },
    );
    await page.type("input[type=search]", STU_NO);
    await page.waitForFunction(
      (no) => document.body.innerText.includes(no),
      { timeout: 10000 },
      STU_NO,
    );
    check("fresh student created", true);

    await page.goto(`${BASE}/enrollments`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Enrollment Register"), {
      timeout: 20000,
    });
    await clickButtonWithText(page, "Enroll student");
    await page.waitForFunction(() => document.body.innerText.includes("FIND STUDENT"), {
      timeout: 5000,
    });
    await page.type("[role=dialog] input[type=search]", STU_LAST);
    await page.waitForFunction(
      (no) => !document.body.innerText.includes("Searching…"),
      { timeout: 10000 },
      STU_NO,
    );
    await page.evaluate((last) => {
      const btn = Array.from(document.querySelectorAll("[role=dialog] form button")).find((b) =>
        b.textContent?.includes(last),
      );
      if (!btn) throw new Error("student pick button not found");
      btn.click();
    }, STU_LAST);
    // academic year: explicitly pick 2026/2027 (has classes; the current
    // flag should also point here, but don't rely on it)
    await pickSelectOption(page, "(current)", "2026/2027");
    await page.waitForFunction(() => {
      const selects = Array.from(document.querySelectorAll("[role=dialog] form select"));
      const classSel = selects.find((s) => s.textContent?.includes("— select class —"));
      return classSel && classSel.options.length > 1;
    }, { timeout: 10000 });
    // class select: first class in the academic-year-filtered list
    await page.evaluate(() => {
      const selects = Array.from(document.querySelectorAll("[role=dialog] form select"));
      const classSel = selects.find((s) => s.textContent?.includes("— select class —"));
      if (!classSel || classSel.options.length < 2) throw new Error("no class options");
      const proto = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set;
      proto.call(classSel, classSel.options[1].value);
      classSel.dispatchEvent(new Event("change", { bubbles: true }));
    });
    await submitLastForm(page);
    await page
      .waitForFunction(() => document.body.innerText.includes("enrolled in"), { timeout: 10000 })
      .catch(async () => {
        console.log("DEBUG after enroll submit, dialog body:");
        await debugDialog(page);
        throw new Error("enroll wait failed");
      });
    check("student enrolled via modal", true);

    // --- 4. Status transition from the register row (search first: fresh
    // enrollments sort by enrollment_date DESC so the row is on page 1, but
    // the active filter state after step 2 is "any") ---
    await page.type("input[type=search]", STU_NO);
    await page.waitForFunction(
      (no) => document.body.innerText.includes(no),
      { timeout: 10000 },
      STU_NO,
    );
    const transitioned = await page.evaluate((no) => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find((r) => r.textContent?.includes(no));
      if (!row) return false;
      const sel = row.querySelector("select");
      if (!sel) return false;
      const proto = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set;
      proto.call(sel, "COMPLETED");
      sel.dispatchEvent(new Event("change", { bubbles: true }));
      return true;
    }, STU_NO);
    await page
      .waitForFunction(() => document.body.innerText.includes("marked completed"), {
        timeout: 10000,
      })
      .catch(async () => {
        console.log(`DEBUG transitioned=${transitioned} body:`);
        console.log((await text()).slice(0, 900));
        throw new Error("transition wait failed");
      });
    check("enrollment status transitioned", transitioned);

    // --- 5. Student profile: enrollment history + enroll-in-class ---
    await page.goto(`${BASE}/students`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Student Records"), {
      timeout: 20000,
    });
    await page.type("input[type=search]", STU_NO);
    await page.waitForFunction(
      (no) => document.body.innerText.includes(no),
      { timeout: 10000 },
      STU_NO,
    );
    const stuHref = await page.evaluate((no) => {
      const l = Array.from(document.querySelectorAll("a")).find((a) =>
        a.closest("tr")?.textContent?.includes(no),
      );
      return l?.getAttribute("href");
    }, STU_NO);
    if (!stuHref) throw new Error("student profile link not found");
    await page.goto(`${BASE}${stuHref}`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Enrollment history"), {
      timeout: 20000,
    });
    t = await text();
    check(
      "profile shows enrollment history",
      (t.includes("COMPLETED") || t.includes("Completed")) && t.includes("Form 1"),
    );

    // enroll in a class from the profile — in 2027/2028: the DB-level
    // partial-unique index blocks ANY re-enroll while the 2026/2027 row is
    // ENROLLED/COMPLETED/TRANSFERRED, so step 4's COMPLETED transition is a
    // history test, not an unlock. 2027/2028 has its own Form 1A class.
    await clickButtonWithText(page, "Enroll in class");
    await page.waitForSelector("form select", { timeout: 5000 });
    await page.evaluate(() => {
      const selects = Array.from(document.querySelectorAll("form select"));
      const yearSel = selects[0];
      const opt = Array.from(yearSel.options).find((o) => o.textContent?.includes("2027/2028"));
      if (!opt) throw new Error("2027/2028 option missing");
      const proto = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set;
      proto.call(yearSel, opt.value);
      yearSel.dispatchEvent(new Event("change", { bubbles: true }));
    });
    await page.waitForFunction(() => {
      const selects = Array.from(document.querySelectorAll("form select"));
      const classSel = selects.find((s) => s.textContent?.includes("— select class —"));
      return classSel && Array.from(classSel.options).some((o) => o.textContent?.includes("Form 1A"));
    }, { timeout: 10000 });
    await page.evaluate(() => {
      const selects = Array.from(document.querySelectorAll("form select"));
      const classSel = selects.find((s) => s.textContent?.includes("— select class —"));
      const opt = Array.from(classSel.options).find((o) => o.textContent?.includes("Form 1A"));
      const proto = Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, "value").set;
      proto.call(classSel, opt.value);
      classSel.dispatchEvent(new Event("change", { bubbles: true }));
    });
    await submitLastForm(page);
    await page
      .waitForFunction(() => document.body.innerText.includes("Student enrolled."), {
        timeout: 10000,
      })
      .catch(async () => {
        console.log("DEBUG after profile enroll submit:");
        await debugDialog(page);
        throw new Error("profile enroll wait failed");
      });
    check("enrolled from profile", true);

    // --- 6. Teaching assignments: create + delete (Sarah Mensah + Form 1B +
    // MATH: MATH is in Form 1B's curriculum and not yet assigned to Sarah) ---
    await page.goto(`${BASE}/teaching`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Teaching Assignments"), {
      timeout: 20000,
    });
    t = await text();
    check("teaching assignments directory renders", t.includes("teacher → class → subject"));

    await clickButtonWithText(page, "Assign teacher");
    await page.waitForFunction(() => document.body.innerText.includes("— select teacher —"), {
      timeout: 5000,
    });
    const teacherName = await pickSelectOption(page, "Sarah Mensah", "Sarah Mensah");
    const className = await pickSelectOption(page, "— select class —", "Form 1B");
    await page.waitForFunction(() => {
      const selects = Array.from(document.querySelectorAll("form select"));
      const subj = selects.find((s) => s.textContent?.includes("— select subject —"));
      return subj && subj.options.length > 1;
    }, { timeout: 10000 });
    const subjectName = await pickSelectOption(page, "— select subject —", "MATH");
    if (!teacherName.includes("Sarah") || !className.includes("Form 1B")) {
      throw new Error(`unexpected picks: ${teacherName} / ${className}`);
    }
    await submitLastForm(page);
    await page
      .waitForFunction(() => document.body.innerText.includes("Teacher assigned."), {
        timeout: 10000,
      })
      .catch(async () => {
        console.log("DEBUG after assignment create submit:");
        await debugDialog(page);
        throw new Error("assignment create wait failed");
      });
    check("teacher assigned via modal", true);

    // duplicate assignment should show the backend error inline
    await clickButtonWithText(page, "Assign teacher");
    await page.waitForFunction(() => document.body.innerText.includes("— select teacher —"), {
      timeout: 5000,
    });
    await pickSelectOption(page, "Sarah Mensah", "Sarah Mensah");
    await pickSelectOption(page, "— select class —", "Form 1B");
    await page.waitForFunction(() => {
      const selects = Array.from(document.querySelectorAll("form select"));
      const subj = selects.find((s) => s.textContent?.includes("— select subject —"));
      return subj && subj.options.length > 1;
    }, { timeout: 10000 });
    await pickSelectOption(page, "— select subject —", "MATH");
    await submitLastForm(page);
    await page
      .waitForFunction(() => document.body.innerText.includes("already"), { timeout: 10000 })
      .catch(async () => {
        console.log("DEBUG duplicate submit:");
        await debugDialog(page);
        throw new Error("duplicate wait failed");
      });
    check("duplicate assignment rejected inline", true);
    // close the modal
    await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll("[role=dialog] button"));
      const cancel = btns.find((b) => b.textContent?.trim() === "Cancel");
      if (cancel) cancel.click();
    });

    // delete the assignment just created (Sarah + Form 1B; Ada's Form 1B rows
    // don't collide because they carry her name)
    const removed = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find(
        (r) => r.textContent?.includes("Sarah Mensah") && r.textContent?.includes("Form 1B"),
      );
      const btn = row?.querySelector("button[title='Remove assignment']");
      if (btn) {
        btn.click();
        return true;
      }
      return false;
    });
    await page
      .waitForFunction(() => document.body.innerText.includes("unassigned"), { timeout: 10000 })
      .catch(async () => {
        console.log("DEBUG remove wait, body:");
        console.log((await text()).slice(0, 900));
        throw new Error("remove wait failed");
      });
    check("assignment removed", removed);

    console.log(
      failures === 0 ? "\nALL M27 BROWSER CHECKS PASSED" : `\n${failures} CHECK(S) FAILED`,
    );
    process.exit(failures === 0 ? 0 : 1);
  } finally {
    await browser.close();
  }
})();
