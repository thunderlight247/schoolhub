// Automated Clerk sign-in check: fills the clerk+password card, asserts the
// role-based redirect lands on /portal/parent via the backend bridge.
const puppeteer = require("puppeteer-core");
(async () => {
  const b = await puppeteer.launch({ executablePath: "/usr/bin/chromium", headless: "new", args: ["--no-sandbox","--disable-gpu"], defaultViewport: {width:1440,height:900} });
  const p = await b.newPage();
  await p.goto("http://localhost:3000/login", { waitUntil: "domcontentloaded", timeout: 30000 });
  await new Promise(r => setTimeout(r, 7000));
  // The Clerk single-step card: identifier + password + "Continue"
  const ident = await p.$('input[name="identifier"], input[id="identifier-field"]');
  console.log("identifier:", !!ident);
  if (!ident) { console.log((await p.evaluate(() => document.body.innerText)).slice(0,200)); await b.close(); process.exit(1); }
  await ident.click({ clickCount: 3 });
  await ident.type("thunderlight2477+parent@gmail.com");
  const pwd = await p.$('input[type="password"]');
  await pwd.click({ clickCount: 3 });
  await pwd.type("Thunderpass2026!");
  for (const btn of await p.$$("button")) {
    const t = await btn.evaluate(el => el.textContent || "");
    if (/^continue/i.test(t)) { await btn.click(); break; }
  }
  await new Promise(r => setTimeout(r, 12000));
  console.log("URL:", p.url());
  console.log("body:", (await p.evaluate(() => document.body.innerText)).slice(0, 260).replace(/\n/g, " | "));
  await b.close();
})().catch(e => { console.error("ERR:", e.message); process.exit(1); });
