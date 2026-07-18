export type FeatureFlagKey =
  | "ai_generation_enabled"
  | "ai_repair_enabled"
  | "copilot_enabled"
  | "route_alternatives_enabled"
  | "template_adaptation_enabled"
  | "public_sharing_enabled"
  | "data_exports_enabled"
  | "real_providers_enabled"
  | "calendar_sync_enabled"
  | "availability_search_enabled"
  | "transport_search_enabled"
  | "receipt_ocr_enabled"
  | "workspace_approvals_enabled"
  | "policy_repair_enabled"
  | "web_push_enabled"
  | "email_notifications_enabled"
  | "notification_digests_enabled"
  | "offline_mode_enabled"
  | "ops_dashboard_enabled";

export type PublicFeatureFlagsResponse = {
  flags: Partial<Record<FeatureFlagKey, boolean>>;
  environment: string;
  updatedAt?: string | null;
};

const riskyFlags: FeatureFlagKey[] = [
  "ai_repair_enabled", "public_sharing_enabled", "real_providers_enabled",
  "calendar_sync_enabled", "receipt_ocr_enabled", "web_push_enabled", "ops_dashboard_enabled"
];

export function fallbackFeatureFlags(): Record<FeatureFlagKey, boolean> {
  const production = (process.env.NEXT_PUBLIC_APP_ENV ?? "local").toLowerCase() === "production";
  const flags = Object.fromEntries(riskyFlags.map((key) => [key, false])) as Record<FeatureFlagKey, boolean>;
  flags.ai_generation_enabled = !production;
  flags.copilot_enabled = !production;
  flags.route_alternatives_enabled = !production;
  flags.template_adaptation_enabled = !production;
  flags.data_exports_enabled = !production;
  flags.availability_search_enabled = !production;
  flags.transport_search_enabled = !production;
  flags.workspace_approvals_enabled = !production;
  flags.policy_repair_enabled = false;
  flags.email_notifications_enabled = !production;
  flags.notification_digests_enabled = !production;
  flags.offline_mode_enabled = true;
  return flags;
}
