export function publicShareTokenKey(shareToken: string) {
  return `public-share-access-token:${shareToken}`;
}

export function publicShareTokenExpiryKey(shareToken: string) {
  return `public-share-access-token-exp:${shareToken}`;
}

export function clearStoredPublicShareToken(shareToken: string) {
  if (typeof window === "undefined") {
    return;
  }
  sessionStorage.removeItem(publicShareTokenKey(shareToken));
  sessionStorage.removeItem(publicShareTokenExpiryKey(shareToken));
}
