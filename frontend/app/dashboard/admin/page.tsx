"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { AdminDashboardData } from "@/types/dashboard";
import { fetchAdminDashboard } from "@/services/dashboard";

export default function AdminDashboardPage() {
  const [data, setData] = useState<AdminDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchAdminDashboard();
      setData(res);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load dashboard data. Ensure backend is running and you are logged in as admin.");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 pb-16">
      {/* Top Header */}
      <header className="border-b border-slate-200 bg-white sticky top-0 z-10 shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-3">
            <Link href="/" className="text-xl font-bold text-brand-900 hover:text-brand-700">
              SchoolHub
            </Link>
            <span className="text-slate-300">/</span>
            <span className="text-sm font-medium text-slate-600">Admin Dashboard</span>
          </div>

          <div className="flex items-center gap-3">
            {data?.academic_context?.academic_year_name && (
              <span className="rounded-full bg-brand-50 px-3 py-1 text-xs font-semibold text-brand-700 border border-brand-100">
                AY {data.academic_context.academic_year_name}
              </span>
            )}
            {data?.academic_context?.term_name && (
              <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-700 border border-slate-200">
                {data.academic_context.term_name}
              </span>
            )}
            <button
              onClick={loadData}
              disabled={loading}
              className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
            >
              Refresh
            </button>
          </div>
        </div>
      </header>

      {/* Main Container */}
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8 space-y-8">
        {/* Title & Welcome */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">Institutional Overview</h1>
            <p className="text-sm text-slate-500 mt-1">
              Live key performance indicators, attendance health, announcements, and administrative activity.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center rounded-md bg-emerald-50 px-2 py-1 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/20">
              ● System Active
            </span>
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800 flex items-center justify-between">
            <div>
              <span className="font-semibold">Notice:</span> {error}
            </div>
            <button
              onClick={loadData}
              className="text-xs font-semibold underline hover:text-red-900 ml-4"
            >
              Try Again
            </button>
          </div>
        )}

        {/* KPI Summary Cards */}
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {/* Active Students */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Active Students</div>
            <div className="mt-2 text-3xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.active_students ?? 0)}
            </div>
            <div className="mt-2 text-xs text-slate-500">Enrolled across class cohorts</div>
          </div>

          {/* Active Staff */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Staff Members</div>
            <div className="mt-2 text-3xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.active_staff ?? 0)}
            </div>
            <div className="mt-2 text-xs text-brand-700 font-medium">
              {loading ? "" : `${data?.counts?.active_teachers ?? 0} teaching staff`}
            </div>
          </div>

          {/* Classes */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Class Cohorts</div>
            <div className="mt-2 text-3xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.total_classes ?? 0)}
            </div>
            <div className="mt-2 text-xs text-slate-500">Active curriculum streams</div>
          </div>

          {/* Active Enrollments */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Active Enrollments</div>
            <div className="mt-2 text-3xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.active_enrollments ?? 0)}
            </div>
            <div className="mt-2 text-xs text-slate-500">Current academic cycle</div>
          </div>
        </div>

        {/* Attendance & Quick Actions Grid */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Attendance KPI Card */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between">
                <h2 className="text-base font-semibold text-slate-900">Attendance Today</h2>
                <span className="text-xs text-slate-500">{data?.attendance?.date ?? "Today"}</span>
              </div>
              <p className="text-xs text-slate-500 mt-1">Aggregated class register results</p>

              <div className="my-6 flex items-center justify-center">
                <div className="text-center">
                  <div className="text-4xl font-extrabold text-brand-700">
                    {loading ? "..." : `${data?.attendance?.attendance_rate ?? 0}%`}
                  </div>
                  <div className="text-xs font-medium text-slate-500 mt-1">Overall Present Rate</div>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-2 text-xs">
                <div className="rounded-lg bg-emerald-50 p-2.5 border border-emerald-100">
                  <div className="text-emerald-700 font-medium">Present</div>
                  <div className="text-lg font-bold text-emerald-900 mt-0.5">
                    {data?.attendance?.present_count ?? 0}
                  </div>
                </div>
                <div className="rounded-lg bg-amber-50 p-2.5 border border-amber-100">
                  <div className="text-amber-700 font-medium">Late</div>
                  <div className="text-lg font-bold text-amber-900 mt-0.5">
                    {data?.attendance?.late_count ?? 0}
                  </div>
                </div>
                <div className="rounded-lg bg-red-50 p-2.5 border border-red-100">
                  <div className="text-red-700 font-medium">Absent</div>
                  <div className="text-lg font-bold text-red-900 mt-0.5">
                    {data?.attendance?.absent_count ?? 0}
                  </div>
                </div>
                <div className="rounded-lg bg-sky-50 p-2.5 border border-sky-100">
                  <div className="text-sky-700 font-medium">Excused</div>
                  <div className="text-lg font-bold text-sky-900 mt-0.5">
                    {data?.attendance?.excused_count ?? 0}
                  </div>
                </div>
              </div>
            </div>

            <div className="mt-6 pt-4 border-t border-slate-100 text-xs text-slate-500 flex justify-between">
              <span>Recorded sessions: {data?.attendance?.total_sessions ?? 0}</span>
              <span>Total records: {data?.attendance?.total_marks ?? 0}</span>
            </div>
          </div>

          {/* Quick Actions (2 Columns) */}
          <div className="lg:col-span-2 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-base font-semibold text-slate-900">Administrative Quick Actions</h2>
            <p className="text-xs text-slate-500 mt-1">Frequently used administrative workflows and portals</p>

            <div className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-2">
              {(data?.quick_actions ?? []).map((action) => (
                <div
                  key={action.id}
                  className="rounded-lg border border-slate-200 p-3.5 hover:border-brand-600 hover:shadow-sm transition-all group flex flex-col justify-between"
                >
                  <div>
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-semibold text-slate-900 group-hover:text-brand-700">
                        {action.title}
                      </span>
                      <span className="text-[10px] uppercase font-semibold text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded">
                        {action.category}
                      </span>
                    </div>
                    <p className="text-xs text-slate-500 mt-1">{action.description}</p>
                  </div>
                  <div className="mt-3 text-right">
                    <span className="text-xs font-semibold text-brand-700 group-hover:underline">
                      Launch &rarr;
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Announcements & Recent Audit Activity */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Recent Announcements */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-base font-semibold text-slate-900">Recent Announcements</h2>
                <p className="text-xs text-slate-500">School broadcasts and targeted notices</p>
              </div>
              <span className="text-xs font-semibold text-brand-700 bg-brand-50 px-2 py-1 rounded">
                M13 Domain
              </span>
            </div>

            <div className="space-y-3">
              {(!data?.recent_announcements || data.recent_announcements.length === 0) ? (
                <div className="text-xs text-slate-400 py-6 text-center">No announcements recorded yet.</div>
              ) : (
                data.recent_announcements.map((ann) => (
                  <div key={ann.id} className="rounded-lg border border-slate-100 bg-slate-50/50 p-3">
                    <div className="flex items-center justify-between gap-2">
                      <h3 className="text-sm font-semibold text-slate-800 truncate">{ann.title}</h3>
                      <div className="flex items-center gap-1.5 shrink-0">
                        <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded uppercase ${
                          ann.priority === 'URGENT'
                            ? 'bg-red-100 text-red-800'
                            : ann.priority === 'HIGH'
                            ? 'bg-amber-100 text-amber-800'
                            : 'bg-slate-100 text-slate-700'
                        }`}>
                          {ann.priority}
                        </span>
                        <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${
                          ann.status === 'PUBLISHED'
                            ? 'bg-emerald-100 text-emerald-800'
                            : 'bg-yellow-100 text-yellow-800'
                        }`}>
                          {ann.status}
                        </span>
                      </div>
                    </div>
                    <div className="mt-1 flex items-center justify-between text-xs text-slate-500">
                      <span>Target: {ann.target_type} {ann.target_role ? `(${ann.target_role})` : ''} {ann.target_class_name ? `(${ann.target_class_name})` : ''}</span>
                      <span>By {ann.author_name}</span>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {/* Recent Audit Activity */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h2 className="text-base font-semibold text-slate-900">Recent System Activity</h2>
                <p className="text-xs text-slate-500">Audit trail of institutional mutations</p>
              </div>
              <span className="text-xs font-semibold text-slate-600 bg-slate-100 px-2 py-1 rounded">
                Live Audit
              </span>
            </div>

            <div className="space-y-2.5">
              {(!data?.recent_activity || data.recent_activity.length === 0) ? (
                <div className="text-xs text-slate-400 py-6 text-center">No system activities recorded yet.</div>
              ) : (
                data.recent_activity.map((act) => (
                  <div key={act.id} className="flex items-center justify-between py-2 border-b border-slate-100 last:border-b-0 text-xs">
                    <div className="min-w-0 pr-3">
                      <div className="font-semibold text-slate-800 truncate">
                        {act.action}
                      </div>
                      <div className="text-slate-500 truncate text-[11px]">
                        {act.resource_type} &middot; by {act.actor_name || act.actor_email}
                      </div>
                    </div>
                    <div className="shrink-0 text-slate-400 text-[11px]">
                      {new Date(act.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
