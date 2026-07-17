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
import {
  login as loginRequest,
  logout as logoutRequest,
  me,
  refresh as refreshRequest,
  register as registerRequest,
  type Credentials
} from "@/shared/api/auth";
import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  saveTokens
} from "@/shared/api/auth";
import { clearCommandPaletteRecentItems } from "@/lib/command-palette/recent-items";
import { clearOfflineData, purgeStaleOfflineData } from "@/lib/offline/trip-cache";
import type { AuthUser, TokenResponse } from "@/shared/api/auth";

type AuthContextValue = {
  user: AuthUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (credentials: Credentials) => Promise<void>;
  register: (credentials: Credentials) => Promise<void>;
  logout: () => Promise<void>;
  refresh: () => Promise<TokenResponse>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

type AuthProviderProps = {
  children: ReactNode;
};

export function AuthProvider({ children }: AuthProviderProps) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const loadCurrentUser = useCallback(async (accessToken?: string) => {
    const token = accessToken ?? getAccessToken();
    if (!token) {
      setUser(null);
      return null;
    }

    const currentUser = await me(token);
    setUser(currentUser);
    return currentUser;
  }, []);

  useEffect(() => {
    let mounted = true;

    async function initialize() {
      try {
        const token = getAccessToken();
        if (!token) {
          clearTokens();
          if (mounted) {
            setUser(null);
          }
          return;
        }

        const currentUser = await me(token);
        if (mounted) {
          setUser(currentUser);
        }
        await purgeStaleOfflineData(currentUser.id);
      } catch {
        const refreshToken = getRefreshToken();
        if (refreshToken) {
          try {
            const refreshed = await refreshRequest(refreshToken);
            saveTokens(refreshed.accessToken, refreshed.refreshToken);
            const currentUser = await me(refreshed.accessToken);
            if (mounted) {
              setUser(currentUser);
            }
            await purgeStaleOfflineData(currentUser.id);
            return;
          } catch {
            clearTokens();
          }
        } else {
          clearTokens();
        }

        if (mounted) {
          setUser(null);
        }
      } finally {
        if (mounted) {
          setIsLoading(false);
        }
      }
    }

    void initialize();

    function handleSessionExpired() {
      clearTokens();
      void clearOfflineData();
      setUser(null);
    }

    window.addEventListener("auth:session-expired", handleSessionExpired);

    return () => {
      mounted = false;
      window.removeEventListener("auth:session-expired", handleSessionExpired);
    };
  }, []);

  const login = useCallback(async (credentials: Credentials) => {
    const response = await loginRequest(credentials);
    saveTokens(response.accessToken, response.refreshToken);
    setUser(response.user);
    await purgeStaleOfflineData(response.user.id);
  }, []);

  const register = useCallback(async (credentials: Credentials) => {
    const response = await registerRequest(credentials);
    saveTokens(response.accessToken, response.refreshToken);
    setUser(response.user);
    await purgeStaleOfflineData(response.user.id);
  }, []);

  const logout = useCallback(async () => {
    const refreshToken = getRefreshToken();
    const userId = user?.id;
    try {
      if (refreshToken) {
        await logoutRequest(refreshToken);
      }
    } catch {
      // Local logout should still complete if Auth Service is unreachable.
    } finally {
      await clearOfflineData(userId);
      clearCommandPaletteRecentItems(userId);
      clearTokens();
      setUser(null);
    }
  }, [user?.id]);

  const refresh = useCallback(async () => {
    const refreshToken = getRefreshToken();
    if (!refreshToken) {
      clearTokens();
      setUser(null);
      throw new Error("No refresh token is available.");
    }

    const response = await refreshRequest(refreshToken);
    saveTokens(response.accessToken, response.refreshToken);
    if (!user) {
      await loadCurrentUser(response.accessToken);
    }
    return response;
  }, [loadCurrentUser, user]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: Boolean(user),
      isLoading,
      login,
      register,
      logout,
      refresh
    }),
    [isLoading, login, logout, refresh, register, user]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return value;
}
