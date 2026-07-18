"use client";

import { useEffect, useState } from "react";

/** Keeps background polling dormant without coupling feature hooks to the DOM. */
export function useDocumentVisibility() {
  const [visible, setVisible] = useState(
    () => typeof document === "undefined" || document.visibilityState !== "hidden"
  );

  useEffect(() => {
    const updateVisibility = () => setVisible(document.visibilityState !== "hidden");
    updateVisibility();
    document.addEventListener("visibilitychange", updateVisibility);
    return () => document.removeEventListener("visibilitychange", updateVisibility);
  }, []);

  return visible;
}
