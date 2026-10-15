"use client";

// M34: forced first-login password change. When the backend's
// must_change_password flag is set, the auth middleware only lets
// change-password / logout / me through — so ProtectedRoute renders this
// full-screen gate instead of page content until the password is rotated.
//
// Two credential stores, one UX:
//   * classic mode — POST /auth/change-password (backend re-checks the
//     current password, stores the new Argon2id hash, clears the flag,
//     kills every other session for the account).
//   * Clerk mode   — the password is Clerk's business: user.updatePassword
//     in Clerk's store, then POST /auth/clerk-password-changed clears the
//     flag server-side.
//
// The two variants are separate components so each calls its hooks
// unconditionally (useUser needs ClerkProvider and would throw in classic
// mode) — the shared field markup lives in PasswordFields.

import { useEffect, useRef, useState, type FormEvent } from "react";
import { useUser } from "@clerk/nextjs";
import { useAuth } from "@/components/auth-provider";
import { TextField } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import { isClerkEnabled } from "@/lib/clerk";
import { changePassword, clerkPasswordChanged } from "@/services/auth";

// A hard reload at the auth boundary mirrors the login page's
// navigateAfterAuth: no stale client-cache segments from the flagged
// session survive the transition to the working session.
function navigateAfterChange(dest: string) {
  window.location.replace(dest);
}

export function PasswordChangeGate() {
  const { user } = useAuth();

  if (!user) return null; // ProtectedRoute only renders us with a session
  if (isClerkEnabled()) return <ClerkPasswordForms email={user.email} />;
  return <ClassicPasswordForms email={user.email} />;
}

function ClassicPasswordForms({ email }: { email: string }) {
  const { logout, refresh, user } = useAuth();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (validationError(next, confirm, current, setError)) return;
    setBusy(true);
    setError(null);
    try {
      await changePassword(current, next);
      setDone(true);
      // Server state changed underneath the cached client state; a fresh
      // load re-resolves /me cleanly before the workspace renders.
      await refresh();
      setTimeout(() => navigateAfterChange("/"), 1200);
    } catch (err) {
      setError(changeErrorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <GateLayout
      email={email}
      headingRef={headingRef}
      done={done}
      error={error}
      busy={busy}
      current={current}
      next={next}
      confirm={confirm}
      onCurrent={setCurrent}
      onNext={setNext}
      onConfirm={setConfirm}
      onSubmit={handleSubmit}
      onLogout={async () => {
        await logout();
        navigateAfterChange("/login");
      }}
      userName={user ? `${user.first_name} ${user.last_name}` : ""}
    />
  );
}

function ClerkPasswordForms({ email }: { email: string }) {
  const { logout, refresh, user } = useAuth();
  const { user: clerkUser, isLoaded } = useUser();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    headingRef.current?.focus();
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (validationError(next, confirm, current, setError)) return;
    if (!isLoaded || !clerkUser) {
      setError("Clerk session is still loading — try again in a moment.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      // Rotate in Clerk's credential store first; the backend call below
      // only clears the must_change_password flag.
      await clerkUser.updatePassword({
        currentPassword: current,
        newPassword: next,
        signOutOfOtherSessions: true,
      });
      await clerkPasswordChanged();
      setDone(true);
      await refresh();
      setTimeout(() => navigateAfterChange("/"), 1200);
    } catch (err) {
      setError(
        changeErrorText(err, {
          "Password change rejected.": "Clerk rejected the change — the temporary password may not match Clerk's record.",
        }),
      );
    } finally {
      setBusy(false);
    }
  };

  return (
    <GateLayout
      email={email}
      headingRef={headingRef}
      done={done}
      error={error}
      busy={busy}
      current={current}
      next={next}
      confirm={confirm}
      onCurrent={setCurrent}
      onNext={setNext}
      onConfirm={setConfirm}
      onSubmit={handleSubmit}
      onLogout={async () => {
        await logout();
        navigateAfterChange("/login");
      }}
      userName={user ? `${user.first_name} ${user.last_name}` : ""}
    />
  );
}

// Shared presentational shell — both credential-store variants render this.
interface GateLayoutProps {
  email: string;
  headingRef: React.RefObject<HTMLHeadingElement | null>;
  done: boolean;
  error: string | null;
  busy: boolean;
  current: string;
  next: string;
  confirm: string;
  onCurrent: (v: string) => void;
  onNext: (v: string) => void;
  onConfirm: (v: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onLogout: () => void;
  userName: string;
}

function GateLayout(props: GateLayoutProps) {
  const {
    email, headingRef, done, error, busy,
    current, next, confirm, onCurrent, onNext, onConfirm,
    onSubmit, onLogout,
  } = props;

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <h1
            ref={headingRef}
            tabIndex={-1}
            className="text-xl font-bold tracking-tight text-slate-900"
          >
            Set your own password
          </h1>
          <p className="mt-2 text-sm text-slate-600">
            You are signed in with a temporary password. Before anything else,
            choose a new one — after this you never need the temporary password
            again.
          </p>
          <p className="mt-1 text-xs text-slate-400">{email}</p>
        </div>

        {done ? (
          <div
            role="status"
            className="rounded-xl border border-emerald-200 bg-emerald-50 p-6 text-center text-sm font-medium text-emerald-800"
          >
            Password changed — loading your workspace…
          </div>
        ) : (
          <form
            onSubmit={onSubmit}
            className="space-y-4 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
          >
            <TextField
              label="Temporary password"
              type="password"
              required
              autoComplete="current-password"
              value={current}
              onChange={(e) => onCurrent(e.target.value)}
            />
            <TextField
              label="New password"
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              value={next}
              onChange={(e) => onNext(e.target.value)}
              error={next && next.length < 8 ? "At least 8 characters." : undefined}
            />
            <TextField
              label="Repeat new password"
              type="password"
              required
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => onConfirm(e.target.value)}
              error={confirm && confirm !== next ? "Passwords do not match." : undefined}
            />
            {error && (
              <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
                {error}
              </p>
            )}
            <button
              type="submit"
              disabled={busy}
              className="w-full rounded-lg bg-brand-700 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-brand-600 disabled:opacity-50"
            >
              {busy ? "Changing password…" : "Change password"}
            </button>
            <button
              type="button"
              onClick={onLogout}
              className="w-full rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50"
            >
              Sign out instead
            </button>
          </form>
        )}
      </div>
    </main>
  );
}

function validationError(
  next: string,
  confirm: string,
  current: string,
  setError: (s: string | null) => void,
): boolean {
  if (next.length < 8) {
    setError("The new password must be at least 8 characters.");
    return true;
  }
  if (next !== confirm) {
    setError("The two new-password entries do not match.");
    return true;
  }
  if (next === current) {
    setError("Choose a password different from the temporary one.");
    return true;
  }
  return false;
}

function changeErrorText(err: unknown, overrides: Record<string, string> = {}) {
  const message =
    err instanceof ApiError
      ? err.message
      : "Could not change the password. Check the temporary password and try again.";
  return overrides[message] ?? (err instanceof ApiError ? message : message);
}
