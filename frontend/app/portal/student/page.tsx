"use client";

import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { Tabs } from "@/components/ui/tabs";
import {
  StudentAttendanceRecord,
  StudentEnrollmentHistoryItem,
  StudentOverviewData,
  StudentResult,
} from "@/types/student_portal";
import {
  fetchStudentAttendance,
  fetchStudentEnrollments,
  fetchStudentOverview,
  fetchStudentResults,
} from "@/services/student_portal";

export default function StudentPortalPage() {
  const [data, setData] = useState<StudentOverviewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Active view tab
  const [activeTab, setActiveTab] = useState<
    "overview" | "attendance" | "results" | "enrollments" | "announcements"
  >("overview");

  // Tab-specific fuller data
  const [attendanceRecords, setAttendanceRecords] = useState<StudentAttendanceRecord[]>([]);
  const [attendanceLoading, setAttendanceLoading] = useState(false);

  const [resultsList, setResultsList] = useState<StudentResult[]>([]);
  const [resultsLoading, setResultsLoading] = useState(false);

  const [enrollmentsList, setEnrollmentsList] = useState<StudentEnrollmentHistoryItem[]>([]);
  const [enrollmentsLoading, setEnrollmentsLoading] = useState(false);

  const loadOverview = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchStudentOverview();
      setData(res);
      setAttendanceRecords(res.recent_attendance || []);
      setResultsList(res.recent_results || []);
      setEnrollmentsList(res.enrollment_history || []);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load student portal. Please ensure you are logged in with a student account.");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadOverview();
  }, []);

  // When switching to dedicated tabs, load full records if needed
  useEffect(() => {
    if (activeTab === "attendance" && data) {
      const loadFullAttendance = async () => {
        setAttendanceLoading(true);
        try {
          const records = await fetchStudentAttendance(100);
          setAttendanceRecords(records);
        } catch {
          // fallback to overview recent
          if (data.recent_attendance) setAttendanceRecords(data.recent_attendance);
        } finally {
          setAttendanceLoading(false);
        }
      };
      loadFullAttendance();
    } else if (activeTab === "results" && data) {
      const loadFullResults = async () => {
        setResultsLoading(true);
        try {
          const results = await fetchStudentResults();
          setResultsList(results);
        } catch {
          if (data.recent_results) setResultsList(data.recent_results);
        } finally {
          setResultsLoading(false);
        }
      };
      loadFullResults();
    } else if (activeTab === "enrollments" && data) {
      const loadFullEnrollments = async () => {
        setEnrollmentsLoading(true);
        try {
          const enrollments = await fetchStudentEnrollments();
          setEnrollmentsList(enrollments);
        } catch {
          if (data.enrollment_history) setEnrollmentsList(data.enrollment_history);
        } finally {
          setEnrollmentsLoading(false);
        }
      };
      loadFullEnrollments();
    }
  }, [activeTab, data]);

  // Attendance rate calculations
  const counts = data?.attendance_counts || { present: 0, absent: 0, late: 0, excused: 0 };
  const totalMarks = counts.present + counts.absent + counts.late + counts.excused;
  const presentRate = totalMarks > 0 ? Math.round((counts.present / totalMarks) * 100) : 0;

  const contextChips = (
    <>
      {data?.academic_context?.academic_year_name && (
        <span className="rounded-full bg-brand-50 px-3 py-1 text-xs font-semibold text-brand-700 border border-brand-100">
          AY {data.academic_context.academic_year_name}
          {data.academic_context.term_name ? ` · ${data.academic_context.term_name}` : ""}
        </span>
      )}
      <button
        onClick={loadOverview}
        disabled={loading}
        className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
      >
        Refresh
      </button>
    </>
  );

  return (
    <ProtectedRoute requiredRole="STUDENT">
      <AppShell title="Student Portal" actions={contextChips}>
        <div className="space-y-6">
        {/* Student Welcome Banner */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="inline-flex items-center gap-2 mb-2">
              <span className="text-xs font-bold uppercase tracking-wider text-brand-700 bg-brand-50 px-2 py-0.5 rounded border border-brand-100">
                Student Workspace
              </span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-200">
                {data?.profile?.student_status || "ACTIVE"}
              </span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Welcome, {data?.profile ? `${data.profile.first_name} ${data.profile.last_name}` : "Student"}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              Student ID:{" "}
              <span className="font-mono font-medium text-slate-700">
                {data?.profile?.student_number || "—"}
              </span>
              {data?.enrollment?.class_name && (
                <>
                  {" "}
                  &middot; Class:{" "}
                  <span className="font-semibold text-brand-700">{data.enrollment.class_name}</span>
                </>
              )}
              {data?.enrollment?.programme_name && (
                <>
                  {" "}
                  &middot; Track:{" "}
                  <span className="text-slate-700">{data.enrollment.programme_name}</span>
                </>
              )}
            </p>
          </div>

          <div className="flex flex-wrap gap-2 text-xs">
            {data?.profile?.admission_date && (
              <span className="inline-flex items-center rounded-md bg-slate-100 px-3 py-1.5 font-medium text-slate-600 border border-slate-200">
                📅 Admitted: {data.profile.admission_date}
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
              onClick={loadOverview}
              className="text-xs font-semibold underline hover:text-red-900 ml-4"
            >
              Try Again
            </button>
          </div>
        )}

        {/* KPI Summary Cards */}
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {/* Class & Academic Context */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <span className="text-xs font-medium text-slate-500">Current Class</span>
            <div className="mt-1 text-2xl font-bold text-slate-900">
              {data?.enrollment?.class_name || "Unassigned"}
            </div>
            <p className="mt-1 text-xs text-slate-500">
              {data?.enrollment?.academic_year_name
                ? `Academic Year ${data.enrollment.academic_year_name}`
                : "No active enrollment"}
            </p>
          </div>

          {/* Attendance KPI */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <span className="text-xs font-medium text-slate-500">Attendance Rate</span>
            <div className="mt-1 flex items-baseline gap-2">
              <span className="text-2xl font-bold text-emerald-700">{presentRate}%</span>
              <span className="text-xs text-slate-400">({counts.present}/{totalMarks} sessions)</span>
            </div>
            <div className="mt-2 flex items-center gap-1.5 text-[11px] text-slate-500">
              <span className="text-emerald-600 font-medium">{counts.present} P</span> &middot;{" "}
              <span className="text-rose-600 font-medium">{counts.absent} A</span> &middot;{" "}
              <span className="text-amber-600 font-medium">{counts.late} L</span> &middot;{" "}
              <span className="text-sky-600 font-medium">{counts.excused} E</span>
            </div>
          </div>

          {/* Performance KPI */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <span className="text-xs font-medium text-slate-500">Academic Average</span>
            <div className="mt-1 text-2xl font-bold text-slate-900">
              {data?.results_summary?.average_score != null
                ? `${data.results_summary.average_score.toFixed(1)}%`
                : "N/A"}
            </div>
            <p className="mt-1 text-xs text-slate-500">
              {data?.results_summary?.results_count ?? 0} assessments recorded
            </p>
          </div>

          {/* Announcements KPI */}
          <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
            <span className="text-xs font-medium text-slate-500">Announcements</span>
            <div className="mt-1 text-2xl font-bold text-slate-900">
              {data?.announcements?.length ?? 0}
            </div>
            <p className="mt-1 text-xs text-slate-500">
              School &amp; class broadcasts
            </p>
          </div>
        </div>

        {/* Tab Navigation */}
        <Tabs
          label="Student portal sections"
          active={activeTab}
          onChange={setActiveTab}
          items={[
            { key: "overview", label: "Overview" },
            { key: "attendance", label: "Attendance Register" },
            { key: "results", label: "Grades & Results" },
            { key: "enrollments", label: "Enrollment History" },
            {
              key: "announcements",
              label: `Announcements (${data?.announcements?.length ?? 0})`,
            },
          ]}
        />

        {/* Tab 1: Overview */}
        {activeTab === "overview" && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Recent Attendance Snapshot */}
              <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="font-bold text-slate-900 text-base">Recent Attendance</h3>
                  <button
                    onClick={() => setActiveTab("attendance")}
                    className="text-xs font-semibold text-brand-700 hover:underline"
                  >
                    View All &rarr;
                  </button>
                </div>
                {(!data?.recent_attendance || data.recent_attendance.length === 0) ? (
                  <p className="text-xs text-slate-400 py-4 text-center">No attendance marks found.</p>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {data.recent_attendance.map((rec) => (
                      <div key={rec.record_id} className="py-2.5 flex items-center justify-between text-xs">
                        <div>
                          <div className="font-medium text-slate-800">
                            {rec.subject_name} ({rec.subject_code})
                          </div>
                          <div className="text-[11px] text-slate-400">
                            {rec.session_date} &middot; {rec.term_name}
                          </div>
                        </div>
                        <span
                          className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                            rec.status === "PRESENT"
                              ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                              : rec.status === "ABSENT"
                              ? "bg-rose-50 text-rose-700 border border-rose-200"
                              : rec.status === "LATE"
                              ? "bg-amber-50 text-amber-700 border border-amber-200"
                              : "bg-sky-50 text-sky-700 border border-sky-200"
                          }`}
                        >
                          {rec.status}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* Recent Results Snapshot */}
              <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="font-bold text-slate-900 text-base">Recent Results</h3>
                  <button
                    onClick={() => setActiveTab("results")}
                    className="text-xs font-semibold text-brand-700 hover:underline"
                  >
                    View All &rarr;
                  </button>
                </div>
                {(!data?.recent_results || data.recent_results.length === 0) ? (
                  <p className="text-xs text-slate-400 py-4 text-center">No results posted yet.</p>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {data.recent_results.map((res) => (
                      <div key={res.result_id} className="py-2.5 flex items-center justify-between text-xs">
                        <div>
                          <div className="font-medium text-slate-800">
                            {res.assessment_title} &middot; {res.subject_name}
                          </div>
                          <div className="text-[11px] text-slate-400">
                            {res.assessment_date} &middot; {res.assessment_type}
                          </div>
                        </div>
                        <div className="text-right">
                          <div className="font-bold text-slate-900">
                            {res.score != null ? `${res.score} / ${res.max_score}` : "—"}
                          </div>
                          {res.grade && (
                            <span className="text-[10px] font-semibold text-brand-700 bg-brand-50 px-1.5 py-0.5 rounded">
                              Grade {res.grade}
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            {/* Profile & Enrollment Details */}
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <h3 className="font-bold text-slate-900 text-base mb-4">Student Profile Details</h3>
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 text-xs">
                <div>
                  <span className="text-slate-400 block">Full Name</span>
                  <span className="font-medium text-slate-800">
                    {data?.profile?.first_name} {data?.profile?.other_names || ""} {data?.profile?.last_name}
                  </span>
                </div>
                <div>
                  <span className="text-slate-400 block">Student Number</span>
                  <span className="font-mono font-medium text-slate-800">{data?.profile?.student_number}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Gender</span>
                  <span className="font-medium text-slate-800">{data?.profile?.gender || "Not specified"}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Date of Birth</span>
                  <span className="font-medium text-slate-800">{data?.profile?.date_of_birth || "—"}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Contact Phone</span>
                  <span className="font-medium text-slate-800">{data?.profile?.phone || "—"}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Address</span>
                  <span className="font-medium text-slate-800">{data?.profile?.address || "—"}</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Attendance Register */}
        {activeTab === "attendance" && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-bold text-slate-900 text-base">Full Attendance History</h3>
                <p className="text-xs text-slate-500">Record of every marked session for your classes</p>
              </div>
              <div className="flex gap-2 text-xs">
                <span className="rounded bg-emerald-50 px-2 py-1 font-semibold text-emerald-700 border border-emerald-100">
                  {counts.present} Present
                </span>
                <span className="rounded bg-rose-50 px-2 py-1 font-semibold text-rose-700 border border-rose-100">
                  {counts.absent} Absent
                </span>
                <span className="rounded bg-amber-50 px-2 py-1 font-semibold text-amber-700 border border-amber-100">
                  {counts.late} Late
                </span>
                <span className="rounded bg-sky-50 px-2 py-1 font-semibold text-sky-700 border border-sky-100">
                  {counts.excused} Excused
                </span>
              </div>
            </div>

            {attendanceLoading ? (
              <p className="text-xs text-slate-400 py-8 text-center">Loading attendance register...</p>
            ) : attendanceRecords.length === 0 ? (
              <p className="text-xs text-slate-400 py-8 text-center">No attendance records found.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                    <tr>
                      <th className="py-2.5 px-3 font-semibold">Date</th>
                      <th className="py-2.5 px-3 font-semibold">Subject</th>
                      <th className="py-2.5 px-3 font-semibold">Class</th>
                      <th className="py-2.5 px-3 font-semibold">Term</th>
                      <th className="py-2.5 px-3 font-semibold">Status</th>
                      <th className="py-2.5 px-3 font-semibold">Notes</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {attendanceRecords.map((r) => (
                      <tr key={r.record_id} className="hover:bg-slate-50/50">
                        <td className="py-2.5 px-3 font-medium text-slate-800">{r.session_date}</td>
                        <td className="py-2.5 px-3 text-slate-700">
                          {r.subject_name}{" "}
                          <span className="text-[11px] text-slate-400 font-mono">({r.subject_code})</span>
                        </td>
                        <td className="py-2.5 px-3 text-slate-600">{r.class_name}</td>
                        <td className="py-2.5 px-3 text-slate-500">{r.term_name}</td>
                        <td className="py-2.5 px-3">
                          <span
                            className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                              r.status === "PRESENT"
                                ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                : r.status === "ABSENT"
                                ? "bg-rose-50 text-rose-700 border border-rose-200"
                                : r.status === "LATE"
                                ? "bg-amber-50 text-amber-700 border border-amber-200"
                                : "bg-sky-50 text-sky-700 border border-sky-200"
                            }`}
                          >
                            {r.status}
                          </span>
                        </td>
                        <td className="py-2.5 px-3 text-slate-400 italic">{r.notes || "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Tab 3: Grades & Results */}
        {activeTab === "results" && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="font-bold text-slate-900 text-base">Academic Results History</h3>
                <p className="text-xs text-slate-500">Graded assignments, tests, and examination scores</p>
              </div>
              {data?.results_summary?.average_score != null && (
                <div className="rounded-lg bg-brand-50 border border-brand-100 px-3 py-1.5 text-xs">
                  <span className="text-slate-600">Overall Average: </span>
                  <span className="font-bold text-brand-700">
                    {data.results_summary.average_score.toFixed(1)}%
                  </span>
                </div>
              )}
            </div>

            {resultsLoading ? (
              <p className="text-xs text-slate-400 py-8 text-center">Loading grades &amp; results...</p>
            ) : resultsList.length === 0 ? (
              <p className="text-xs text-slate-400 py-8 text-center">No graded assessment results found.</p>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                    <tr>
                      <th className="py-2.5 px-3 font-semibold">Date</th>
                      <th className="py-2.5 px-3 font-semibold">Assessment Title</th>
                      <th className="py-2.5 px-3 font-semibold">Subject</th>
                      <th className="py-2.5 px-3 font-semibold">Type</th>
                      <th className="py-2.5 px-3 font-semibold">Term</th>
                      <th className="py-2.5 px-3 font-semibold">Score / Max</th>
                      <th className="py-2.5 px-3 font-semibold">Grade</th>
                      <th className="py-2.5 px-3 font-semibold">Remarks</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {resultsList.map((r) => {
                      const pct = r.score != null && r.max_score > 0 ? Math.round((r.score / r.max_score) * 100) : null;
                      return (
                        <tr key={r.result_id} className="hover:bg-slate-50/50">
                          <td className="py-2.5 px-3 font-medium text-slate-800">{r.assessment_date}</td>
                          <td className="py-2.5 px-3 font-medium text-slate-900">{r.assessment_title}</td>
                          <td className="py-2.5 px-3 text-slate-700">
                            {r.subject_name}{" "}
                            <span className="text-[11px] text-slate-400 font-mono">({r.subject_code})</span>
                          </td>
                          <td className="py-2.5 px-3 text-slate-500">{r.assessment_type}</td>
                          <td className="py-2.5 px-3 text-slate-500">{r.term_name}</td>
                          <td className="py-2.5 px-3">
                            <span className="font-bold text-slate-900">
                              {r.score != null ? r.score : "—"}
                            </span>
                            <span className="text-slate-400 font-normal"> / {r.max_score}</span>
                            {pct != null && (
                              <span className="ml-2 text-[10px] text-slate-500">({pct}%)</span>
                            )}
                          </td>
                          <td className="py-2.5 px-3">
                            {r.grade ? (
                              <span className="rounded bg-brand-50 border border-brand-100 px-2 py-0.5 text-[11px] font-bold text-brand-700">
                                {r.grade}
                              </span>
                            ) : (
                              <span className="text-slate-400">—</span>
                            )}
                          </td>
                          <td className="py-2.5 px-3 text-slate-500 italic">{r.remarks || "—"}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Tab 4: Enrollment History */}
        {activeTab === "enrollments" && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div>
              <h3 className="font-bold text-slate-900 text-base">Enrollment History</h3>
              <p className="text-xs text-slate-500">
                Your past and current academic memberships across years and cohorts
              </p>
            </div>

            {enrollmentsLoading ? (
              <p className="text-xs text-slate-400 py-8 text-center">Loading enrollment history...</p>
            ) : enrollmentsList.length === 0 ? (
              <p className="text-xs text-slate-400 py-8 text-center">No enrollment records found.</p>
            ) : (
              <div className="space-y-3">
                {enrollmentsList.map((e) => (
                  <div
                    key={e.enrollment_id}
                    className="flex flex-col sm:flex-row sm:items-center justify-between rounded-lg border border-slate-200 bg-slate-50/50 p-4 gap-3"
                  >
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-slate-900 text-sm">{e.class_name}</span>
                        {e.class_level && (
                          <span className="text-xs text-slate-500">({e.class_level})</span>
                        )}
                        <span
                          className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                            e.enrollment_status === "ENROLLED"
                              ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                              : "bg-slate-100 text-slate-600 border border-slate-200"
                          }`}
                        >
                          {e.enrollment_status}
                        </span>
                      </div>
                      <div className="text-xs text-slate-500 mt-1">
                        Academic Year:{" "}
                        <span className="font-medium text-slate-700">{e.academic_year_name}</span>
                        {e.programme_name && (
                          <>
                            {" "}
                            &middot; Programme:{" "}
                            <span className="font-medium text-slate-700">{e.programme_name}</span>
                          </>
                        )}
                      </div>
                    </div>

                    <div className="text-xs text-slate-400 sm:text-right">
                      <div>Enrolled: {e.enrollment_date}</div>
                      {e.exit_date && <div>Exit Date: {e.exit_date}</div>}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Tab 5: Announcements */}
        {activeTab === "announcements" && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div>
              <h3 className="font-bold text-slate-900 text-base">Announcements Feed</h3>
              <p className="text-xs text-slate-500">School-wide notices and updates for your class</p>
            </div>

            {(!data?.announcements || data.announcements.length === 0) ? (
              <p className="text-xs text-slate-400 py-8 text-center">No announcements at this time.</p>
            ) : (
              <div className="space-y-3">
                {data.announcements.map((ann) => (
                  <div key={ann.id} className="rounded-lg border border-slate-200 p-4 bg-white shadow-sm space-y-2">
                    <div className="flex items-center justify-between gap-2">
                      <h4 className="font-bold text-slate-900 text-sm">{ann.title}</h4>
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                          ann.priority === "URGENT"
                            ? "bg-rose-50 text-rose-700 border border-rose-200"
                            : ann.priority === "HIGH"
                            ? "bg-amber-50 text-amber-700 border border-amber-200"
                            : "bg-slate-100 text-slate-600"
                        }`}
                      >
                        {ann.priority}
                      </span>
                    </div>
                    <p className="text-xs text-slate-600 whitespace-pre-wrap">{ann.content}</p>
                    <div className="pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-400">
                      <span>By {ann.author_name}</span>
                      <span>{ann.published_at ? new Date(ann.published_at).toLocaleDateString() : ann.created_at}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
