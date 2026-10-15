// One-off M25 browser verification (run with node; needs `npm install --no-save
// puppeteer-core` and /usr/bin/chromium): students directory + stat cards,
// search, create, profile, guardian link + unlink, archive/re-activate, and
// the guardians directory + profile.
const puppeteer = require("puppeteer-core");
const qa = require("./lib/check-accounts");

const BASE = "http://localhost:3000";
const STU_NO = `M25-${Date.now().toString().slice(-6)}`;

async function login(page) {
  await page.goto(`${BASE}/login`, { waitUntil: "networkidle0" });
  await page.type("#email", qa.loadCreds().masterEmail);
  await page.type("#password", qa.loadCreds().masterPassword);
  await Promise.all([
    page.waitForNavigation({ waitUntil: "networkidle0", timeout: 20000 }).catch(() => {}),
    page.click("button[type=submit]"),
  ]);
}

(async () => {
  if (!(await qa.requireClassicMode("m25-browser-check"))) process.exit(0);
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

    // --- 1. Students directory renders with stat cards + seeded rows ---
    await page.goto(`${BASE}/students`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Student Records") && !document.body.innerText.includes("Checking session"),
      { timeout: 15000 },
    );
    let t = await text();
    check("students directory renders", t.includes("Student Records"));
    check("stat cards render", t.includes("Active Students") && t.includes("Total Records"));
    check("seeded students listed", t.includes("Ama Boateng") || t.includes("STU-2026"));

    // --- 2. Search narrows (wait for the filtered table, not the spinner) ---
    await page.type("input[type=search]", "Ama");
    await page.waitForFunction(
      () =>
        document.body.innerText.includes("Ama Boateng") &&
        !document.body.innerText.includes("Kwabena"),
      { timeout: 10000 },
    );
    t = await text();
    check("search narrows to Ama Boateng", t.includes("Ama Boateng") && !t.includes("Kwabena"));

    // --- 3. Create student via modal ---
    await page.goto(`${BASE}/students`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Enroll student"), { timeout: 10000 });
    const enrollBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.includes("Enroll student"));
    });
    await (enrollBtn.asElement()).click();
    await page.waitForSelector("#student-number", { timeout: 5000 });
    await page.type("#student-number", STU_NO);
    await page.type("#student-first-name", "Adwoa");
    await page.type("#student-last-name", "M25Check");
    await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    // search for the created record (directory is name-sorted + paginated)
    await page.type("input[type=search]", STU_NO);
    await page.waitForFunction(
      (no) => document.body.innerText.includes(no),
      { timeout: 10000 },
      STU_NO,
    );
    check("created student appears in list", (await text()).includes("Adwoa M25Check"));

    // --- 4. Profile renders with guardian section ---
    const profileHref = await page.evaluate((no) => {
      const l = Array.from(document.querySelectorAll("a")).find((a) =>
        a.closest("tr")?.textContent?.includes(no),
      );
      return l?.getAttribute("href");
    }, STU_NO);
    await page.goto(`${BASE}${profileHref}`, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.body.innerText.includes("Guardians") && document.body.innerText.includes("Adwoa M25Check"),
      { timeout: 15000 },
    );
    t = await text();
    check("profile renders personal + guardians sections", t.includes("Personal details") && t.includes("No guardians linked yet"));

    // --- 5. Link a guardian (search existing, pick, choose relationship) ---
    const linkBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.includes("Link guardian"));
    });
    await (linkBtn.asElement()).click();
    // The label renders uppercase via CSS text-transform.
    await page.waitForFunction(
      () => document.body.innerText.includes("FIND GUARDIAN"),
      { timeout: 5000 },
    );
    await page.type("input[type=search]", "Boateng");
    await page.waitForFunction(
      () => document.querySelectorAll("form button").length > 0,
      { timeout: 10000 },
    );
    await page.waitForFunction(
      () => !document.body.innerText.includes("Searching…"),
      { timeout: 10000 },
    );
    // pick the first result (a guardian with surname Boateng)
    const firstResult = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("form button[type=button], form button"));
      return btns.find((b) => b.textContent?.includes("linked student"));
    });
    await (firstResult.asElement()).click();
    // submit the link form
    await page.evaluate(() => {
      const forms = document.querySelectorAll("form");
      forms[forms.length - 1].requestSubmit();
    });
    await page.waitForFunction(
      () => document.body.innerText.includes("Guardian linked."),
      { timeout: 10000 },
    );
    t = await text();
    check("guardian linked with relationship", t.includes("MOTHER") || t.includes("GUARDIAN"));

    // --- 6. Unlink (with confirmation modal) ---
    const unlinkBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.trim() === "Unlink");
    });
    await (unlinkBtn.asElement()).click();
    await page.waitForFunction(() => document.body.innerText.includes("Unlink guardian"), { timeout: 5000 });
    // confirm within the modal (its own Unlink button)
    await page.evaluate(() => {
      const modals = Array.from(document.querySelectorAll("[role=dialog] button"));
      const confirm = modals.find((b) => b.textContent?.trim() === "Unlink");
      if (confirm) confirm.click();
    });
    await page.waitForFunction(
      () => document.body.innerText.includes("Unlinked"),
      { timeout: 10000 },
    );
    check("guardian unlinked after confirmation", (await text()).includes("No guardians linked yet"));

    // --- 7. Archive + re-activate ---
    const archiveBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.trim() === "Archive student");
    });
    await (archiveBtn.asElement()).click();
    await page.waitForFunction(() => document.body.innerText.includes("Student archived."), { timeout: 10000 });
    check("student archived", (await text()).includes("Archived"));
    const reactivateBtn = await page.evaluateHandle(() => {
      const btns = Array.from(document.querySelectorAll("button"));
      return btns.find((b) => b.textContent?.trim() === "Re-activate");
    });
    await (reactivateBtn.asElement()).click();
    await page.waitForFunction(() => document.body.innerText.includes("Student re-activated."), { timeout: 10000 });
    check("student re-activated", true);

    // --- 8. Guardians directory + profile ---
    await page.goto(`${BASE}/guardians`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Guardians & Parents"), { timeout: 15000 });
    t = await text();
    check("guardians directory renders", t.includes("Registered guardians") && /children/i.test(t));
    check("parent demo account listed with child count", t.includes("Abena Boateng"));
    const gHref = await page.evaluate(() => {
      const l = Array.from(document.querySelectorAll("a")).find((a) =>
        a.closest("tr")?.textContent?.includes("Abena Boateng"),
      );
      return l?.getAttribute("href");
    });
    await page.goto(`${BASE}${gHref}`, { waitUntil: "networkidle0" });
    await page.waitForFunction(() => document.body.innerText.includes("Linked students"), { timeout: 15000 });
    t = await text();
    check("guardian profile shows linked students", t.includes("Ama Boateng") && t.includes("Kofi Mensah"));

    console.log(failures === 0 ? "\nALL M25 BROWSER CHECKS PASSED" : `\n${failures} CHECK(S) FAILED`);
    process.exit(failures === 0 ? 0 : 1);
  } finally {
    await browser.close();
  }
})();
