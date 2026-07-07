"use client";

import {
  ReactNode,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState
} from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/components/auth/AuthProvider";
import { listWorkspaces, workspaceKeys } from "@/lib/api/workspaces";
import type { Workspace, WorkspaceRole } from "@/entities/workspace/model";

const STORAGE_KEY = "travel_ai_current_workspace_id";

export type WorkspaceScope = "all" | "personal" | "workspace";

type WorkspaceContextValue = {
  workspaces: Workspace[];
  editableWorkspaces: Workspace[];
  isLoading: boolean;
  currentScope: WorkspaceScope;
  currentWorkspaceId: string | null;
  currentWorkspace: Workspace | null;
  selectionValue: string;
  setAllTrips: () => void;
  setPersonalTrips: () => void;
  setCurrentWorkspace: (workspaceId: string) => void;
  setSelectionValue: (value: string) => void;
  refreshWorkspaces: () => Promise<void>;
};

const WorkspaceContext = createContext<WorkspaceContextValue | undefined>(undefined);

type WorkspaceProviderProps = {
  children: ReactNode;
};

export function WorkspaceProvider({ children }: WorkspaceProviderProps) {
  const { isAuthenticated, isLoading: authLoading } = useAuth();
  const queryClient = useQueryClient();
  const [selectionValue, setSelectionValueState] = useState("all");

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      setSelectionValueState(stored);
    }
  }, []);

  const workspacesQuery = useQuery({
    queryKey: workspaceKeys.list(),
    queryFn: listWorkspaces,
    enabled: isAuthenticated && !authLoading
  });

  const workspaces = workspacesQuery.data ?? [];

  const setSelectionValue = useCallback((value: string) => {
    const next = value === "personal" || value === "all" ? value : value.trim();
    setSelectionValueState(next || "all");
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, next || "all");
    }
  }, []);

  useEffect(() => {
    if (!isAuthenticated) {
      setSelectionValueState("all");
      return;
    }
    if (selectionValue === "all" || selectionValue === "personal" || workspacesQuery.isLoading) {
      return;
    }
    if (!workspaces.some((workspace) => workspace.id === selectionValue)) {
      setSelectionValue("all");
    }
  }, [isAuthenticated, selectionValue, setSelectionValue, workspaces, workspacesQuery.isLoading]);

  const currentScope: WorkspaceScope =
    selectionValue === "personal" ? "personal" : selectionValue === "all" ? "all" : "workspace";
  const currentWorkspaceId = currentScope === "workspace" ? selectionValue : null;
  const currentWorkspace =
    currentWorkspaceId != null
      ? workspaces.find((workspace) => workspace.id === currentWorkspaceId) ?? null
      : null;
  const editableWorkspaces = workspaces.filter((workspace) =>
    canCreateTripsInWorkspace(workspace.currentUserRole)
  );

  const refreshWorkspaces = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: workspaceKeys.all });
  }, [queryClient]);

  const value = useMemo<WorkspaceContextValue>(
    () => ({
      workspaces,
      editableWorkspaces,
      isLoading: workspacesQuery.isLoading,
      currentScope,
      currentWorkspaceId,
      currentWorkspace,
      selectionValue,
      setAllTrips: () => setSelectionValue("all"),
      setPersonalTrips: () => setSelectionValue("personal"),
      setCurrentWorkspace: setSelectionValue,
      setSelectionValue,
      refreshWorkspaces
    }),
    [
      currentScope,
      currentWorkspace,
      currentWorkspaceId,
      editableWorkspaces,
      refreshWorkspaces,
      selectionValue,
      setSelectionValue,
      workspaces,
      workspacesQuery.isLoading
    ]
  );

  return <WorkspaceContext.Provider value={value}>{children}</WorkspaceContext.Provider>;
}

export function useWorkspaces() {
  const value = useContext(WorkspaceContext);
  if (!value) {
    throw new Error("useWorkspaces must be used within WorkspaceProvider");
  }
  return value;
}

export function canManageWorkspace(role: WorkspaceRole) {
  return role === "owner" || role === "admin";
}

export function canArchiveWorkspace(role: WorkspaceRole) {
  return role === "owner";
}

export function canInviteWorkspaceMembers(role: WorkspaceRole) {
  return role === "owner" || role === "admin";
}

export function canCreateTripsInWorkspace(role: WorkspaceRole) {
  return role === "owner" || role === "admin" || role === "member";
}

export function formatWorkspaceRole(role: WorkspaceRole) {
  return role.charAt(0).toUpperCase() + role.slice(1);
}
