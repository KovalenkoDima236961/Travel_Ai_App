self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(APP_SHELL_CACHE)
      .then((cache) => cache.addAll(APP_SHELL_URLS))
      .catch(() => undefined)
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((key) => key !== APP_SHELL_CACHE)
            .map((key) => caches.delete(key))
        )
      )
      .then(() => clients.claim())
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") {
    return;
  }

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) {
    return;
  }

  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(async () => {
        const fallback = await caches.match("/offline");
        return fallback || Response.error();
      })
    );
    return;
  }

  if (url.pathname.startsWith("/_next/static/")) {
    event.respondWith(cacheFirst(request));
  }
});

self.addEventListener("push", (event) => {
  let payload = {};

  if (event.data) {
    try {
      payload = event.data.json();
    } catch {
      payload = {};
    }
  }

  const title = typeof payload.title === "string" && payload.title ? payload.title : "Travel App";
  const url = safeRelativeURL(payload.url);
  const notificationId =
    typeof payload.notificationId === "string" ? payload.notificationId : undefined;
  const type = typeof payload.type === "string" ? payload.type : undefined;

  event.waitUntil(
    self.registration.showNotification(title, {
      body: typeof payload.body === "string" ? payload.body : "",
      icon: typeof payload.icon === "string" ? payload.icon : undefined,
      badge: typeof payload.badge === "string" ? payload.badge : undefined,
      data: { url, notificationId },
      tag: notificationId || type || "travel-ai-notification"
    })
  );
});

const APP_SHELL_CACHE = "travel-ai-app-shell-v1";
const APP_SHELL_URLS = ["/offline"];

async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) {
    return cached;
  }

  const response = await fetch(request);
  if (response.ok) {
    const cache = await caches.open(APP_SHELL_CACHE);
    await cache.put(request, response.clone());
  }
  return response;
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const targetURL = safeRelativeURL(event.notification.data && event.notification.data.url);

  event.waitUntil(openOrFocusClient(targetURL));
});

async function openOrFocusClient(path) {
  const target = new URL(path, self.location.origin).href;
  const windowClients = await clients.matchAll({ type: "window", includeUncontrolled: true });

  for (const client of windowClients) {
    const clientURL = new URL(client.url);
    if (clientURL.origin !== self.location.origin) {
      continue;
    }
    if ("navigate" in client) {
      await client.navigate(target);
    }
    return client.focus();
  }

  return clients.openWindow(target);
}

function safeRelativeURL(value) {
  if (typeof value !== "string" || !value.trim()) {
    return "/notifications";
  }

  try {
    const parsed = new URL(value, self.location.origin);
    if (parsed.origin !== self.location.origin) {
      return "/notifications";
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return "/notifications";
  }
}
