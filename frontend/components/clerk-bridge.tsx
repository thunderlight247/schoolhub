"use client";

// Clerk bridge (frontend). When Clerk keys are configured:
//  - <ClerkBridge> mounts <ClerkProvider> around the app (inside the layout).
//  - /login renders Clerk's <SignIn/> instead of the legacy email/password form.
//  - After Clerk sign-in, the Clerk session JWT is posted to the backend's
//    POST /auth/clerk-session bridge, which verifies the token server-side and
//    issues the same httpOnly session cookie the legacy login produces.
//    AuthProvider then resolves GET /me as usual — RBAC, roles, and every
//    downstream screen are unchanged.
//  - <ClerkSignOut /> mounted by AppShell hooks "sign out" so BOTH the backend
//    session and the Clerk session end (otherwise the Clerk side immediately
//    re-bridges and the user never really leaves).
//
// When no Clerk publishable key is configured, everything renders as before
// (plain session-auth app) with zero Clerk code paths.

import { ClerkProvider, useAuth, useClerk, useSession } from "@clerk/nextjs";
import { createContext, useContext, useEffect, useRef, type ReactNode } from "react";
import { isClerkEnabled } from "@/lib/clerk";
import { useAuth as useAppAuth } from "@/components/auth-provider";
import { createClerkSession } from "@/services/auth";

/**
 * Watches the Clerk session; the moment one becomes active, posts its JWT to
 * the backend bridge so our httpOnly session cookie exists too. Skips when a
 * backend session already resolves (GET /me succeeded).
 */
function ClerkSessionBridge({ children }: { children: ReactNode }) {
  const { isLoaded, isSignedIn, session } = useSession();
  const { getToken } = useAuth();
  const appAuth = useAppAuthSafe();
  const bridging = useRef(false);

  useEffect(() => {
    if (!isLoaded || !isSignedIn || !session || bridging.current) return;
    if (appAuth?.user) return; // backend session already live
    bridging.current = true;
    (async () => {
      try {
        const token = await getToken();
        if (token) await createClerkSession(token);
        await appAuth?.refresh();
      } catch {
        // Bridge failure surfaces later when a gated page calls GET /me and
        // ProtectedRoute redirects to /login — the Clerk session remains.
      }
    })();
  }, [isLoaded, isSignedIn, session, getToken, appAuth]);

  // Register a sign-out hook the app shell calls in addition to its own
  // backend logout.
  useEffect(() => {
    if (!isClerkEnabled()) return;
    clerkSignOutHook = async () => {
      try {
        await useClerkRef?.signOut();
      } catch {
        // already signed out of Clerk — fine
      }
    };
    return () => {
      clerkSignOutHook = null;
    };
  }, []);

  return <>{children}</>;
}

// Module-level hook indirection so AppShell (which may render outside
// ClerkProvider when Clerk is disabled) can trigger a Clerk sign-out without
// importing Clerk hooks unconditionally.
let clerkSignOutHook: (() => Promise<void>) | null = null;
let useClerkRef: ReturnType<typeof useClerk> | null = null;

/** Returns a sign-out extender: awaits the hook when Clerk mode is on. */
export function getClerkSignOutHook(): () => Promise<void> {
  return async () => {
    if (clerkSignOutHook) await clerkSignOutHook();
  };
}

function BridgeInner({ children }: { children: ReactNode }) {
  useClerkRef = useClerk();
  useSession(); // ensure session subscription is alive inside the provider
  return <ClerkSessionBridge>{children}</ClerkSessionBridge>;
}

/**
 * Wraps the app in <ClerkProvider> when keys exist; renders children untouched
 * otherwise. ClerkProvider goes inside <body> per Clerk's Next.js guidance.
 */
export function ClerkBridge({ children }: { children: ReactNode }) {
  if (!isClerkEnabled()) {
    return <>{children}</>;
  }
  return (
    <ClerkProvider>
      <BridgeInner>{children}</BridgeInner>
    </ClerkProvider>
  );
}

// Small indirection so the bridge can depend on the app AuthProvider without a
// circular import at module scope.
function useAppAuthSafe() {
  try {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    return useAppAuth();
  } catch {
    return null; // AuthProvider not mounted (shouldn't happen — it wraps us)
  }
}

// Suppress unused-import warning for the context utilities if we refactor
// later to a hook-based sign-out registration.
void createContext;
void useContext;
