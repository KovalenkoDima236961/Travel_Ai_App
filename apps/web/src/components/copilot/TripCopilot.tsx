"use client";

import dynamic from "next/dynamic";
import { useState } from "react";
import { CopilotButton } from "./CopilotButton";
import type { CopilotClientContext } from "@/types/copilot";

const CopilotPanel = dynamic(
  () => import("./CopilotPanel").then((module) => module.CopilotPanel),
  {
    ssr: false,
    loading: () => (
      <div className="fixed inset-0 z-50 bg-cocoa-900/25" aria-label="Loading Copilot" />
    )
  }
);

export function TripCopilot({ tripId, currentTab, currentPath, clientContext, openOnMount = false }: { tripId: string; currentTab?: string; currentPath?: string; clientContext?: CopilotClientContext; openOnMount?: boolean }) {
	const [open, setOpen] = useState(openOnMount);
  return (
    <>
      <CopilotButton onClick={() => setOpen(true)} />
      {open ? (
        <CopilotPanel
          clientContext={clientContext}
          currentPath={currentPath}
          currentTab={currentTab}
          onClose={() => setOpen(false)}
          open
          tripId={tripId}
        />
      ) : null}
    </>
  );
}
