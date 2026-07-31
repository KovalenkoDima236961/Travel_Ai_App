# Add a Trip Workspace action

1. Add a stable action ID, localization key, group, and availability rule in `apps/web/src/lib/trip-workspace/actions.ts`.
2. Use backend capabilities, not role-name inference, where a capability exists. Add the matching backend authorization before exposing a new mutation.
3. Use canonical internal routes only; never accept a user-provided absolute URL. Keep domain forms outside the registry.
4. Mark destructive actions and use `ConfirmDialog` with reversible-impact copy. Provide an offline/read-only/archived disabled reason.
5. Emit `trip_workspace_quick_action_used` with only action ID, role, and lifecycle. Never log user-authored labels or content.
6. Test owner/editor/viewer, archived, offline, and feature-disabled filtering in the registry and component.
