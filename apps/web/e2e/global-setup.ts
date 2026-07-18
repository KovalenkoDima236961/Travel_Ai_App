import type { FullConfig } from "@playwright/test";

export default async function globalSetup(config: FullConfig) {
  const project = config.projects[0];
  const baseURL = String(project?.use.baseURL ?? "http://127.0.0.1:3000").replace(/\/+$/, "");
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 10_000);

  try {
    const response = await fetch(`${baseURL}/api/ready`, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`readiness returned HTTP ${response.status}`);
    }
  } catch (error) {
    throw new Error(
      `The Playwright stack is not ready at ${baseURL}. Run ../../scripts/test-stack-up.sh first. ${error instanceof Error ? error.message : String(error)}`
    );
  } finally {
    clearTimeout(timeout);
  }
}
