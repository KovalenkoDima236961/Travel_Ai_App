import type { QueryClient } from "@tanstack/react-query";

export const OPS_REFRESH_INTERVAL = 20_000;

export function withReason(message: string, action: (reason: string) => void) {
  const reason = window.prompt(`${message}\n\nReason:`);
  if (reason?.trim()) {
    action(reason.trim());
  }
}

export function invalidateOps(queryClient: QueryClient) {
  return queryClient.invalidateQueries({ queryKey: ["ops"] });
}

export function shortId(value?: string | null) {
  if (!value) {
    return "-";
  }
  return value.length > 12 ? `${value.slice(0, 8)}...` : value;
}

export function formatOpsDate(value?: string | null) {
  if (!value) {
    return "-";
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

export function undefinedValue() {
  return "";
}
