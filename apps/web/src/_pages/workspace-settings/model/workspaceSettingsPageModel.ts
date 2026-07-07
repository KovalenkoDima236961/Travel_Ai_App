import type { QueryClient } from "@tanstack/react-query";
import { workspaceKeys } from "@/lib/api/workspaces";
import type { WorkspaceRole } from "@/entities/workspace/model";

export const inviteRoles: Array<Exclude<WorkspaceRole, "owner">> = [
  "admin",
  "member",
  "viewer"
];

export async function invalidateWorkspace(queryClient: QueryClient, workspaceId: string) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: workspaceKeys.all }),
    queryClient.invalidateQueries({ queryKey: workspaceKeys.detail(workspaceId) }),
    queryClient.invalidateQueries({ queryKey: workspaceKeys.members(workspaceId) })
  ]);
}
