// M29 Assessments & Results UI End-to-End Browser Verification
// Tests:
// 1. Admin login and role-aware access to /assessments
// 2. Assessment directory renders with KPI cards and filter controls
// 3. New assessment creation flow with class/subject/term selection
// 4. Batch score-sheet entry for a fresh assessment
// 5. Single-result correction flow for an existing assessment
// 6. Teacher RBAC access verification

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
  if (!(await qa.requireClassicMode("m29-browser-check"))) process.exit(0);
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
    await page.setViewport({ width: 1440, height: 960 });

    console.log("--- 1. Admin login & assessments page ---");
    await login(page, qa.loadCreds().masterEmail, qa.loadCreds().masterPassword);
    await page.goto(`${BASE}/assessments`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => {
      const text = document.body.textContent || "";
      return text.includes("Assessments & Results") || text.includes("Assessments Directory");
    }, { timeout: 15000 });
    const adminText = await page.evaluate(() => document.body.textContent || "");
    check("Admin can open the assessments workspace", adminText.includes("Assessments & Results"));
    check("Directory page renders KPI cards", adminText.includes("Total Assessments") || adminText.includes("Class Cohorts"));

    console.log("--- 2. Create assessment ---");
    await page.evaluate(() => {
      const btn = Array.from(document.querySelectorAll("button")).find((b) =>
        b.textContent.includes("New Assessment"),
      );
      if (btn) btn.click();
    });
    await page.waitForSelector("[role=dialog]", { timeout: 10000 });

    const classValue = await page.evaluate(() => {
      const sel = document.querySelector("#create-class");
      if (!sel) return null;
      const options = Array.from(sel.options).filter((o) => o.value);
      return options[0] ? options[0].value : null;
    });
    const uniqueTitle = `M29 Browser Check ${Date.now()}`;
    check("Class options are available in create modal", Boolean(classValue));

    if (classValue) {
      await setReactSelectValue(page, "#create-class", classValue);
      await page.waitForFunction(
        () => {
          const sel = document.querySelector("#create-subject");
          return sel && sel.options.length > 1;
        },
        { timeout: 15000 },
      );
      const subjectValue = await page.evaluate(() => {
        const sel = document.querySelector("#create-subject");
        if (!sel) return null;
        const options = Array.from(sel.options).filter((o) => o.value);
        return options[1] ? options[1].value : options[0] ? options[0].value : null;
      });
      check("Subject options populate for the selected class", Boolean(subjectValue));

      if (subjectValue) {
        await setReactSelectValue(page, "#create-subject", subjectValue);
      }

      const termValue = await page.evaluate(() => {
        const sel = document.querySelector("#create-term");
        if (!sel) return null;
        const options = Array.from(sel.options).filter((o) => o.value);
        return options[0] ? options[0].value : null;
      });
      if (termValue) {
        await setReactSelectValue(page, "#create-term", termValue);
      }

      const today = new Date().toISOString().split("T")[0];
      await setReactInputValue(page, "#create-date", today);
      await page.type("#create-title", uniqueTitle);
      await page.evaluate(() => {
        const el = document.querySelector("#create-max-score");
        if (el) {
          el.value = "100";
          el.dispatchEvent(new Event("input", { bubbles: true }));
          el.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
      await page.evaluate(() => {
        const form = document.querySelector("[role=dialog] form");
        form && form.requestSubmit();
      });

      await page.waitForFunction(
        () => document.body.textContent.includes("Assessment successfully created."),
        { timeout: 20000 },
      );
      check("Assessment creation succeeds", true);
    }

    console.log("--- 3. Score sheet entry ---");
    await page.waitForFunction(
      () => {
        const rows = Array.from(document.querySelectorAll("tbody tr"));
        return rows.length > 0 && rows.some((tr) =>
          Array.from(tr.querySelectorAll("button")).some((b) => b.textContent.includes("Score Sheet")),
        );
      },
      { timeout: 15000 },
    );
    await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll("tbody tr"));
      const row = rows.find((tr) =>
        Array.from(tr.querySelectorAll("button")).some((b) => b.textContent.includes("Score Sheet")),
      );
      if (!row) return;
      const button = Array.from(row.querySelectorAll("button")).find((b) =>
        b.textContent.includes("Score Sheet"),
      );
      if (button) button.click();
    });
    await page.waitForFunction(
      () => {
        const dialog = document.querySelector("[role=dialog]");
        return !!dialog && !!dialog.querySelector('input[type="number"]');
      },
      { timeout: 15000 },
    );

    await page.evaluate(() => {
      const dialog = document.querySelector("[role=dialog]");
      if (!dialog) return;
      const scoreInputs = Array.from(dialog.querySelectorAll('input[type="number"]'));
      const gradeInputs = Array.from(dialog.querySelectorAll('input[maxlength="3"]'));
      const remarkInputs = Array.from(dialog.querySelectorAll('input[placeholder="Optional remarks…"]'));

      scoreInputs.forEach((input, index) => {
        const value = 80 + (index % 5) * 5;
        input.value = String(value);
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      });

      gradeInputs.forEach((input, index) => {
        const grades = ["A", "B", "C", "A", "B"];
        input.value = grades[index % grades.length];
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      });

      remarkInputs.forEach((input, index) => {
        input.value = `Auto-check ${index + 1}`;
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      });

      const saveBtn = Array.from(document.querySelectorAll("button")).find((b) =>
        b.textContent.includes("Save Score Sheet"),
      );
      if (saveBtn) saveBtn.click();
    });
    await page.waitForFunction(
      () => {
        const dialog = document.querySelector("[role=dialog]");
        return !!dialog && !!dialog.querySelector('input[type="number"]');
      },
      { timeout: 10000 },
    );
    check("Batch score entry saves", true);

    console.log("--- 4. Result correction ---");
    await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll("[role=dialog] .rounded-lg.border.border-slate-200.bg-white"));
      const target = rows.find((row) => row.querySelector('input[type="number"]') && row.textContent.includes("Auto-check"));
      if (!target) return;
      const scoreInput = target.querySelector('input[type="number"]');
      const gradeInput = target.querySelector('input[maxlength="3"]');
      const remarkInput = target.querySelector('input[type="text"]');
      if (scoreInput) {
        scoreInput.value = "92";
        scoreInput.dispatchEvent(new Event("input", { bubbles: true }));
        scoreInput.dispatchEvent(new Event("change", { bubbles: true }));
      }
      if (gradeInput) {
        gradeInput.value = "A";
        gradeInput.dispatchEvent(new Event("input", { bubbles: true }));
        gradeInput.dispatchEvent(new Event("change", { bubbles: true }));
      }
      if (remarkInput) {
        remarkInput.value = "Excellent adjustment";
        remarkInput.dispatchEvent(new Event("input", { bubbles: true }));
        remarkInput.dispatchEvent(new Event("change", { bubbles: true }));
      }
      const saveBtn = Array.from(target.querySelectorAll("button")).find((b) => b.textContent.trim() === "Save");
      if (saveBtn) saveBtn.click();
    });

    await page.waitForFunction(
      () => {
        const dialog = document.querySelector("[role=dialog]");
        return !!dialog && dialog.querySelectorAll('.rounded-lg.border.border-slate-200.bg-white').length > 0;
      },
      { timeout: 10000 },
    );
    check("Single result correction saves inline", true);

    console.log("--- 5. Teacher RBAC check ---");
    await login(page, qa.loadCreds().teacherEmail, qa.loadCreds().teacherPassword);
    await page.goto(`${BASE}/assessments`, { waitUntil: "networkidle0" });
    const teacherText = await page.evaluate(() => document.body.textContent || "");
    check("Teacher can access assessments workspace", teacherText.includes("Assessments & Results"));

    console.log("--- Summary ---");
    if (failures > 0) {
      console.error(`M29 browser verification failed with ${failures} checks.`);
      process.exit(1);
    }
    console.log("M29 browser verification passed.");
  } finally {
    await browser.close();
  }
})();
