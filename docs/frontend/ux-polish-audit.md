# Frontend UX Polish Audit

Date: 2026-07-16  
Scope: Frontend UX Polish & Usability Hardening v1

This audit covers the existing Next.js web app without changing its design system or product architecture. “Implemented” means addressed in this pass; “Follow-up” identifies a bounded improvement that should use the shared primitives introduced here.

| Location | Current UX problem | Proposed fix | Priority | Implemented |
| --- | --- | --- | --- | --- |
| Trip detail initial load | A single text row left most of the page blank and caused a large layout shift. | Render a labeled page skeleton with header and independent card shapes. | High | Yes |
| Trip detail load failure | The API error message was displayed directly and recovery required leaving the page. | Use contextual copy, hide technical detail outside development, and offer Retry and Trips actions. | High | Yes |
| Trip detail empty itinerary | DRAFT/FAILED copy referenced backend services and gave no direct next step. | Explain the user outcome and connect the empty-state action to Generate itinerary. | High | Yes |
| Trip Command Center | Source queries resolve independently, but the surrounding trip load had no progressive placeholder. | Keep the header-first data flow and use page/card skeletons before the trip payload arrives. | High | Yes |
| Trip Health | Loading was plain text; failure exposed `error.message`; no retry was available. | Use section skeletons and a local, retryable error that leaves the rest of trip detail usable. | High | Yes |
| Trip Health issues | A healthy trip showed “No issues match this filter,” which reads like a filter result rather than success. | Show a positive, review-oriented empty state when the source issue list is empty. | High | Yes |
| Group Readiness | Loading and errors were plain text/raw and not recoverable in place. | Use progressive skeletons and a card-level Retry action. | High | Yes |
| Budget | “No budget set” was buried in summary rows and had no prominent action. | Add a permission-aware Set budget empty state while retaining useful estimated totals. | High | Yes |
| Budget Confidence | A generic string replaced the entire card on load/error and could expose backend text. | Use a local skeleton and contextual Retry error; clarify that the saved budget is unchanged. | High | Yes |
| Budget form | Validation was not associated with fields and mobile actions could move below the viewport. | Add localized field errors/hints and a sticky mobile Save/Cancel bar. | High | Yes |
| Expenses | Empty content was one sentence with actions separated at the top of a dense section. | Use an action-oriented, permission-aware empty state for Add expense and Upload receipt. | High | Yes |
| Expense deletion | Delete happened from the row without a confirmation. | Explain the effect on totals and attached receipts before performing the mutation. | High | Yes |
| Receipt deletion | Delete happened without explaining whether the linked expense was affected. | Use a shared destructive dialog that states the expense remains. | High | Yes |
| Receipt upload | File input had no visible label, selected filename, size guidance, early validation, or privacy warning. | Label the field, validate type/size on selection, show filename and supported limits, and add privacy copy. | High | Yes |
| Checklist | Loading/error/empty states were plain blocks; the empty state did not make permissions explicit. | Use shared loading/error/empty states, Retry, and an editor-only generation action. | High | Yes |
| Checklist progress | Progress was conveyed visually without progress semantics. | Add a labeled `progressbar` with current/min/max values. | High | Yes |
| Reminders | Existing localized empty and mutation states are useful, but delete still uses a browser confirm. | Migrate reminder deletion to `ConfirmDialog` in a follow-up consistency pass. | Medium | Follow-up |
| Route Builder | Desktop and mobile shared one dense sticky action block; discarding changes used browser confirms. | Keep desktop controls, use the shared mobile action bar, and use the unsaved-changes dialog for in-app navigation/discard. | High | Yes |
| Route deep links | Route leg/stop links scrolled but did not focus/highlight or explain a missing target. | Retry target resolution, focus and highlight matches, then show a friendly message if absent. | High | Yes |
| Budget/Health/Expense/Activity deep links | Query parameters had no stable target IDs. | Add stable target IDs and central deep-link mapping for category, issue, expense, and event targets. | High | Yes |
| Checklist assignee deep link | The checklist supports “mine” but not an arbitrary assignee filter. | Add a permission-safe assignee filter before enabling `assignedTo` targeting. | Medium | Follow-up |
| Comment deep link | Item comments open in a contextual panel and do not yet expose a page-level stable comment target. | Resolve comment-to-item metadata, open the panel, and focus the matching comment. | Medium | Follow-up |
| Public share loading | A single text block caused layout shift. | Use the standard page skeleton within the existing warm shell. | High | Yes |
| Public share unavailable/expired | Expired, disabled, and missing links shared vague copy. | Render dedicated expired copy when status metadata identifies expiry; otherwise use a recoverable unavailable state. | High | Yes |
| Public share password | Wrong-password handling could surface an API message; labels were hardcoded. | Use localized, non-technical feedback and localized password/unlock copy. | High | Yes |
| Public share empty itinerary | Plain text did not clearly frame owner responsibility/read-only behavior. | Use a localized read-only empty state and consistent provider disclaimer. | High | Yes |
| Public share privacy | The current page omits private command center, readiness, budgets, expenses, receipts, approvals, policy, activity, and comments. | Preserve the sanitized API and public-only component boundary. | High | Yes (verified) |
| Auth fields | Error text was visible but not referenced by its input. API errors could expose server copy. | Connect errors with `aria-describedby` and use safe localized login/register failure messages. | High | Yes |
| Settings loading/error | Settings had a tailored skeleton but raw load errors and no retry. | Retain the existing skeleton; use a shared localized Retry error with dev-only detail. | High | Yes |
| Version restore | Browser confirm copy omitted some unaffected trip data and had no focus management. | Use an accessible confirmation that explains a new version is created and comments/expenses/receipts/collaborators remain. | High | Yes |
| Public share disable | Browser confirm did not state immediate access loss or reversibility. | Use the standard confirmation copy and pending state. | High | Yes |
| Offline settings clear | Browser confirm varied by screen and lacked focus management. | Use one localized confirmation stating device-only deletion and cloud-data preservation. | High | Yes |
| Offline trips page | Loading/error/empty states were custom text; load errors exposed raw storage messages. | Use shared progressive, contextual, retryable, and action-oriented states. | High | Yes |
| Offline per-trip removal/discard | Per-item actions still use browser confirms and an alert. | Migrate to `ConfirmDialog`; keep pending-change safeguards. | Medium | Follow-up |
| AI generation button | Error fallback could show a backend message; loading had text only. | Use a width-stable disabled button with spinner and contextual retryable error. | High | Yes |
| AI generation job | Running state did not announce progress and failed jobs printed `errorMessage`. | Add accessible progress, staged user copy, completed-with-warnings state, and reference code instead of raw error. | High | Yes |
| Approval dialogs | Several dialogs duplicate shells and do not consistently trap/restore focus. | Migrate approval shells to the shared dialog foundation without changing workflow. | High | Follow-up |
| Workspace/template destructive actions | Multiple browser confirms use inconsistent copy. | Migrate archive/remove actions to `ConfirmDialog` by feature. | Medium | Follow-up |
| Repeated badges | Feature badges generally include text, but tone/layout implementations are duplicated. | Use `StatusBadge` for new badges and migrate existing badges only when touched. | Medium | Foundation added |
| Mobile dense tables | Transport comparison and analytics tables can still require horizontal scrolling. | Preserve tables where comparison is essential; add card alternatives at narrow viewports. | Medium | Follow-up |
| Toast/inline feedback | Success/error banners are implemented independently and some duplicate mutation messages. | Keep critical errors inline; standardize short success copy during feature migrations. | Medium | Partial |
| New shared UI copy | Several feature namespaces were incomplete across locales. | Add matching `loading`, `emptyStates`, `errors`, `confirmations`, `forms`, `accessibility`, `offline`, and `publicShare` keys in all four catalogs. | High | Yes |

## Highest-impact result

The private trip, public share, budget, expenses/receipts, checklist, route editing, generation, auth, settings, and offline entry points now have a consistent recovery model. Remaining items are migrations of lower-risk duplicated patterns rather than missing foundations.
