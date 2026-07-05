"use client";

import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from "react";
import type { UserMe } from "./api";
import { login as apiLogin, logout as apiLogout, fetchMe } from "./api";
import { clearTokens, getAccessToken, saveTokens } from "./tokens";

/**
 * Authenticated user type.
 */
export type User = UserMe;

/**
 * AuthContext shape.
 */
interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

/**
 * Provider component that manages authentication state.
 * On mount, if an access token exists, hydrates the user from GET /me.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Hydrate user on mount if token exists
  useEffect(() => {
    const token = getAccessToken();
    if (token) {
      fetchMe(token)
        .then((u) => setUser(u))
        .catch(() => {
          // Token invalid; clear and stay logged out
          clearTokens();
          setUser(null);
        })
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    setIsLoading(true);
    try {
      const { access_token, refresh_token } = await apiLogin(email, password);
      saveTokens({ access_token, refresh_token });
      const u = await fetchMe(access_token);
      setUser(u);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const logout = useCallback(async () => {
    setIsLoading(true);
    try {
      const token = getAccessToken();
      if (token) {
        await apiLogout(token);
      }
    } finally {
      clearTokens();
      setUser(null);
      setIsLoading(false);
    }
  }, []);

  const value: AuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading,
    login,
    logout,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/**
 * Hook to access auth context. Throws if used outside AuthProvider.
 */
export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
