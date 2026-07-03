let registrationPromise: Promise<ServiceWorkerRegistration> | null = null;

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration> {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
    throw new Error("Service workers are not supported in this browser.");
  }

  registrationPromise ??= navigator.serviceWorker.register("/sw.js").catch((error) => {
    registrationPromise = null;
    throw error;
  });
  return registrationPromise;
}
