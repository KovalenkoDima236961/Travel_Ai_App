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
