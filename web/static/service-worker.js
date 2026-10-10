// Cache only public static assets. Private pages, API responses and redirects
// never enter CacheStorage, including pages viewed before signing out.
const CACHE = "lifeops-shell-v64-states";
const SHELL = ["/static/css/styles.css?v=appearance-1", "/static/css/ui-tokens.css?v=vn-2", "/static/css/ui-components.css?v=states-1", "/static/css/ui-shell.css?v=motion-1", "/static/css/ui-today.css?v=ui-2", "/static/css/ui-projects.css?v=ui-4", "/static/css/ui-galaxy.css?v=galaxywin-1", "/static/css/ui-health.css?v=weight-1", "/static/css/ui-training.css?v=train-2", "/static/css/ui-health-v3.css?v=hv3-9", "/static/css/ui-courses.css?v=ui-10", "/static/js/theme.js?v=appearance-1", "/static/js/charts.js?v=ui-2", "/static/js/app.js?v=offline-1", "/static/js/completion-toast.js?v=toast-1","/static/favicon.svg"];
self.addEventListener("install", event => event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(SHELL)).then(() => self.skipWaiting())));
self.addEventListener("activate", event => event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(key => key.startsWith("lifeops-shell-") && key !== CACHE).map(key => caches.delete(key)))).then(() => self.clients.claim())));
const OFFLINE = '<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>GYST · Offline</title><body><main><h1>You’re offline</h1><p>Connect to the internet to view your GYST data.</p><p><a href="">Try again</a></p></main></body></html>';
self.addEventListener("fetch", event => {
  const request = event.request;
  const url = new URL(request.url);
  if (request.method !== "GET" || url.origin !== self.location.origin || request.headers.has("Authorization")) return;
  if (request.mode === "navigate") {
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
