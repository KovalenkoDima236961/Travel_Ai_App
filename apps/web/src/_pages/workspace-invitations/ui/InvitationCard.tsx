import { Button } from "@/shared/ui/button";
import { Card } from "@/shared/ui/card";
import { formatWorkspaceRole } from "@/components/workspaces/WorkspaceProvider";
import { formatDate } from "@/lib/utils";
import type { WorkspaceInvitation } from "@/entities/workspace/model";

type InvitationCardProps = {
  invitation: WorkspaceInvitation;
  onAccept: () => void;
  onDecline: () => void;
  pending: boolean;
};

export function InvitationCard({
  invitation,
  onAccept,
  onDecline,
  pending
}: InvitationCardProps) {
  return (
    <Card className="flex h-full flex-col gap-5">
      <div>
        <h2 className="text-lg font-semibold text-slate-950">{invitation.workspaceName}</h2>
        <p className="mt-1 text-sm text-slate-500">
          Invited as {formatWorkspaceRole(invitation.role)}
        </p>
      </div>
      <div className="grid grid-cols-2 gap-3 text-sm">
        <InvitationFact label="Status" value={invitation.status} />
        <InvitationFact
          label="Sent"
          value={invitation.createdAt ? formatDate(invitation.createdAt) : "Unknown"}
        />
        <InvitationFact label="Email" value={invitation.email} />
        <InvitationFact
          label="Expires"
          value={invitation.expiresAt ? formatDate(invitation.expiresAt) : "No expiration"}
        />
      </div>
      <div className="mt-auto flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <Button disabled={pending} variant="secondary" onClick={onDecline}>
          Decline
        </Button>
        <Button disabled={pending} onClick={onAccept}>
          Accept
        </Button>
      </div>
    </Card>
  );
}

function InvitationFact({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-1 truncate font-semibold text-slate-800">{value}</p>
    </div>
  );
}
