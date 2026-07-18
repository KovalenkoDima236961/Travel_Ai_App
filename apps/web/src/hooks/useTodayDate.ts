"use client";

import { useMemo } from "react";

// The browser supplies the user's local calendar date. The backend never
// infers it from tracking or location data.
export function useTodayDate() {
  return useMemo(() => {
    const now = new Date();
    const offset = now.getTimezoneOffset() * 60_000;
    return new Date(now.getTime() - offset).toISOString().slice(0, 10);
  }, []);
}
