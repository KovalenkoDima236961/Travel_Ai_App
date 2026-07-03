"use client";

import { useEffect, useState } from "react";

export function useNetworkStatus() {
  const [online, setOnline] = useState(getInitialOnline);
  const [wasOffline, setWasOffline] = useState(() => !getInitialOnline());

  useEffect(() => {
    function handleOnline() {
      setOnline(true);
    }

    function handleOffline() {
      setWasOffline(true);
      setOnline(false);
    }

    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, []);

  return { online, wasOffline };
}

function getInitialOnline() {
  if (typeof navigator === "undefined") {
    return true;
  }

  return navigator.onLine;
}
