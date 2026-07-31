# Create a Trip Workspace deep link

1. Choose a section/view and build it with `buildTripWorkspaceHref`; use canonical entity keys (`item`, `leg`, `stop`, `expense`, `receipt`, `comment`, `member`, `reminder`, `decision`, `version`, `event`, `issue`).
2. Give the destination a unique, deterministic DOM ID and scroll margin. Add it to `getDeepLinkDomTarget` with any required legacy alias.
3. Load the target only after trip authorization and feature/capability resolution. Do not reveal metadata in forbidden/not-found states.
4. Verify focus, non-color highlight, reduced motion, timeout/user-interaction cleanup, deleted entity fallback, offline behavior, and mobile browser back.
5. Update Global Search/notification producers to emit the canonical route; retain the old alias in the compatibility table and tests.
