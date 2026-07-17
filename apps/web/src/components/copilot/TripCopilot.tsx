"use client";

import { useState } from "react";
import { CopilotButton } from "./CopilotButton";
import { CopilotPanel } from "./CopilotPanel";

export function TripCopilot({ tripId, currentTab, currentPath }: { tripId: string; currentTab?: string; currentPath?: string }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <CopilotButton onClick={() => setOpen(true)} />
      <CopilotPanel currentPath={currentPath} currentTab={currentTab} onClose={() => setOpen(false)} open={open} tripId={tripId} />
    </>
  );
}
