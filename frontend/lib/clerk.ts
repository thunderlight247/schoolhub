// Clerk integration status helper.
//
// The app supports two deployment modes:
//  - Session auth only (default): the Go backend owns login/sessions exactly
//    as before. Nothing about the app changes.
//  - Clerk-authenticated: when NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY is set (and
//    the backend has a Clerk secret key configured), /login renders Clerk's
//    hosted SignIn UI and the session is created via the backend's
//    POST /auth/clerk-session bridge (which verifies the Clerk JWT and issues
//    our session cookie). RBAC, roles, and all downstream behavior are
//    unchanged — the backend remains the source of truth.
//
export const clerkPublishableKey = process.env.NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY ?? "";

/** True when the frontend should mount the Clerk provider + components. */
export function isClerkEnabled(): boolean {
  return clerkPublishableKey.length > 0;
}
