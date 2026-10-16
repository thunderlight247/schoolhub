"use client";

import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { Tabs } from "@/components/ui/tabs";
import {
  ChildDetail,
  ChildSummary,
  ParentOverviewData,
} from "@/types/parent_portal";
import { fetchChildDetail, fetchParentOverview } from "@/services/parent_portal";

export default function ParentPortalPage() {
  const [data, setData] = useState<ParentOverviewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Selected child detail state
  const [selectedChild, setSelectedChild] = useState<ChildSummary | null>(null);
  const [detail, setDetail] = useState<ChildDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [detailTab, setDetailTab] = useState<"attendance" | "results">("attendance");

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchParentOverview();
      setData(res);
      setSelectedChild(res.children?.[0] ?? null);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load parent portal. Please make sure you are logged in as a parent.");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    if (!selectedChild) {
      setDetail(null);
      return;
    }
    const loadDetail = async () => {
      setDetailLoading(true);
      setDetailError(null);
      try {
        const res = await fetchChildDetail(selectedChild.student_id);
        setDetail(res);
      } catch (err: unknown) {
        if (err instanceof Error) {
          setDetailError(err.message);
        } else {
          setDetailError("Failed to load child detail.");
        }
      } finally {
        setDetailLoading(false);
      }
    };
    loadDetail();
  }, [selectedChild]);

  const contextChips = (
    <>
      {data?.children?.[0]?.enrollment?.academic_year_name && (
        <span className="rounded-full bg-brand-50 px-3 py-1 text-xs font-semibold text-brand-700 border border-brand-100">
          AY {data.children[0].enrollment.academic_year_name}
        </span>
      )}
      <button
        onClick={loadData}
        disabled={loading}
        className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
      >
        Refresh
      </button>
    </>
  );

  return (
    <ProtectedRoute requiredRole="PARENT">
      <AppShell title="Parent Portal" actions={contextChips}>
        <div className="space-y-8">
        {/* Guardian Welcome Banner */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="inline-flex items-center gap-2 mb-2">
              <span className="text-xs font-bold uppercase tracking-wider text-brand-700 bg-brand-50 px-2 py-0.5 rounded border border-brand-100">
                Parent Workspace
              </span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Welcome, {data?.profile ? `${data.profile.first_name} ${data.profile.last_name}` : "Parent"}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              {data?.profile?.occupation ?? "Guardian"} &middot;{" "}
              {data?.children?.length ?? 0} linked {data?.children?.length === 1 ? "child" : "children"}
            </p>
          </div>

          <div className="flex flex-wrap gap-2 text-xs">
            {data?.profile?.phone && (
              <span className="inline-flex items-center rounded-md bg-slate-100 px-3 py-1.5 font-medium text-slate-600 border border-slate-200">
                📞 {data.profile.phone}
              </span>
            )}
            {data?.profile?.email && (
              <span className="inline-flex items-center rounded-md bg-slate-100 px-3 py-1.5 font-medium text-slate-600 border border-slate-200">
                ✉️ {data.profile.email}
              </span>
            )}
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

        {/* Children Cards */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-slate-900">My Children</h2>
              <p className="text-xs text-slate-500">Select a child to view attendance and results</p>
            </div>
          </div>

          {(!data?.children || data.children.length === 0) ? (
            <div className="rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center text-xs text-slate-500">
              No children are linked to your guardian account yet.
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {data.children.map((child) => {
                const isSelected = selectedChild?.student_id === child.student_id;
                const att = child.attendance_counts;
                const totalMarks = att.present + att.absent + att.late + att.excused;
                const presentRate = totalMarks > 0 ? Math.round((att.present / totalMarks) * 100) : 0;
                return (
                  <div
                    key={child.student_id}
                    onClick={() => {
                      setDetailTab("attendance");
                      setSelectedChild(child);
                    }}
                    className={`cursor-pointer rounded-xl border p-5 transition-all ${
                      isSelected
                        ? "border-brand-600 bg-brand-50/20 shadow-md ring-1 ring-brand-600"
                        : "border-slate-200 bg-white hover:border-slate-300 shadow-sm"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 text-base">
                        {child.first_name} {child.last_name}
                      </span>
                      <span className="text-[10px] font-semibold uppercase px-2 py-0.5 rounded bg-slate-100 text-slate-600">
                        {child.relationship}
                      </span>
                    </div>

                    <div className="mt-1 text-xs text-slate-400">{child.student_number}</div>

                    <div className="mt-2 flex items-center gap-2 text-xs">
                      {child.enrollment?.class_name ? (
                        <span className="text-sm font-semibold text-brand-700">
                          {child.enrollment.class_name}
                        </span>
                      ) : (
                        <span className="text-sm text-slate-400">No active enrollment</span>
                      )}
                      {child.enrollment?.academic_year_name && (
                        <span className="text-xs text-slate-400">
                          ({child.enrollment.academic_year_name})
                        </span>
                      )}
                    </div>

                    <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500">
                      <span>
                        Present rate:{" "}
                        <span className="font-semibold text-emerald-700">{presentRate}%</span>
                      </span>
                      <span>
                        Results: {child.results_count}
                        {child.average_score != null && (
                          <span className="ml-1 font-semibold text-brand-700">
                            (avg {child.average_score.toFixed(1)})
                          </span>
                        )}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Selected Child Detail */}
        {selectedChild && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border-b border-slate-100 pb-4">
              <div>
                <div className="inline-flex items-center gap-2">
                  <span className="text-base font-bold text-slate-900">
                    {selectedChild.first_name} {selectedChild.last_name}
                  </span>
                  <span className="rounded bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700">
                    {selectedChild.student_number}
                  </span>
                </div>
                <p className="text-xs text-slate-500 mt-0.5">
                  Attendance summary and results history for this child
                </p>
              </div>

              {/* Tab switcher */}
              <Tabs
                label="Child detail sections"
                active={detailTab}
                onChange={setDetailTab}
                variant="pills"
                items={[
                  { key: "attendance", label: "Attendance" },
                  { key: "results", label: "Results" },
                ]}
              />
            </div>

            {detailLoading ? (
              <div className="py-8 text-center text-xs text-slate-400">Loading child data...</div>
            ) : detailError ? (
              <div className="py-6 text-center text-xs text-red-600">{detailError}</div>
            ) : !detail ? (
              <div className="py-8 text-center text-xs text-slate-400">Select a child above.</div>
            ) : (
              <>
                {/* Attendance aggregate strip */}
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div className="rounded-lg border border-emerald-100 bg-emerald-50 p-3 text-center">
                    <div className="text-xl font-bold text-emerald-700">
                      {detail.attendance_counts.present}
                    </div>
                    <div className="text-[11px] text-emerald-600 font-medium uppercase">Present</div>
                  </div>
                  <div className="rounded-lg border border-amber-100 bg-amber-50 p-3 text-center">
                    <div className="text-xl font-bold text-amber-700">
                      {detail.attendance_counts.late}
                    </div>
                    <div className="text-[11px] text-amber-600 font-medium uppercase">Late</div>
                  </div>
                  <div className="rounded-lg border border-red-100 bg-red-50 p-3 text-center">
                    <div className="text-xl font-bold text-red-700">
                      {detail.attendance_counts.absent}
                    </div>
                    <div className="text-[11px] text-red-600 font-medium uppercase">Absent</div>
                  </div>
                  <div className="rounded-lg border border-sky-100 bg-sky-50 p-3 text-center">
                    <div className="text-xl font-bold text-sky-700">
                      {detail.attendance_counts.excused}
                    </div>
                    <div className="text-[11px] text-sky-600 font-medium uppercase">Excused</div>
                  </div>
                </div>

                {/* Attendance tab */}
                {detailTab === "attendance" && (
                  <div className="space-y-3">
                    <h3 className="text-sm font-bold text-slate-900">Recent Attendance Marks</h3>
                    {detail.recent_attendance.length === 0 ? (
                      <div className="py-6 text-center text-xs text-slate-400">
                        No attendance marks recorded yet.
                      </div>
                    ) : (
                      <div className="overflow-x-auto">
                        <table className="w-full min-w-[620px] text-left text-xs">
                          <thead className="bg-slate-50 text-slate-600 font-semibold border-b border-slate-200">
                            <tr>
                              <th className="py-3 px-3">Date</th>
                              <th className="py-3 px-3">Status</th>
                              <th className="py-3 px-3">Class</th>
                              <th className="py-3 px-3">Subject</th>
                              <th className="py-3 px-3">Term</th>
                              <th className="py-3 px-3">Notes</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-100">
                            {detail.recent_attendance.map((rec) => (
                              <tr key={rec.record_id} className="hover:bg-slate-50/60">
                                <td className="py-3 px-3 text-slate-500">{rec.session_date}</td>
                                <td className="py-3 px-3">
                                  <span
                                    className={`rounded px-2 py-0.5 text-[10px] font-semibold ${
                                      rec.status === "PRESENT"
                                        ? "bg-emerald-50 text-emerald-700"
                                        : rec.status === "LATE"
                                        ? "bg-amber-50 text-amber-700"
                                        : rec.status === "EXCUSED"
                                        ? "bg-sky-50 text-sky-700"
                                        : "bg-red-50 text-red-700"
                                    }`}
                                  >
                                    {rec.status}
                                  </span>
                                </td>
                                <td className="py-3 px-3 text-slate-500">{rec.class_name}</td>
                                <td className="py-3 px-3 font-medium text-slate-800">
                                  {rec.subject_name}{" "}
                                  <span className="text-slate-400">({rec.subject_code})</span>
                                </td>
                                <td className="py-3 px-3 text-slate-500">{rec.term_name}</td>
                                <td className="py-3 px-3 text-slate-400">{rec.notes ?? "—"}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                )}

                {/* Results tab */}
                {detailTab === "results" && (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <h3 className="text-sm font-bold text-slate-900">Results History</h3>
                      {detail.student.average_score != null && (
                        <span className="text-xs font-semibold text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                          Average: {detail.student.average_score.toFixed(1)}
                        </span>
                      )}
                    </div>
                    {detail.results.length === 0 ? (
                      <div className="py-6 text-center text-xs text-slate-400">
                        No results recorded yet.
                      </div>
                    ) : (
                      <div className="overflow-x-auto">
                        <table className="w-full min-w-[760px] text-left text-xs">
                          <thead className="bg-slate-50 text-slate-600 font-semibold border-b border-slate-200">
                            <tr>
                              <th className="py-3 px-3">Assessment</th>
                              <th className="py-3 px-3">Type</th>
                              <th className="py-3 px-3">Subject</th>
                              <th className="py-3 px-3">Term</th>
                              <th className="py-3 px-3">Date</th>
                              <th className="py-3 px-3">Score</th>
                              <th className="py-3 px-3">Grade</th>
                              <th className="py-3 px-3">Remarks</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-100">
                            {detail.results.map((res) => (
                              <tr key={res.result_id} className="hover:bg-slate-50/60">
                                <td className="py-3 px-3 font-semibold text-slate-900">
                                  {res.assessment_title}
                                </td>
                                <td className="py-3 px-3">
                                  <span className="rounded bg-brand-50 px-2 py-0.5 text-[10px] font-bold uppercase text-brand-700">
                                    {res.assessment_type}
                                  </span>
                                </td>
                                <td className="py-3 px-3 text-slate-500">
                                  {res.subject_name}{" "}
                                  <span className="text-slate-400">({res.subject_code})</span>
                                </td>
                                <td className="py-3 px-3 text-slate-500">{res.term_name}</td>
                                <td className="py-3 px-3 text-slate-500">{res.assessment_date}</td>
                                <td className="py-3 px-3 font-semibold text-slate-900">
                                  {res.score != null ? `${res.score} / ${res.max_score}` : "—"}
                                </td>
                                <td className="py-3 px-3 text-slate-500">{res.grade ?? "—"}</td>
                                <td className="py-3 px-3 text-slate-400">{res.remarks ?? "—"}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        )}

        {/* Announcements Feed */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-base font-bold text-slate-900">Notices &amp; Announcements</h3>
              <p className="text-xs text-slate-500">School-wide and class-targeted communications</p>
            </div>
            <span className="text-xs font-semibold text-brand-700 bg-brand-50 px-2 py-0.5 rounded">
              M13 Domain
            </span>
          </div>

          {(!data?.announcements || data.announcements.length === 0) ? (
            <div className="py-6 text-center text-xs text-slate-400">
              No notices published for you or your children's classes.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {data.announcements.map((ann) => (
                <div key={ann.id} className="rounded-lg border border-slate-100 bg-slate-50/50 p-3.5 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-slate-900">{ann.title}</span>
                    <span
                      className={`text-[10px] font-bold px-1.5 py-0.5 rounded uppercase ${
                        ann.priority === "URGENT"
                          ? "bg-red-100 text-red-800"
                          : ann.priority === "HIGH"
                          ? "bg-amber-100 text-amber-800"
                          : "bg-slate-100 text-slate-700"
                      }`}
                    >
                      {ann.priority}
                    </span>
                  </div>
                  <p className="mt-1.5 text-slate-600 line-clamp-2">{ann.content}</p>
                  <div className="mt-2 flex items-center justify-between text-[11px] text-slate-400">
                    <span>
                      {ann.target_type === "CLASS" && ann.target_class
                        ? `Class: ${ann.target_class}`
                        : ann.target_type === "ROLE"
                        ? "Parent bulletin"
                        : "School-wide"}
                    </span>
                    <span>By {ann.author_name}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
