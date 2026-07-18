/** Public, non-sensitive build metadata embedded in the Web App bundle. */
function publicValue(name: "NEXT_PUBLIC_APP_VERSION" | "NEXT_PUBLIC_GIT_SHA" | "NEXT_PUBLIC_BUILD_TIME", fallback: string) {
  const value = process.env[name]?.trim();
  return value || fallback;
}

export const appVersion = publicValue("NEXT_PUBLIC_APP_VERSION", "dev");
export const gitSHA = publicValue("NEXT_PUBLIC_GIT_SHA", "unknown");
export const buildTime = publicValue("NEXT_PUBLIC_BUILD_TIME", "unknown");

export const shortGitSHA = gitSHA === "unknown" ? gitSHA : gitSHA.slice(0, 12);

export const webVersionInfo = {
  service: "web-app",
  version: appVersion,
  gitSha: gitSHA,
  buildTime,
  environment: process.env.NEXT_PUBLIC_APP_ENV?.trim() || "local",
  apiContractVersion: appVersion
} as const;
