"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { registerServiceWorker } from "@/lib/push/register-service-worker";

export type ServiceWorkerUpdateState = {
  updateAvailable: boolean;
  waitingWorker: ServiceWorker | null;
  refreshing: boolean;
  applyUpdate: () => void;
};

export function useServiceWorkerUpdate(): ServiceWorkerUpdateState {
  const [waitingWorker, setWaitingWorker] = useState<ServiceWorker | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const refreshingRef = useRef(false);

  useEffect(() => {
    if (typeof window === "undefined" || !("serviceWorker" in navigator)) {
      return;
    }

    let registration: ServiceWorkerRegistration | null = null;

    function trackInstallingWorker(worker: ServiceWorker | null | undefined) {
      if (!worker) {
        return;
      }

      worker.addEventListener("statechange", () => {
        if (worker.state === "installed" && navigator.serviceWorker.controller) {
          setWaitingWorker(worker);
        }
      });
    }

    registerServiceWorker()
      .then((nextRegistration) => {
        registration = nextRegistration;
        if (nextRegistration.waiting && navigator.serviceWorker.controller) {
          setWaitingWorker(nextRegistration.waiting);
        }
        trackInstallingWorker(nextRegistration.installing);

        nextRegistration.addEventListener("updatefound", () => {
          trackInstallingWorker(nextRegistration.installing);
        });
      })
      .catch(() => {
        // Service worker updates are unavailable in this browser/session.
      });

    function handleControllerChange() {
      if (!refreshingRef.current) {
        return;
      }
      window.location.reload();
    }

    navigator.serviceWorker.addEventListener("controllerchange", handleControllerChange);

    return () => {
      navigator.serviceWorker.removeEventListener("controllerchange", handleControllerChange);
      registration = null;
    };
  }, []);

  const applyUpdate = useCallback(() => {
    if (!waitingWorker) {
      return;
    }

    refreshingRef.current = true;
    setRefreshing(true);
    waitingWorker.postMessage({ type: "SKIP_WAITING" });
  }, [waitingWorker]);

  return {
    updateAvailable: Boolean(waitingWorker),
    waitingWorker,
    refreshing,
    applyUpdate
  };
}
