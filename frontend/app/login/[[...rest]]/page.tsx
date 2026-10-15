"use client";

// Login page (M23): the first screen wired to the real M4 session auth.
// On success the user lands on ?next=... or their role's home workspace.
//
// Clerk mode (when NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY is set): renders Clerk's
// hosted <SignIn/> component instead. The ClerkSessionBridge in
// components/clerk-bridge.tsx handles converting the Clerk session into our
// backend session cookie; this page just renders the component and lets the
// sign-in complete.
//
// routing="path" requires the Clerk component to own the SUB-routes of its
// mount path too (e.g. /login/verify-email-address, /login/factor-one for
// two-step flows). A plain page.tsx 404s on those — hence this catch-all
// [[...rest]] page rendering <SignIn/> for every depth.

import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Suspense, useEffect, useRef, useState, type FormEvent } from "react";
import { SignIn, useUser } from "@clerk/nextjs";
import { useAuth } from "@/components/auth-provider";
import { ApiError } from "@/lib/api";
import { isClerkEnabled } from "@/lib/clerk";

function roleHome(roles: string[]): string {
  if (roles.includes("ADMIN")) return "/dashboard/admin";
  if (roles.includes("TEACHER")) return "/portal/teacher";
  if (roles.includes("PARENT")) return "/portal/parent";
  if (roles.includes("STUDENT")) return "/portal/student";
  return "/";
}

/**
 * Full-page navigation at the auth boundary. The app-router client cache
 * holds RSC segments fetched under the previous session's permissions; a
 * client-side router.replace after login/sign-out can replay those stale
 * fetches and trigger Next's NetworkError overlay (fixed only by a manual
 * refresh). Hard-navigating guarantees every segment loads under the new
 * session — and the extra round-trip is imperceptible at login time.
 */
function navigateAfterAuth(dest: string) {
  window.location.replace(dest);
}

function rolesFromClerk(u: ReturnType<typeof useUser>["user"] | null | undefined): string[] {
  return u?.publicMetadata && typeof u.publicMetadata === "object"
    ? ((u.publicMetadata as Record<string, unknown>).roles as string[]) ?? []
    : [];
}

function ClerkLoginForm() {
  const { isLoaded, isSignedIn, user } = useUser();
  const appAuth = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = searchParams.get("next");
  const bridged = useRef(false);

  // After Clerk sign-in completes, the component-level bridge (clerk-bridge)
  // posts the JWT to the backend. Once a backend user exists, redirect.
  useEffect(() => {
    if (!isLoaded || !isSignedIn || bridged.current) return;
    if (!appAuth.user) return; // bridge not finished yet
    bridged.current = true;
    const roles = appAuth.user.roles.length ? appAuth.user.roles : rolesFromClerk(user);
    navigateAfterAuth(
      nextPath && nextPath.startsWith("/") ? nextPath : roleHome(roles),
    );
  }, [isLoaded, isSignedIn, appAuth.user, bridged, nextPath, user]);

  // Already-signed-in visits: route to the workspace by backend role BEFORE
  // rendering <SignIn>, because Clerk's own auto-redirect for activated
  // sessions would otherwise bounce to fallbackRedirectUrl ("/") first.
  // While the backend session resolves, show a spinner until the redirect
  // effect fires — router.replace must live in an effect, not in render.
  const signedInWithBackendSession = isLoaded && isSignedIn && !!appAuth.user;

  useEffect(() => {
    if (!signedInWithBackendSession || bridged.current) return;
    bridged.current = true;
    navigateAfterAuth(
      nextPath && nextPath.startsWith("/") ? nextPath : roleHome(appAuth.user!.roles),
    );
  }, [signedInWithBackendSession, bridged, nextPath, appAuth.user]);

  if (signedInWithBackendSession) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-50">
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
      </main>
    );
  }
  if (isLoaded && isSignedIn && !appAuth.user) {
    // Clerk session but no backend session yet: let the bridge run
    // (render a spinner instead of the SignIn card so Clerk's own
    // auto-redirect to fallbackRedirectUrl cannot race us).
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-50">
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
      </main>
    );
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <Link href="/" className="text-2xl font-extrabold tracking-tight text-brand-900 hover:text-brand-700">
            SchoolHub
          </Link>
          <p className="mt-1 text-sm text-slate-500">Sign in to your workspace</p>
        </div>
        <div className="flex justify-center">
          {/* No forceRedirectUrl: the useEffect above routes by the actual
              backend role once the bridge resolves, so teacher/parent
              sign-ins land in the right portal. fallbackRedirectUrl only
              fires if the bridge hasn't resolved, and "/" just shows the
              landing page rather than a wrong portal. */}
          <SignIn
            path="/login"
            routing="path"
            signUpUrl="/register"
            fallbackRedirectUrl="/"
            appearance={{ variables: { colorPrimary: "#2f6f63" } }}
          />
        </div>
      </div>
    </main>
  );
}

function LegacyLoginForm() {
  const { login } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const nextPath = searchParams.get("next");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const user = await login(email, password);
      navigateAfterAuth(
        nextPath && nextPath.startsWith("/") ? nextPath : roleHome(user.roles),
      );
    } catch (err) {
      if (err instanceof ApiError && err.isUnauthenticated) {
        setError("Invalid email or password.");
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Could not reach the server. Is the backend running?");
      }
      setSubmitting(false);
    }
  };

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <Link href="/" className="text-2xl font-extrabold tracking-tight text-brand-900 hover:text-brand-700">
            SchoolHub
          </Link>
          <p className="mt-1 text-sm text-slate-500">Sign in to your workspace</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="space-y-4 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
        >
          <div>
            <label htmlFor="email" className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
              Email
            </label>
            <input
              id="email"
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              placeholder="you@schoolhub.test"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              placeholder="••••••••"
            />
          </div>

          {error && (
            <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={submitting}
            className="w-full rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-brand-600 disabled:opacity-50"
          >
            {submitting ? "Signing in…" : "Sign in"}
          </button>
        </form>
      </div>
    </main>
  );
}

function LoginInner() {
  // Same isClerkEnabled() gate as the layout's ClerkBridge: without a key
  // no Clerk code mounts, so prerendering never touches useUser().
  if (isClerkEnabled()) {
    return <ClerkLoginForm />;
  }
  return <LegacyLoginForm />;
}

// useSearchParams requires a suspense boundary in the app router during
// static prerendering.
export default function LoginPage() {
  return (
    <Suspense>
      <LoginInner />
    </Suspense>
  );
}
