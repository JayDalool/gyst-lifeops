// Cache only public static assets. Private pages, API responses and redirects
// never enter CacheStorage, including pages viewed before signing out.
const CACHE = "lifeops-shell-v91-signin";
const SHELL = ["/static/css/styles.css?v=aurora-1", "/static/css/ui-tokens.css?v=aurora-1", "/static/css/ui-accents.css?v=aurora-1", "/static/css/ui-pins.css?v=aurora-1", "/static/css/ui-components.css?v=icons-1.aurora-1", "/static/css/ui-shell.css?v=projects-nav-1.launcher-5.aurora-1.galaxy-bar-1", "/static/css/ui-nav.css?v=nav-8", "/static/css/ui-today.css?v=now-1.aurora-1", "/static/css/ui-projects.css?v=projects-nav-1.aurora-1", "/static/css/ui-galaxy.css?v=galaxywin-4", "/static/css/ui-health.css?v=weight-1.aurora-1", "/static/css/ui-training.css?v=train-2.aurora-1", "/static/css/ui-health-v3.css?v=hv3-13", "/static/css/ui-courses.css?v=ui-10.aurora-1", "/static/js/theme.js?v=appearance-2", "/static/js/charts.js?v=ui-2.aurora-1", "/static/js/app.js?v=launcher-5.now-1.dayends-1", "/static/js/launcher.js?v=launcher-5", "/static/js/nav.js?v=nav-4", "/static/js/completion-toast.js?v=toast-1","/static/js/ui-overlay.js?v=now-1", "/static/favicon.svg", "/static/icons.svg?v=r7-6", "/static/fonts/Geist-Variable.woff2?v=1.3.0"];
self.addEventListener("install", event => event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(SHELL)).then(() => self.skipWaiting())));
self.addEventListener("activate", event => event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(key => key.startsWith("lifeops-shell-") && key !== CACHE).map(key => caches.delete(key)))).then(() => self.clients.claim())));
// GYST-188 (a4, Jay's answer 30): a plain page with Try again. One constant with no request data, no script, no src and no outside reference: it is built while the network is down, so it carries its own style and icon, in Aurora's literal colours by OS scheme (the stored theme needs theme.js, which cannot load). It is not cached and holds nothing private.
const OFFLINE = '<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"><meta name="color-scheme" content="light dark"><title>Offline · GYST</title><style>:root{color-scheme:light dark;--bg:#F1EFF9;--card:#FFFFFF;--ink:#0D0C16;--ink2:#4D4961;--line:rgb(11 13 18 / 8%);--circle:#FDF1E1;--icon:#A64C07;--btn:#6163F2;--onbtn:#FFFFFF}@media (prefers-color-scheme:dark){:root{--bg:#09080F;--card:#13111F;--ink:#F3F2F5;--ink2:#A7A5B6;--line:rgb(255 255 255 / 7%);--circle:#33260C;--icon:#FCD062;--btn:#96A2FE;--onbtn:#0B0D12}}html{background:var(--bg)}body{margin:0;min-height:100vh;min-height:100dvh;box-sizing:border-box;display:grid;grid-template-rows:1fr auto 2fr;justify-items:center;padding:24px 16px;background:var(--bg);color:var(--ink);font:14px/19px -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}main{grid-row:2;width:100%;max-width:420px;box-sizing:border-box;background:var(--card);border-radius:20px;box-shadow:inset 0 0 0 1px var(--line);padding:36px 16px 28px;display:grid;justify-items:center;text-align:center;gap:8px}.i{width:64px;height:64px;border-radius:50%;display:grid;place-items:center;margin-bottom:6px;background:var(--circle);color:var(--icon)}h1{margin:0;font-size:20px;line-height:25px;font-weight:600;letter-spacing:-.02em}p{margin:0;max-width:40ch;color:var(--ink2)}a{display:inline-flex;align-items:center;justify-content:center;min-height:44px;margin-top:10px;padding:0 16px;border-radius:12px;background:var(--btn);color:var(--onbtn);font:500 16px/1 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;text-decoration:none}a:focus-visible{outline:2px solid var(--btn);outline-offset:2px}</style></head><body><main><span class="i"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M3 3l18 18M5 12.500a10 10 0 0 1 4-2.600M19 12.500a10 10 0 0 0-5-3M8.500 16a5 5 0 0 1 7 0M12 19.500v.500"/></svg></span><h1>You are offline</h1><p>GYST needs a connection to open this page. Connect, then try again.</p><a href="">Try again</a></main></body></html>';
self.addEventListener("fetch", event => {
  const request = event.request;
  const url = new URL(request.url);
  if (request.method !== "GET" || url.origin !== self.location.origin || request.headers.has("Authorization")) return;
  if (request.mode === "navigate") {
    // GYST-188: navigations stay network-only (docs/audit/vnext/01_CURRENT_SYSTEM.md section 11): cache "no-store" makes the browser skip its HTTP cache, which otherwise answers a 308 or a plain-text 404 without Cache-Control from disk (measured with the worker active). That init makes the Fetch spec turn the request's mode from navigate into same-origin, so the server sees Sec-Fetch-Mode: same-origin, Dest: empty and Cache-Control: no-cache; state_pages.go accepts exactly that signature for the styled 404 and 500 pages. Re-fetching the original request instead keeps navigate but lets the HTTP cache answer, and navigation preload too.
    event.respondWith(fetch(request, {cache: "no-store"}).catch(() => new Response(OFFLINE, {status: 503, headers: {"Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store"}})));
    return;
  }
  if (!url.pathname.startsWith("/static/")) return;
  event.respondWith(fetch(request).then(async response => {
    if (response.ok && response.type === "basic" && !response.redirected && !response.headers.get("Cache-Control")?.includes("no-store") && !response.headers.get("Content-Type")?.includes("text/html")) {
      const cache = await caches.open(CACHE);
      await cache.put(request, response.clone());
    }
    return response;
  }).catch(async () => (await caches.match(request, {cacheName: CACHE})) || Response.error()));
});
self.addEventListener("push", event => {
  let data = {title:"Prayer time", body:"It is time for prayer.", url:"/today", tag:"prayer"};
  if (event.data) { try { data = Object.assign(data, event.data.json()); } catch (_) { data.body = event.data.text(); } }
  event.waitUntil(self.registration.showNotification(data.title, {body:data.body, icon:"/static/favicon.svg", badge:"/static/favicon.svg", tag:data.tag, data:{url:data.url}}));
});
self.addEventListener("notificationclick", event => {
  event.notification.close();
  event.waitUntil(clients.matchAll({type:"window", includeUncontrolled:true}).then(windows => { for (const client of windows) { if ("focus" in client) { client.navigate(event.notification.data.url); return client.focus(); } } return clients.openWindow(event.notification.data.url); }));
});
