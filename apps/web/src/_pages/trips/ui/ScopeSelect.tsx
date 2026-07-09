"use client";

import { useWorkspaces } from "@/components/workspaces/WorkspaceProvider";
import { BriefcaseIcon, ChevronDownIcon } from "./icons";

/**
 * Header scope picker. A styled native <select> bound to the single
 * `selectionValue`/`setSelectionValue` source of truth in WorkspaceProvider, so
 * keyboard nav and click-outside come for free and it never drifts from the
 * hero segmented control (both are views of the same scope state).
 */
export function ScopeSelect() {
  const { workspaces, selectionValue, setSelectionValue, isLoading } = useWorkspaces();

  return (
    <div className="relative inline-flex items-center">
      <BriefcaseIcon className="pointer-events-none absolute left-3.5 h-[15px] w-[15px] text-cocoa-400" />
      <select
        aria-label="Filter trips by scope"
        disabled={isLoading}
        value={selectionValue}
        onChange={(event) => setSelectionValue(event.target.value)}
        className="h-[38px] appearance-none rounded-full border border-sand-400 bg-white pl-9 pr-9 text-[13.5px] font-medium text-cocoa-700 transition hover:border-sand-600 focus:outline-none focus:ring-2 focus:ring-clay/40 disabled:cursor-not-allowed disabled:opacity-60"
      >
        <option value="all">All trips</option>
        <option value="personal">Personal</option>
        {workspaces.map((workspace) => (
          <option key={workspace.id} value={workspace.id}>
            {workspace.name}
          </option>
        ))}
      </select>
      <ChevronDownIcon className="pointer-events-none absolute right-3.5 h-[13px] w-[13px] text-cocoa-400" />
    </div>
  );
}
