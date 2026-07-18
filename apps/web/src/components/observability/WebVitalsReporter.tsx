"use client";

import { useReportWebVitals } from "next/web-vitals";

const endpoint = process.env.NEXT_PUBLIC_WEB_VITALS_ENDPOINT?.trim();

export function WebVitalsReporter() {
  useReportWebVitals((metric) => {
    const detail = {
      id: metric.id,
      name: metric.name,
      value: metric.value,
      rating: metric.rating,
      navigationType: metric.navigationType,
      routeGroup: routeGroup(window.location.pathname)
    };
    window.dispatchEvent(new CustomEvent("travel-ai:web-vital", { detail }));
    if (process.env.NODE_ENV === "development") {
      // Low-cardinality route pattern only; never log a trip, share token, or query.
      console.debug("[web-vital]", detail);
    }
    if (endpoint && navigator.sendBeacon) {
      navigator.sendBeacon(endpoint, new Blob([JSON.stringify(detail)], { type: "application/json" }));
    }
  });
  return null;
}

function routeGroup(pathname: string) {
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length === 0) {
    return "home";
  }

  const [root, child, leaf] = segments;
  if (["trips", "templates", "workspaces", "share"].includes(root) && child) {
    const normalizedChild = root === "trips" && child === "new" ? "new" : ":id";
    return [root, normalizedChild, leaf].filter(Boolean).join("/");
  }

  // Top-level routes are app-owned literals. Never add arbitrary later path
  // segments, which could include opaque public-share tokens or user content.
  return root;
}
