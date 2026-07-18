"use client";

import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";

const SEG_BASE =
  "h-11 rounded-full px-4 text-[13.5px] transition disabled:cursor-not-allowed disabled:opacity-40";
const SEG_ACTIVE = "bg-cocoa-900 font-semibold text-sand-150";
const SEG_IDLE = "font-medium text-cocoa-500 hover:bg-sand-200";

/**
 * Hero scope filter. A second view of the same `selectionValue` the header
 * ScopeSelect drives, so the two never drift. "Workspaces" is workspace-mode:
 * active whenever a workspace is selected, and clicking it selects the current
 * (or first) workspace — the header picker refines *which* one.
 */
export function ScopeSegmentedControl() {
  const {
    currentScope,
    currentWorkspaceId,
    workspaces,
    setAllTrips,
    setPersonalTrips,
    setCurrentWorkspace
  } = useWorkspaces();

  const hasWorkspaces = workspaces.length > 0;

  function selectWorkspaceScope() {
    const target = currentWorkspaceId ?? workspaces[0]?.id;
    if (target) {
      setCurrentWorkspace(target);
    }
  }

  return (
    <div className="inline-flex max-w-full items-center gap-1 overflow-x-auto rounded-full border border-sand-300 bg-white p-1 [scrollbar-width:thin]">
      <button
        type="button"
        onClick={setAllTrips}
        className={`${SEG_BASE} ${currentScope === "all" ? SEG_ACTIVE : SEG_IDLE}`}
      >
        All
      </button>
      <button
        type="button"
        onClick={setPersonalTrips}
        className={`${SEG_BASE} ${currentScope === "personal" ? SEG_ACTIVE : SEG_IDLE}`}
      >
        Personal
      </button>
      <button
        type="button"
        onClick={selectWorkspaceScope}
        disabled={!hasWorkspaces}
        className={`${SEG_BASE} ${currentScope === "workspace" ? SEG_ACTIVE : SEG_IDLE}`}
      >
        Workspaces
      </button>
    </div>
  );
}
