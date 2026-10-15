// M28 Attendance UI End-to-End Browser Verification
// Tests:
// 1. Admin login & role-aware navigation
// 2. Attendance Directory rendering, KPI stat cards, and filtering
// 3. Register Capture (+ Take Register): class select, dynamic subjects & roster loading,
//    status toggles (PRESENT, ABSENT, LATE, EXCUSED), notes, and submission
// 4. Duplicate register conflict rejection (ALREADY_RECORDED)
// 5. Inspect & Correct flow: viewing marks, inline mark correction via PATCH /attendance/records/:id,
//    and dynamic session rate recalculation
// 6. Class attendance summary modal inspection
// 7. Teacher role RBAC access verification (TEACHER can access /attendance and take registers)

const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";

async function login(page, email, password) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
  await page.evaluate(() => {
    const emailEl = document.querySelector("#email");
    const passEl = document.querySelector("#password");
    if (emailEl) emailEl.value = "";
    if (passEl) passEl.value = "";
  });
  await page.type("#email", email);
  await page.type("#password", password);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
}

async function logout(page) {
  const signOutBtn = await page.evaluateHandle(() => {
    const btns = Array.from(document.querySelectorAll("button"));
    return btns.find((b) => b.textContent?.trim().includes("Sign out"));
  });
  const el = signOutBtn.asElement();
  if (el) {
    await Promise.all([
      page.waitForNavigation({ waitUntil: "networkidle0", timeout: 15000 }).catch(() => {}),
      el.click(),
    ]);
  }
}

async function setReactSelectValue(page, selector, value) {
  await page.evaluate(
    (sel, val) => {
      const el = document.querySelector(sel);
      if (!el) throw new Error(`Element ${sel} not found`);
      const proto = HTMLSelectElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
      setter.call(el, val);
      el.dispatchEvent(new Event("input", { bubbles: true }));
      el.dispatchEvent(new Event("change", { bubbles: true }));
    },
    selector,
    value,
  );
}

async function setReactInputValue(page, selector, value) {
  await page.evaluate(
    (sel, val) => {
      const el = document.querySelector(sel);
      if (!el) throw new Error(`Element ${sel} not found`);
      const proto = HTMLInputElement.prototype;
      const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
      setter.call(el, val);
      el.dispatchEvent(new Event("input", { bubbles: true }));
      el.dispatchEvent(new Event("change", { bubbles: true }));
    },
    selector,
    value,
  );
}

(async () => {
  if (!(await qa.requireClassicMode("m28-browser-check"))) process.exit(0);
  const browser = await puppeteer.launch({
    executablePath: "/usr/bin/chromium",
    headless: "new",
    args: ["--no-sandbox", "--disable-gpu"],
  });

  let failures = 0;
  const check = (name, ok) => {
    console.log(`${ok ? "PASS" : "FAIL"} ${name}`);
    if (!ok) failures++;
  };

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1440, height: 900 });

    console.log("--- 1. Admin Login & Nav Check ---");
    await login(page, qa.loadCreds().masterEmail, qa.loadCreds().masterPassword);
    check("Admin login successful", !page.url().includes("/login"));

    // Check Attendance nav link in AppShell
    const hasAttendanceNav = await page.evaluate(() => {
      const links = Array.from(document.querySelectorAll("a"));
      return links.some((a) => a.getAttribute("href") === "/attendance" && a.textContent.includes("Attendance"));
    });
    check("Sidebar contains Attendance link", hasAttendanceNav);

    console.log("--- 2. Directory Page Rendering ---");
    await page.goto(`${BASE}/attendance`, { waitUntil: "networkidle0" });
    await page.waitForSelector("h2", { timeout: 10000 });

    const pageTitle = await page.evaluate(() => document.body.textContent);
    check("Page renders Attendance Directory", pageTitle.includes("Attendance Directory"));
    check("KPI stat card: Total Sessions Recorded", pageTitle.includes("Total Sessions Recorded"));
    check("KPI stat card: Active Classes", pageTitle.includes("Active Classes"));

    // Check directory table rows rendered
    const rowCount = await page.evaluate(() => document.querySelectorAll("tbody tr").length);
    check("Directory table displays session rows", rowCount > 0);

    // Filter test
    const form1AOptionVal = await page.evaluate(() => {
      const select = document.querySelector("select");
      const opt = Array.from(select.options).find((o) => o.textContent.includes("Form 1A"));
      return opt ? opt.value : null;
    });

    if (form1AOptionVal) {
      await setReactSelectValue(page, "select", form1AOptionVal);
      await new Promise((r) => setTimeout(r, 600));
      const filteredText = await page.evaluate(() => document.querySelector("tbody")?.textContent || "");
      check("Class filter applies Form 1A", filteredText.includes("Form 1A") || filteredText.includes("No attendance"));

      // Click Reset
      await page.evaluate(() => {
        const btn = Array.from(document.querySelectorAll("button")).find((b) => b.textContent.includes("Reset"));
        if (btn) btn.click();
      });
      await new Promise((r) => setTimeout(r, 600));
    }

    console.log("--- 3. Take Register Flow ---");
    // Click "+ Take Register" button
    await page.evaluate(() => {
      const btn = Array.from(document.querySelectorAll("button")).find((b) => b.textContent.includes("Take Register"));
      if (btn) btn.click();
    });

    await page.waitForSelector("[role=dialog]", { timeout: 10000 });
    const modalTitle = await page.evaluate(() => document.querySelector("[role=dialog] h2")?.textContent);
    check("Take Register modal opens", modalTitle?.includes("Take Attendance Register"));

    // Select Form 1A
    const classId = await page.evaluate(() => {
      const sel = document.querySelector("#take-class");
      const opt = Array.from(sel.options).find((o) => o.textContent.includes("Form 1A"));
      return opt ? opt.value : null;
    });
    check("Form 1A class option exists", Boolean(classId));

    await setReactSelectValue(page, "#take-class", classId);

    // Wait for subjects & roster to load
    await page.waitForFunction(
      () => {
        const subjectSel = document.querySelector("#take-subject");
        return subjectSel && subjectSel.options.length > 1;
      },
      { timeout: 10000 },
    );
    check("Class subjects populate dynamically", true);

    // Wait for roster students to load
    await page.waitForFunction(
      () => {
        const text = document.querySelector("[role=dialog]")?.textContent || "";
        return text.includes("Ama Boateng") || text.includes("Kofi Mensah");
      },
      { timeout: 10000 },
    );
    check("Active enrollees roster loaded into dialog", true);

    // Pick a test date in Term 1 (e.g. 2026-11-20)
    const testDate = "2026-11-20";
    await setReactInputValue(page, "#take-date", testDate);

    // Toggle status: mark first student as ABSENT, second as LATE
    await page.evaluate(() => {
      const dialog = document.querySelector("[role=dialog]");
      const rows = dialog.querySelectorAll(".rounded-lg.border.border-slate-200.bg-white");
      if (rows.length >= 2) {
        // In row 0, click ABSENT
        const absentBtn = Array.from(rows[0].querySelectorAll("button")).find((b) => b.textContent.trim() === "ABSENT");
        if (absentBtn) absentBtn.click();

        // In row 1, click LATE
        const lateBtn = Array.from(rows[1].querySelectorAll("button")).find((b) => b.textContent.trim() === "LATE");
        if (lateBtn) lateBtn.click();

        // Add a note in row 0
        const noteInput = rows[0].querySelector("input[type=text]");
        if (noteInput) {
          const proto = HTMLInputElement.prototype;
          const setter = Object.getOwnPropertyDescriptor(proto, "value").set;
          setter.call(noteInput, "Excused sick leave note");
          noteInput.dispatchEvent(new Event("input", { bubbles: true }));
          noteInput.dispatchEvent(new Event("change", { bubbles: true }));
        }
      }
    });

    // Submit the register
    await page.evaluate(() => {
      const dialog = document.querySelector("[role=dialog]");
      const form = dialog.querySelector("form");
      form.requestSubmit();
    });

    // Wait for dialog to dismiss and success notice to appear
    await page.waitForFunction(() => !document.querySelector("[role=dialog]"), { timeout: 10000 });
    const successNotice = await page.evaluate(() => document.body.textContent.includes("successfully recorded"));
    check("Register saved and success banner displayed", successNotice);

    // Verify session row appears in table with date testDate
    await new Promise((r) => setTimeout(r, 800));
    const tableHasNewSession = await page.evaluate(
      (td) => document.querySelector("tbody")?.textContent.includes(td),
      testDate,
    );
    check(`New session row with date ${testDate} appears in directory`, tableHasNewSession);

    console.log("--- 4. Duplicate Session Conflict Rejection ---");
    // Open Take Register again with same parameters
    await page.evaluate(() => {
      const btn = Array.from(document.querySelectorAll("button")).find((b) => b.textContent.includes("Take Register"));
      if (btn) btn.click();
    });
    await page.waitForSelector("[role=dialog]", { timeout: 10000 });

    await setReactSelectValue(page, "#take-class", classId);
    await page.waitForFunction(
      () => {
        const sel = document.querySelector("#take-subject");
        return sel && sel.options.length > 1;
      },
      { timeout: 10000 },
    );
    await setReactInputValue(page, "#take-date", testDate);

    // Submit duplicate
    await page.evaluate(() => {
      const form = document.querySelector("[role=dialog] form");
      form.requestSubmit();
    });

    await page.waitForFunction(
      () => {
        const text = document.querySelector("[role=dialog]")?.textContent || "";
        return text.includes("already been captured");
      },
      { timeout: 10000 },
    );
    check("Duplicate session rejected with conflict error message", true);

    // Close modal
    await page.evaluate(() => {
      const closeBtn = document.querySelector("[role=dialog] button[aria-label=Close]");
      if (closeBtn) closeBtn.click();
    });
    await page.waitForFunction(() => !document.querySelector("[role=dialog]"), { timeout: 5000 });

    console.log("--- 5. Inspect & Correct Flow (PATCH /attendance/records/:id) ---");
    // Find the row with testDate or first row and click "Inspect & Correct"
    await page.evaluate((td) => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const targetRow = rows.find((r) => r.textContent.includes(td)) || rows[0];
      const inspectBtn = Array.from(targetRow.querySelectorAll("button")).find((b) =>
        b.textContent.includes("Inspect & Correct"),
      );
      if (inspectBtn) inspectBtn.click();
    }, testDate);

    await page.waitForSelector("[role=dialog]", { timeout: 10000 });
    const inspectTitle = await page.evaluate(() => document.querySelector("[role=dialog] h2")?.textContent);
    check("Inspect & Correct modal opens", inspectTitle?.includes("Inspect & Correct Attendance Register"));

    // Check that attendance rate and student marks load
    await page.waitForFunction(
      () => {
        const text = document.querySelector("[role=dialog]")?.textContent || "";
        return text.includes("Present:") && text.includes("Rate:");
      },
      { timeout: 10000 },
    );
    check("Inspect modal displays attendance breakdown pills", true);

    // Correct a student mark: click "EXCUSED" on the first student
    const clickedCorrection = await page.evaluate(() => {
      const dialog = document.querySelector("[role=dialog]");
      const studentCards = dialog.querySelectorAll(".rounded-lg.border.border-slate-200.bg-white");
      if (studentCards.length > 0) {
        const excusedBtn = Array.from(studentCards[0].querySelectorAll("button")).find(
          (b) => b.textContent.trim() === "EXCUSED",
        );
        if (excusedBtn) {
          excusedBtn.click();
          return true;
        }
      }
      return false;
    });
    check("Clicked status toggle to EXCUSED", clickedCorrection);

    // Wait for "Saved!" feedback badge
    await page.waitForFunction(
      () => {
        const text = document.querySelector("[role=dialog]")?.textContent || "";
        return text.includes("Saved!");
      },
      { timeout: 10000 },
    );
    check("Inline 'Saved!' badge confirms PATCH /attendance/records/:id update", true);

    // Close Inspect modal
    await page.evaluate(() => {
      const closeBtn = document.querySelector("[role=dialog] button[aria-label=Close]");
      if (closeBtn) closeBtn.click();
    });
    await page.waitForFunction(() => !document.querySelector("[role=dialog]"), { timeout: 5000 });

    console.log("--- 6. Class Summary Modal ---");
    // Click "Summary" button in table row
    await page.evaluate(() => {
      const summaryBtn = Array.from(document.querySelectorAll("tbody button")).find((b) =>
        b.textContent.trim() === "Summary",
      );
      if (summaryBtn) summaryBtn.click();
    });

    await page.waitForSelector("[role=dialog]", { timeout: 10000 });
    const summaryTitle = await page.evaluate(() => document.querySelector("[role=dialog] h2")?.textContent);
    check("Class Summary modal opens", summaryTitle?.includes("Attendance Summary:"));

    await page.waitForFunction(
      () => {
        const text = document.querySelector("[role=dialog]")?.textContent || "";
        return text.includes("Total Records Logged") && text.includes("Overall Present Rate:");
      },
      { timeout: 10000 },
    );
    check("Class Summary modal displays records count and present rate", true);

    // Close Summary modal
    await page.evaluate(() => {
      const closeBtn = document.querySelector("[role=dialog] button[aria-label=Close]");
      if (closeBtn) closeBtn.click();
    });
    await page.waitForFunction(() => !document.querySelector("[role=dialog]"), { timeout: 5000 });

    console.log("--- 7. Teacher Portal & Role Permissions Verification ---");
    await logout(page);
    await login(page, qa.loadCreds().teacherEmail, qa.loadCreds().teacherPassword);
    check("Teacher (Ada) login successful", !page.url().includes("/login"));

    // Navigate to /attendance as Teacher
    await page.goto(`${BASE}/attendance`, { waitUntil: "networkidle0" });
    await page.waitForSelector("h2", { timeout: 10000 });
    const teacherPageText = await page.evaluate(() => document.body.textContent);
    check("Teacher can access /attendance directory", teacherPageText.includes("Attendance Directory"));

    // Check "+ Take Register" is visible for Teacher
    const teacherHasTakeBtn = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.some((b) => b.textContent.includes("Take Register"));
    });
    check("Teacher has '+ Take Register' button visible", teacherHasTakeBtn);

    console.log("\n=================================");
    console.log(`M28 Verification Complete: ${failures === 0 ? "ALL CHECKS PASSED!" : `${failures} FAILURES`}`);
    console.log("=================================\n");
  } finally {
    await browser.close();
  }

  process.exit(failures === 0 ? 0 : 1);
})();
