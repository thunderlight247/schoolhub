"use client";

// Protected-route wrapper (M23). Declarative gating for any page:
//
//   <ProtectedRoute requiredRole="ADMIN"> ... </ProtectedRoute>
//   <ProtectedRoute> ... </ProtectedRoute>              // any signed-in user
//
// The backend remains the source of truth (RBAC middleware still rejects
// unauthorized API calls with 403); this only shapes the frontend experience:
// unauthenticated users are redirected to /login (preserving the return
// path), and users without the required role see an explicit access-denied
// panel rather than a page full of failing requests.

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { PasswordChangeGate } from "@/components/password-change-gate";
import { useAuth } from "@/components/auth-provider";

interface ProtectedRouteProps {
  children: ReactNode;
  /** When set, the signed-in user must hold this role to see the content. */
  requiredRole?: string;
  /** When set, the signed-in user must hold this permission (mirrors the backend's permission-gated routes, e.g. staff.view). */
  requiredPermission?: string;
}

export function ProtectedRoute({ children, requiredRole, requiredPermission }: ProtectedRouteProps) {
  const { user, loading, hasRole, hasPermission } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !user) {
      const next = pathname ? `?next=${encodeURIComponent(pathname)}` : "";
      router.replace(`/login${next}`);
    }
  }, [loading, user, router, pathname]);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-50">
        <div className="flex flex-col items-center gap-3">
          <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
          <span className="text-sm text-slate-500">Checking session…</span>
        </div>
      </div>
    );
  }

  if (!user) {
    // Render nothing — the redirect above is already in flight.
    return null;
  }

  // M34: a flagged (must_change_password) session can reach nothing through
  // the middleware until the password is rotated — render the change screen
  // first, before even the role check, so every protected route shows the
  // same gate.
  if (user.must_change_password) {
    return <PasswordChangeGate />;
  }

  const roleOk = !requiredRole || hasRole(requiredRole);
  const permissionOk = !requiredPermission || hasPermission(requiredPermission);
  if (!roleOk || !permissionOk) {
    const requirement = requiredRole
      ? `the ${requiredRole} role`
      : `the ${requiredPermission} permission`;
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-50 px-6">
        <div className="max-w-md rounded-xl border border-slate-200 bg-white p-8 text-center shadow-sm">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-50 text-2xl">
            🚫
          </div>
          <h1 className="text-lg font-bold text-slate-900">Access denied</h1>
          <p className="mt-2 text-sm text-slate-500">
            This area requires {requirement}. You are signed in as{" "}
            <span className="font-medium text-slate-700">
              {user.first_name} {user.last_name}
            </span>{" "}
            ({user.roles.join(", ")}).
          </p>
          <Link
            href="/"
            className="mt-6 inline-flex items-center justify-center rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50"
          >
            Back to home
          </Link>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
