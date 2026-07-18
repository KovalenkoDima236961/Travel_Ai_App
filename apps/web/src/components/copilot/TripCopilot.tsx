"use client";

import { useState } from "react";
import { CopilotButton } from "./CopilotButton";
import { CopilotPanel } from "./CopilotPanel";
import type { CopilotClientContext } from "@/types/copilot";

export function TripCopilot({ tripId, currentTab, currentPath, clientContext, openOnMount = false }: { tripId: string; currentTab?: string; currentPath?: string; clientContext?: CopilotClientContext; openOnMount?: boolean }) {
	const [open, setOpen] = useState(openOnMount);
  return (
    <>
      <CopilotButton onClick={() => setOpen(true)} />
      <CopilotPanel clientContext={clientContext} currentPath={currentPath} currentTab={currentTab} onClose={() => setOpen(false)} open={open} tripId={tripId} />
    </>
  );
}
