"use client";

// AppShell (M23): the shared layout every workspace screen reuses — a
// role-aware sidebar for navigation plus a topbar with the page title and
// a user chip with sign-out. Later admin screens (M24+) mount their content
// inside it and never re-implement navigation or session chrome.

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { type ReactNode } from "react";
import { useAuth } from "@/components/auth-provider";
import { getClerkSignOutHook } from "@/components/clerk-bridge";

interface NavItem {
  href: string;
  label: string;
  /** Only show this item to users holding the role (undefined = everyone signed in). */
  role?: string;
  /** Only show this item to users holding one of these roles. */
  roles?: string[];
  icon: string;
}

const NAV_ITEMS: NavItem[] = [
  { href: "/dashboard/admin", label: "Admin Dashboard", role: "ADMIN", icon: "📊" },
  { href: "/staff", label: "Staff Directory", role: "ADMIN", icon: "🧑‍💼" },
  { href: "/students", label: "Students", role: "ADMIN", icon: "🎓" },
  { href: "/guardians", label: "Guardians", roles: ["ADMIN", "MASTER_ADMIN", "TEACHER"], icon: "👪" },
  { href: "/academics", label: "Academic Structure", role: "ADMIN", icon: "📚" },
  { href: "/enrollments", label: "Enrollments", role: "ADMIN", icon: "📝" },
  { href: "/attendance", label: "Attendance", roles: ["ADMIN", "TEACHER"], icon: "📋" },
  { href: "/assessments", label: "Assessments", roles: ["ADMIN", "TEACHER"], icon: "✍️" },
  { href: "/announcements", label: "Announcements", role: "ADMIN", icon: "📢" },
  { href: "/teaching", label: "Teaching Assignments", role: "ADMIN", icon: "🧑‍🏫" },
  { href: "/portal/teacher", label: "Teacher Portal", role: "TEACHER", icon: "🧑‍🏫" },
  { href: "/portal/parent", label: "Parent Portal", role: "PARENT", icon: "👪" },
  { href: "/portal/student", label: "Student Portal", role: "STUDENT", icon: "🎓" },
  { href: "/reports", label: "Reports Center", icon: "📈" },
  { href: "/access", label: "Access Control", roles: ["ADMIN", "MASTER_ADMIN", "TEACHER"], icon: "🔐" },
  { href: "/audit", label: "Audit Log", roles: ["ADMIN", "MASTER_ADMIN"], icon: "📋" },
];

interface AppShellProps {
  /** Page title shown in the topbar breadcrumb. */
  title: string;
  /** Optional right-aligned controls (refresh button, context chips, …). */
  actions?: ReactNode;
  children: ReactNode;
}

export function AppShell({ title, actions, children }: AppShellProps) {
  const { user, logout } = useAuth();
  const pathname = usePathname();
  const router = useRouter();

  const visibleItems = NAV_ITEMS.filter((item) => {
    if (item.roles) {
      return item.roles.some((r) => user?.roles.includes(r));
    }
    if (item.role) {
      return user?.roles.includes(item.role);
    }
    return true;
  });

  const handleLogout = async () => {
    await logout();
    // In Clerk mode also end the Clerk session — otherwise the session
    // bridge immediately re-authenticates and "sign out" does nothing.
    const clerkSignOut = getClerkSignOutHook();
    await clerkSignOut();
    // Full-page navigation, not router.replace: the app-router RSC cache
    // still holds segments (and in-flight prefetches) from the *previous*
    // session; rendering the login page through the client router re-fires
    // those fetches and surfaces Next's NetworkError dev overlay until a
    // manual refresh. A hard reload starts the next session with clean state.
    window.location.replace("/login");
  };

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900">
      {/* Sidebar (lg+) */}
      <aside className="fixed inset-y-0 left-0 z-20 hidden w-60 flex-col border-r border-slate-200 bg-white lg:flex">
        <div className="flex h-16 items-center border-b border-slate-200 px-5">
          <Link href="/" className="text-lg font-bold text-brand-900 hover:text-brand-700">
            SchoolHub
          </Link>
        </div>
        <nav aria-label="Primary" className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {visibleItems.map((item) => {
            const active = pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  active
                    ? "bg-brand-50 text-brand-900 ring-1 ring-inset ring-brand-100"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                }`}
              >
                <span aria-hidden>{item.icon}</span>
                {item.label}
              </Link>
            );
          })}
        </nav>
        {user && (
          <div className="border-t border-slate-200 p-4">
            <p className="truncate text-sm font-semibold text-slate-800">
              {user.first_name} {user.last_name}
            </p>
            <p className="truncate text-xs text-slate-500">{user.email}</p>
            <span className="mt-2 inline-flex rounded-full bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700 ring-1 ring-inset ring-brand-100">
              {user.roles.join(", ")}
            </span>
          </div>
        )}
      </aside>

      {/* Topbar */}
      <header className="sticky top-0 z-10 border-b border-slate-200 bg-white shadow-sm lg:pl-60">
        <div className="flex min-h-16 flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-2 sm:px-6 lg:h-16 lg:flex-nowrap lg:py-0 lg:px-8">
          <div className="flex min-w-0 items-center gap-2">
            <Link href="/" className="text-lg font-bold text-brand-900 hover:text-brand-700 lg:hidden">
              SchoolHub
            </Link>
            <span className="text-slate-300 lg:hidden">/</span>
            <span className="truncate text-sm font-medium text-slate-600">{title}</span>
          </div>

          <div className="flex flex-wrap items-center justify-end gap-3">
            {actions}
            {user ? (
              <button
                onClick={handleLogout}
                className="min-h-11 rounded-md border border-slate-200 bg-white px-3 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50"
              >
                Sign out
              </button>
            ) : (
              <Link
                href="/login"
                className="min-h-11 rounded-md bg-brand-700 px-3 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
              >
                Sign in
              </Link>
            )}
          </div>
        </div>

        {/* Mobile nav strip */}
        <nav aria-label="Primary (mobile)" className="flex gap-1 overflow-x-auto border-t border-slate-100 px-4 py-2 lg:hidden">
          {visibleItems.map((item) => {
            const active = pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={`inline-flex min-h-9 items-center whitespace-nowrap rounded-md px-3 py-1.5 text-xs font-medium ${
                  active ? "bg-brand-50 text-brand-900" : "text-slate-600 hover:bg-slate-50"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>
      </header>

      {/* Content */}
      <main className="px-4 py-8 sm:px-6 lg:pl-[17.5rem] lg:pr-8">{children}</main>
    </div>
  );
}
