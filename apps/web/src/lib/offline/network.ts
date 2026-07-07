import { ApiError } from "@/shared/api/client";

export function isBrowserOnline() {
  if (typeof navigator === "undefined") {
    return true;
  }

  return navigator.onLine;
}

export function isOfflineLikeError(error: unknown) {
  if (!isBrowserOnline()) {
    return true;
  }

  if (error instanceof ApiError) {
    return error.status === 0;
  }

  if (error instanceof TypeError) {
    return true;
  }

  return error instanceof Error && /failed to fetch|network|load failed/i.test(error.message);
}
