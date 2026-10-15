const puppeteer = require("puppeteer-core");
(async () => {
  if (!(await require("./lib/check-accounts").requireClassicMode("parent-check"))) process.exit(0);
  const b = await puppeteer.launch({ executablePath: "/usr/bin/chromium", headless: "new", args: ["--no-sandbox","--disable-gpu"], defaultViewport: {width:1440,height:900} });
  const p = await b.newPage();
  await p.goto("http://localhost:3000/login", { waitUntil: "domcontentloaded", timeout: 30000 });
  await new Promise(r => setTimeout(r, 1200));
  const res = await p.evaluate(async () => {
    try {
      const r = await fetch("http://localhost:8080/api/v1/auth/login", {
        method: "POST", credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "qa-parent@schoolhub.test", password: require("./lib/check-accounts").loadCreds().parentPassword }),
      });
      return "login HTTP " + r.status;
    } catch (e) { return "fetch failed: " + e.message; }
  });
  console.log(res);
  await p.goto("http://localhost:3000/portal/parent", { waitUntil: "domcontentloaded", timeout: 30000 });
  await new Promise(r => setTimeout(r, 5000));
  const text = await p.evaluate(() => document.body.innerText);
  console.log("portal body:", text.slice(0, 320).replace(/\n/g, " | "));
  await b.close();
})().catch(e => { console.error("ERR:", e.message); process.exit(1); });
