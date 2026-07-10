"use client";

type RouteAlternativeVoteControlsProps = {
  pollAvailable?: boolean;
};

export function RouteAlternativeVoteControls({ pollAvailable = false }: RouteAlternativeVoteControlsProps) {
  return (
    <div className="rounded-[12px] border border-sand-300 bg-sand-50 px-3 py-2 text-[12.5px] leading-5 text-cocoa-500">
      {pollAvailable
        ? "Vote with the route poll in Decisions once it is created."
        : "Create a poll from these routes to collect collaborator votes."}
    </div>
  );
}
