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
    if (endpoint && navigator.sendBeacon) {
      navigator.sendBeacon(endpoint, new Blob([JSON.stringify(detail)], { type: "application/json" }));
    }
  });
  return null;
}

function routeGroup(pathname: string) {
  return pathname
    .split("/")
    .filter(Boolean)
    .map((segment) =>
      /^[0-9a-f]{8}-[0-9a-f-]{27,}$/i.test(segment) || /^\d+$/.test(segment)
        ? ":id"
        : segment
    )
    .slice(0, 3)
    .join("/") || "home";
}

