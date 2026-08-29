const CACHE = "lifeops-shell-v6";
const SHELL = ["/today", "/galaxy", "/projects", "/static/css/styles.css?v=design-2", "/static/js/app.js?v=2", "/static/js/galaxy.js?v=6", "/static/js/vendor/three.module.js", "/static/js/vendor/three.core.js", "/static/images/oliver-galaxy-pixel.png", "/static/favicon.svg"];
self.addEventListener("install", event => event.waitUntil(caches.open(CACHE).then(cache => cache.addAll(SHELL)).then(() => self.skipWaiting())));
self.addEventListener("activate", event => event.waitUntil(caches.keys().then(keys => Promise.all(keys.filter(key => key !== CACHE).map(key => caches.delete(key)))).then(() => self.clients.claim())));
self.addEventListener("fetch", event => {
  if (event.request.method !== "GET" || new URL(event.request.url).origin !== location.origin) return;
  event.respondWith(fetch(event.request).then(response => { const copy = response.clone(); caches.open(CACHE).then(cache => cache.put(event.request, copy)); return response; }).catch(() => caches.match(event.request).then(response => response || caches.match("/today"))));
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
