import { chromium } from 'playwright';

const BASE = process.env.BASE || 'http://web';
const EV = process.env.EV;
const OUT = process.env.OUT || 'docs/screenshots';
const VP = { width: 1440, height: 900 };

async function ctx(browser, { lang = 'en', email, password }) {
  const c = await browser.newContext({ viewport: VP, deviceScaleFactor: 2 });
  await c.addInitScript((l) => { try { localStorage.setItem('reduta.lang', l); } catch {} }, lang);
  if (email) {
    const r = await c.request.post(`${BASE}/api/v1/auth/login`, { data: { email, password } });
    if (!r.ok()) throw new Error('login failed ' + email + ' ' + r.status());
  }
  return c;
}

const shot = async (page, name) => {
  await page.waitForTimeout(700);
  await page.screenshot({ path: `${OUT}/${name}.png` });
  console.log('shot', name);
};

const run = async () => {
  const browser = await chromium.launch();
  const admin = { email: 'admin@reduta.local', password: 'admin-dev-password' };
  const player = { email: 'demo1@demo.local', password: 'demo-pass-123' };

  // Player, EN: board
  let c = await ctx(browser, { lang: 'en', ...player });
  let p = await c.newPage();
  await p.goto(`${BASE}/events/${EV}/play`, { waitUntil: 'networkidle' });
  await p.waitForSelector('.chal-card');
  await shot(p, 'board');
  // challenge modal (description) + attempts
  await p.getByText('Login Bypass', { exact: true }).click();
  await p.waitForSelector('.modal .chal-tabs');
  await shot(p, 'challenge');
  await p.getByRole('button', { name: 'Your attempts' }).click();
  await shot(p, 'challenge-attempts');
  await c.close();

  // Player, PL: board (i18n demo)
  c = await ctx(browser, { lang: 'pl', ...player });
  p = await c.newPage();
  await p.goto(`${BASE}/events/${EV}/play`, { waitUntil: 'networkidle' });
  await p.waitForSelector('.chal-card');
  await shot(p, 'board-pl');
  await c.close();

  // Player, EN: scoreboard
  c = await ctx(browser, { lang: 'en', ...player });
  p = await c.newPage();
  await p.goto(`${BASE}/events/${EV}/scoreboard`, { waitUntil: 'networkidle' });
  await p.waitForSelector('svg');
  await shot(p, 'scoreboard');
  await c.close();

  // Admin, EN
  c = await ctx(browser, { lang: 'en', ...admin });
  p = await c.newPage();
  await p.goto(`${BASE}/events/${EV}/admin`, { waitUntil: 'networkidle' });
  // stats tab is default
  await p.waitForSelector('.stat-strip');
  await shot(p, 'admin-stats');
  // challenges tab
  await p.getByRole('button', { name: 'Challenges' }).click();
  await p.waitForSelector('table');
  await p.waitForTimeout(400);
  // select all -> match banner + bulk bar
  await p.locator('thead input[type=checkbox]').check();
  await p.waitForTimeout(300);
  await shot(p, 'admin-challenges');
  // open create drawer
  await p.getByRole('button', { name: 'New challenge' }).click();
  await p.waitForSelector('.drawer');
  await shot(p, 'admin-create');
  await p.locator('.drawer .btn-close').click();
  // teams tab
  await p.getByRole('button', { name: 'Teams' }).click();
  await p.waitForSelector('table');
  await shot(p, 'admin-teams');
  // notifications tab
  await p.getByRole('button', { name: 'Notifications' }).click();
  await p.waitForSelector('textarea');
  await shot(p, 'admin-notifications');
  await c.close();

  await browser.close();
  console.log('done');
};
run().catch((e) => { console.error(e); process.exit(1); });
