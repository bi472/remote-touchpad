const CACHE_NAME = 'remote-touchpad-v6';
const ASSETS = [
  './',
  './index.html',
  './main.css',
  './icon.png',
  './icon.woff',
  './icon-192.png',
  './icon-512.png',
  './app/main.mjs',
  './app/ui.mjs',
  './app/inputcontroller.mjs',
  './app/keyboard.mjs',
  './app/mouse.mjs',
  './app/touchpad.mjs',
  './app/socket.mjs',
  './app/compat.mjs'
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(ASSETS);
    })
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    })
  );
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  if (url.pathname.endsWith('manifest.json')) {
    event.respondWith(fetch(event.request));
    return;
  }
  event.respondWith(
    caches.match(event.request).then((cachedResponse) => {
      if (cachedResponse) {
        return cachedResponse;
      }
      return fetch(event.request);
    })
  );
});
