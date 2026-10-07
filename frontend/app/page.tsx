import Link from "next/link";

export default function HomePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-4xl flex-col items-start justify-center gap-6 px-6 py-12">
      <div className="space-y-2">
        <div className="inline-flex items-center rounded-full bg-brand-50 px-3 py-1 text-xs font-semibold text-brand-700 border border-brand-100">
          Milestones M13, M14 &amp; M15 Implemented
        </div>
        <h1 className="text-3xl font-extrabold tracking-tight text-brand-900 sm:text-4xl">
          SchoolHub Management Portal
        </h1>
        <p className="text-base text-slate-600 max-w-2xl">
          A full-stack institutional administration platform built with Next.js, Go, and PostgreSQL.
          Features modular domain boundaries, strict RBAC authorization, transactional audit logging,
          targeted announcements, administrative KPI analytics, and role-specific operational portals.
        </p>
      </div>

      <div className="flex flex-wrap gap-4 pt-2">
        <Link
          href="/dashboard/admin"
          className="inline-flex items-center justify-center rounded-lg bg-brand-700 px-5 py-3 text-sm font-semibold text-white shadow-sm hover:bg-brand-600 transition-colors"
        >
          Open Admin Dashboard (M14) &rarr;
        </Link>
        <Link
          href="/portal/teacher"
          className="inline-flex items-center justify-center rounded-lg border border-slate-300 bg-white px-5 py-3 text-sm font-semibold text-slate-800 shadow-sm hover:bg-slate-50 transition-colors"
        >
          Open Teacher Portal (M15) &rarr;
        </Link>
      </div>

      <div className="mt-8 border-t border-slate-200 pt-6 w-full">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-slate-400">
          Recent Milestones
        </h2>
        <div className="mt-3 grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs text-slate-600">
          <div className="rounded-lg border border-slate-200 p-3 bg-white">
            <span className="font-bold text-slate-900">M13 — Announcements</span>
            <p className="mt-1 text-slate-500">
              School-wide broadcasts, role/class-targeted communication, draft/published status flows, and audience feed filtering.
            </p>
          </div>
          <div className="rounded-lg border border-slate-200 p-3 bg-white">
            <span className="font-bold text-slate-900">M14 — Admin Dashboard</span>
            <p className="mt-1 text-slate-500">
              Real-time institutional KPIs, attendance rate tracking, active academic context, recent activity feed, and quick actions.
            </p>
          </div>
          <div className="rounded-lg border border-slate-200 p-3 bg-white">
            <span className="font-bold text-slate-900">M15 — Teacher Portal</span>
            <p className="mt-1 text-slate-500">
              Assigned classes &amp; curriculum, student rosters with guardian contacts, attendance sessions, and assessments.
            </p>
          </div>
        </div>
      </div>
    </main>
  );
}
