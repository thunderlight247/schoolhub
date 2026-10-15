"use client";

// Session auth context (M23). The backend owns the session (httpOnly cookie,
// M4); this provider only mirrors who is signed in for the UI. State is
// resolved once on mount via GET /me — there is deliberately no token in
// localStorage to manage, so "logged in" is simply "the cookie is valid".
//
// Usage: mounted once in app/layout.tsx around the whole app. Pages that
// need gating use <ProtectedRoute> instead of consuming this directly.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import { fetchMe, login as loginRequest, logout as logoutRequest } from "@/services/auth";
import { AuthUser } from "@/types/auth";
import { ApiError } from "@/lib/api";

interface AuthContextValue {
  user: AuthUser | null;
  /** true until the initial GET /me has resolved */
  loading: boolean;
  hasRole: (role: string) => boolean;
  hasPermission: (permission: string) => boolean;
  login: (email: string, password: string) => Promise<AuthUser>;
  logout: () => Promise<void>;
  /** Re-resolve GET /me (used after the Clerk→backend session bridge). */
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    fetchMe()
      .then((res) => {
        if (!cancelled) setUser(res.user);
      })
      .catch(() => {
        // 401 (or any failure) just means signed out — stay null.
        if (!cancelled) setUser(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const res = await loginRequest(email, password);
    setUser(res.user);
    return res.user;
  }, []);

  const logout = useCallback(async () => {
    try {
      await logoutRequest();
    } finally {
      setUser(null);
    }
  }, []);

  const hasRole = useCallback(
    (role: string) => user?.roles.includes(role) ?? false,
    [user],
  );

  const hasPermission = useCallback(
    (permission: string) => user?.permissions.includes(permission) ?? false,
    [user],
  );

  const refresh = useCallback(async () => {
    try {
      const res = await fetchMe();
      setUser(res.user);
    } catch {
      setUser(null);
    }
  }, []);

  return (
    <AuthContext.Provider
      value={{ user, loading, hasRole, hasPermission, login, logout, refresh }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}

// Re-export for callers that only need to distinguish "not signed in"
// (ApiError with status 401) from other failures.
export { ApiError };
