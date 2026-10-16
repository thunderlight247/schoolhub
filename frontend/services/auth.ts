"use client";

// Session auth service layer (M23). The backend M4 endpoints own login,
// logout, and /me; the M23 auth-provider consumes these via AuthProvider.
// Clerk bridge POST /auth/clerk-session is optional — only called when Clerk
// keys are configured.

import { apiGet, apiPost } from "@/lib/api";
import { AuthUser } from "@/types/auth";

interface LoginResponse {
  user: AuthUser;
}

interface MeResponse {
  user: AuthUser;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  return apiPost<LoginResponse>("/auth/login", { email, password });
}

export async function logout(): Promise<void> {
  await apiPost<void>("/auth/logout", {});
}

export async function fetchMe(): Promise<MeResponse> {
  return apiGet<MeResponse>("/me");
}

/**
 * Demo deployment capability probe: GET /auth/demo-mode reports whether
 * this backend has the one-click demo sign-in mounted (DEMO_MODE). The
 * login page uses it to decide whether to render the persona picker; a
 * 404 (older backend) simply hides it.
 */
export async function fetchDemoMode(): Promise<boolean> {
  const res = await apiGet<{ demo_mode: boolean }>("/auth/demo-mode");
  return res.demo_mode;
}

export type DemoRole = "admin" | "teacher" | "parent";

/**
 * One-click demo sign-in: the backend resolves the designated persona
 * account for the role (admin, teacher, or parent), seeds an account for
 * it if needed, and issues the ordinary httpOnly session cookie — the
 * response shape matches the password login exactly. Only mounted on
 * demo deployments (DEMO_MODE).
 */
export async function demoLogin(role: DemoRole): Promise<LoginResponse> {
  return apiPost<LoginResponse>("/auth/demo-login", { as: role });
}

/**
 * Clerk bridge: exchange a verified-on-the-server Clerk session JWT for our
 * standard httpOnly session cookie. The backend verifies the JWT against
 * Clerk, matches the email to a provisioned account, and issues the session
 * exactly like POST /auth/login does. Only used when Clerk keys are set.
 */
export async function createClerkSession(clerkToken: string): Promise<LoginResponse> {
  return apiPost<LoginResponse>("/auth/clerk-session", { token: clerkToken });
}

/**
 * M34 forced first-login password change (classic mode): the backend
 * re-checks the current password, stores the new Argon2id hash, clears the
 * must_change_password flag, and invalidates every other session.
 */
export async function changePassword(
  currentPassword: string,
  newPassword: string,
): Promise<{ must_change_password: boolean }> {
  return apiPost<{ must_change_password: boolean }>("/auth/change-password", {
    current_password: currentPassword,
    new_password: newPassword,
  });
}

/**
 * M34, Clerk mode: the password was already rotated in Clerk's credential
 * store; the backend only clears the flag. Requires a valid session.
 */
export async function clerkPasswordChanged(): Promise<{ must_change_password: boolean }> {
  return apiPost<{ must_change_password: boolean }>("/auth/clerk-password-changed", {});
}
