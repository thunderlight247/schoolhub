"use client";

import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { Tabs } from "@/components/ui/tabs";
import {
  ClassAttendanceReport,
  ClassPerformanceReport,
  EnrollmentReport,
  StudentReportCard,
} from "@/types/reports";
import {
  fetchClassAttendanceReport,
  fetchClassPerformanceReport,
  fetchEnrollmentReport,
  fetchStudentReportCard,
} from "@/services/reports";

export default function ReportsPage() {
  const [activeTab, setActiveTab] = useState<
    "enrollment" | "attendance" | "performance" | "report-card"
  >("enrollment");

  // 1. Enrollment Report State
  const [enrollmentReport, setEnrollmentReport] = useState<EnrollmentReport | null>(null);
  const [enrollmentLoading, setEnrollmentLoading] = useState(false);
  const [enrollmentError, setEnrollmentError] = useState<string | null>(null);

  // 2. Class Attendance Report State
  const [attendanceClassId, setAttendanceClassId] = useState("");
  const [attendanceTermId, setAttendanceTermId] = useState("");
  const [attendanceStartDate, setAttendanceStartDate] = useState("");
  const [attendanceEndDate, setAttendanceEndDate] = useState("");
  const [attendanceReport, setAttendanceReport] = useState<ClassAttendanceReport | null>(null);
  const [attendanceLoading, setAttendanceLoading] = useState(false);
  const [attendanceError, setAttendanceError] = useState<string | null>(null);

  // 3. Class Performance Report State
  const [perfClassId, setPerfClassId] = useState("");
  const [perfTermId, setPerfTermId] = useState("");
  const [perfSubjectId, setPerfSubjectId] = useState("");
  const [perfReport, setPerfReport] = useState<ClassPerformanceReport | null>(null);
  const [perfLoading, setPerfLoading] = useState(false);
  const [perfError, setPerfError] = useState<string | null>(null);

  // 4. Student Report Card State
  const [rcStudentId, setRcStudentId] = useState("");
  const [rcTermId, setRcTermId] = useState("");
  const [reportCard, setReportCard] = useState<StudentReportCard | null>(null);
  const [rcLoading, setRcLoading] = useState(false);
  const [rcError, setRcError] = useState<string | null>(null);

  const loadEnrollment = async () => {
    setEnrollmentLoading(true);
    setEnrollmentError(null);
    try {
      const data = await fetchEnrollmentReport();
      setEnrollmentReport(data);
      // Pre-fill class id for convenience if classes exist
      if (data.classes.length > 0) {
        if (!attendanceClassId) setAttendanceClassId(data.classes[0].class_id);
        if (!perfClassId) setPerfClassId(data.classes[0].class_id);
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setEnrollmentError(err.message);
      } else {
        setEnrollmentError("Failed to load enrollment report.");
      }
    } finally {
      setEnrollmentLoading(false);
    }
  };

  useEffect(() => {
    loadEnrollment();
  }, []);

  const handleGenerateAttendance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!attendanceClassId) {
      setAttendanceError("Please enter or select a Class ID.");
      return;
    }
    setAttendanceLoading(true);
    setAttendanceError(null);
    try {
      const data = await fetchClassAttendanceReport(attendanceClassId, {
        term_id: attendanceTermId || undefined,
        start_date: attendanceStartDate || undefined,
        end_date: attendanceEndDate || undefined,
      });
      setAttendanceReport(data);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setAttendanceError(err.message);
      } else {
        setAttendanceError("Failed to generate class attendance report.");
      }
    } finally {
      setAttendanceLoading(false);
    }
  };

  const handleGeneratePerformance = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!perfClassId) {
      setPerfError("Please enter or select a Class ID.");
      return;
    }
    setPerfLoading(true);
    setPerfError(null);
    try {
      const data = await fetchClassPerformanceReport(perfClassId, {
        term_id: perfTermId || undefined,
        subject_id: perfSubjectId || undefined,
      });
      setPerfReport(data);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setPerfError(err.message);
      } else {
        setPerfError("Failed to generate class performance report.");
      }
    } finally {
      setPerfLoading(false);
    }
  };

  const handleGenerateReportCard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!rcStudentId || !rcTermId) {
      setRcError("Both Student ID and Term ID are required to generate a Report Card.");
      return;
    }
    setRcLoading(true);
    setRcError(null);
    try {
      const data = await fetchStudentReportCard(rcStudentId, rcTermId);
      setReportCard(data);
    } catch (err: unknown) {
      if (err instanceof Error) {
        setRcError(err.message);
      } else {
        setRcError("Failed to generate student report card.");
      }
    } finally {
      setRcLoading(false);
    }
  };

  const contextChips = (
    <>
      <button
        onClick={() => {
          if (activeTab === "enrollment") loadEnrollment();
        }}
        className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
      >
        Refresh
      </button>
    </>
  );

  return (
    <ProtectedRoute>
      <AppShell title="Reports Center" actions={contextChips}>
        <div className="space-y-6">
        {/* Banner */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="inline-flex items-center gap-2 mb-2">
            <span className="text-xs font-bold uppercase tracking-wider text-brand-700 bg-brand-50 px-2 py-0.5 rounded border border-brand-100">
              Institutional Intelligence
            </span>
          </div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">
            Reports &amp; Analytics Hub
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            Access institutional enrollment summaries, cohort attendance statistics, assessment
            distributions, and printable student transcripts.
          </p>
        </div>

        {/* Tab Navigation */}
        <Tabs
          label="Report types"
          active={activeTab}
          onChange={setActiveTab}
          items={[
            { key: "enrollment", label: "Enrollment Summary" },
            { key: "attendance", label: "Class Attendance Analytics" },
            { key: "performance", label: "Class Academic Performance" },
            { key: "report-card", label: "Student Report Card" },
          ]}
        />

        {/* Tab 1: Enrollment Summary */}
        {activeTab === "enrollment" && (
          <div className="space-y-6">
            {enrollmentError && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                {enrollmentError}
              </div>
            )}

            {/* KPI Cards */}
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <span className="text-xs font-medium text-slate-500">Total Enrolled</span>
                <div className="mt-1 text-2xl font-bold text-slate-900">
                  {enrollmentReport?.total_enrolled ?? 0}
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  Across {enrollmentReport?.total_classes ?? 0} active classes
                </p>
              </div>

              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <span className="text-xs font-medium text-slate-500">Capacity Utilization</span>
                <div className="mt-1 flex items-baseline gap-2">
                  <span className="text-2xl font-bold text-brand-700">
                    {enrollmentReport?.overall_utilization != null
                      ? `${enrollmentReport.overall_utilization.toFixed(1)}%`
                      : "N/A"}
                  </span>
                  <span className="text-xs text-slate-400">
                    ({enrollmentReport?.total_enrolled ?? 0}/{enrollmentReport?.total_capacity ?? 0})
                  </span>
                </div>
                <p className="mt-1 text-xs text-slate-500">Total classroom capacity</p>
              </div>

              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <span className="text-xs font-medium text-slate-500">Gender Distribution</span>
                <div className="mt-1 text-2xl font-bold text-slate-900">
                  {enrollmentReport?.total_male ?? 0} M / {enrollmentReport?.total_female ?? 0} F
                </div>
                <p className="mt-1 text-xs text-slate-500">
                  {enrollmentReport?.total_other_gender ?? 0} unspecified/other
                </p>
              </div>

              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <span className="text-xs font-medium text-slate-500">Academic Calendar</span>
                <div className="mt-1 text-xl font-bold text-slate-800">
                  {enrollmentReport?.academic_year_name || "All Years"}
                </div>
                <p className="mt-1 text-xs text-slate-500">Institutional scope</p>
              </div>
            </div>

            {/* Class Breakdown Table */}
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-bold text-slate-900 text-base">Classroom Enrollment &amp; Capacity</h3>
                  <p className="text-xs text-slate-500">
                    Detailed per-class seat utilization and status distribution
                  </p>
                </div>
              </div>

              {enrollmentLoading ? (
                <p className="text-xs text-slate-400 py-8 text-center">Loading enrollment report...</p>
              ) : !enrollmentReport?.classes || enrollmentReport.classes.length === 0 ? (
                <p className="text-xs text-slate-400 py-8 text-center">No enrollment records found.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] text-left text-xs">
                    <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                      <tr>
                        <th className="py-2.5 px-3 font-semibold">Class Name</th>
                        <th className="py-2.5 px-3 font-semibold">Level</th>
                        <th className="py-2.5 px-3 font-semibold">Programme</th>
                        <th className="py-2.5 px-3 font-semibold">Capacity</th>
                        <th className="py-2.5 px-3 font-semibold">Enrolled</th>
                        <th className="py-2.5 px-3 font-semibold">Utilization</th>
                        <th className="py-2.5 px-3 font-semibold">Gender (M/F/O)</th>
                        <th className="py-2.5 px-3 font-semibold">Completed / Transferred</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {enrollmentReport.classes.map((cls) => (
                        <tr key={cls.class_id} className="hover:bg-slate-50/50">
                          <td className="py-2.5 px-3 font-bold text-slate-900">{cls.class_name}</td>
                          <td className="py-2.5 px-3 text-slate-600">{cls.class_level || "—"}</td>
                          <td className="py-2.5 px-3 text-slate-600">{cls.programme_name || "—"}</td>
                          <td className="py-2.5 px-3 text-slate-600">{cls.capacity ?? "—"}</td>
                          <td className="py-2.5 px-3 font-semibold text-emerald-700">{cls.enrolled_count}</td>
                          <td className="py-2.5 px-3">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-slate-800">
                                {cls.utilization_rate != null ? `${cls.utilization_rate.toFixed(1)}%` : "—"}
                              </span>
                              {cls.utilization_rate != null && (
                                <div className="w-16 bg-slate-100 rounded-full h-1.5 overflow-hidden">
                                  <div
                                    className={`h-full rounded-full ${
                                      cls.utilization_rate > 100
                                        ? "bg-rose-500"
                                        : cls.utilization_rate > 85
                                        ? "bg-amber-500"
                                        : "bg-emerald-500"
                                    }`}
                                    style={{ width: `${Math.min(cls.utilization_rate, 100)}%` }}
                                  />
                                </div>
                              )}
                            </div>
                          </td>
                          <td className="py-2.5 px-3 text-slate-600">
                            {cls.male_count} / {cls.female_count} / {cls.other_gender_count}
                          </td>
                          <td className="py-2.5 px-3 text-slate-500">
                            {cls.completed_count} / {cls.transferred_count}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tab 2: Class Attendance Analytics */}
        {activeTab === "attendance" && (
          <div className="space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <h3 className="font-bold text-slate-900 text-base mb-3">Filter Attendance Report</h3>
              <form onSubmit={handleGenerateAttendance} className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 text-xs">
                <div>
                  <label className="block text-slate-600 font-medium mb-1">Class ID *</label>
                  <input
                    type="text"
                    value={attendanceClassId}
                    onChange={(e) => setAttendanceClassId(e.target.value)}
                    placeholder="Enter Class UUID"
                    required
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                  {enrollmentReport?.classes && (
                    <div className="mt-1 flex flex-wrap gap-1">
                      {enrollmentReport.classes.slice(0, 3).map((c) => (
                        <button
                          key={c.class_id}
                          type="button"
                          onClick={() => setAttendanceClassId(c.class_id)}
                          className="text-[10px] text-brand-700 hover:underline"
                        >
                          {c.class_name}
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">Term ID (Optional)</label>
                  <input
                    type="text"
                    value={attendanceTermId}
                    onChange={(e) => setAttendanceTermId(e.target.value)}
                    placeholder="Optional Term UUID"
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">Start Date (Optional)</label>
                  <input
                    type="date"
                    value={attendanceStartDate}
                    onChange={(e) => setAttendanceStartDate(e.target.value)}
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">End Date (Optional)</label>
                  <input
                    type="date"
                    value={attendanceEndDate}
                    onChange={(e) => setAttendanceEndDate(e.target.value)}
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div className="sm:col-span-2 md:col-span-4 flex justify-end">
                  <button
                    type="submit"
                    disabled={attendanceLoading}
                    className="rounded-md bg-brand-700 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 disabled:opacity-50"
                  >
                    {attendanceLoading ? "Generating..." : "Generate Attendance Report"}
                  </button>
                </div>
              </form>
            </div>

            {attendanceError && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                {attendanceError}
              </div>
            )}

            {attendanceReport && (
              <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-6">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-4 border-b border-slate-100 gap-4">
                  <div>
                    <h3 className="font-bold text-slate-900 text-lg">
                      {attendanceReport.class_name} Attendance Breakdown
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {attendanceReport.term_name ? `Term: ${attendanceReport.term_name} · ` : ""}
                      Total Sessions: <span className="font-semibold text-slate-700">{attendanceReport.total_sessions}</span> &middot;{" "}
                      Enrolled Students: <span className="font-semibold text-slate-700">{attendanceReport.total_students}</span>
                    </p>
                  </div>

                  <div className="rounded-lg bg-emerald-50 border border-emerald-100 px-4 py-2 text-right">
                    <span className="text-[11px] text-emerald-800 font-medium block">Class Average Attendance</span>
                    <span className="text-xl font-bold text-emerald-700">
                      {attendanceReport.average_attendance_rate.toFixed(1)}%
                    </span>
                  </div>
                </div>

                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] text-left text-xs">
                    <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                      <tr>
                        <th className="py-2.5 px-3 font-semibold">Student Name</th>
                        <th className="py-2.5 px-3 font-semibold">Student ID</th>
                        <th className="py-2.5 px-3 font-semibold">Present</th>
                        <th className="py-2.5 px-3 font-semibold">Absent</th>
                        <th className="py-2.5 px-3 font-semibold">Late</th>
                        <th className="py-2.5 px-3 font-semibold">Excused</th>
                        <th className="py-2.5 px-3 font-semibold">Total Records</th>
                        <th className="py-2.5 px-3 font-semibold">Attendance Rate</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {attendanceReport.students.map((st) => (
                        <tr key={st.student_id} className="hover:bg-slate-50/50">
                          <td className="py-2.5 px-3 font-semibold text-slate-900">
                            {st.last_name}, {st.first_name}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-slate-600">{st.student_number}</td>
                          <td className="py-2.5 px-3 text-emerald-700 font-medium">{st.present_count}</td>
                          <td className="py-2.5 px-3 text-rose-700 font-medium">{st.absent_count}</td>
                          <td className="py-2.5 px-3 text-amber-700 font-medium">{st.late_count}</td>
                          <td className="py-2.5 px-3 text-sky-700 font-medium">{st.excused_count}</td>
                          <td className="py-2.5 px-3 text-slate-600">{st.total_records}</td>
                          <td className="py-2.5 px-3">
                            <span
                              className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                                st.attendance_rate >= 85
                                  ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                  : st.attendance_rate >= 70
                                  ? "bg-amber-50 text-amber-700 border border-amber-200"
                                  : "bg-rose-50 text-rose-700 border border-rose-200"
                              }`}
                            >
                              {st.attendance_rate.toFixed(1)}%
                            </span>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Tab 3: Class Academic Performance */}
        {activeTab === "performance" && (
          <div className="space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <h3 className="font-bold text-slate-900 text-base mb-3">Filter Performance Report</h3>
              <form onSubmit={handleGeneratePerformance} className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
                <div>
                  <label className="block text-slate-600 font-medium mb-1">Class ID *</label>
                  <input
                    type="text"
                    value={perfClassId}
                    onChange={(e) => setPerfClassId(e.target.value)}
                    placeholder="Enter Class UUID"
                    required
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                  {enrollmentReport?.classes && (
                    <div className="mt-1 flex flex-wrap gap-1">
                      {enrollmentReport.classes.slice(0, 3).map((c) => (
                        <button
                          key={c.class_id}
                          type="button"
                          onClick={() => setPerfClassId(c.class_id)}
                          className="text-[10px] text-brand-700 hover:underline"
                        >
                          {c.class_name}
                        </button>
                      ))}
                    </div>
                  )}
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">Term ID (Optional)</label>
                  <input
                    type="text"
                    value={perfTermId}
                    onChange={(e) => setPerfTermId(e.target.value)}
                    placeholder="Optional Term UUID"
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">Subject ID (Optional)</label>
                  <input
                    type="text"
                    value={perfSubjectId}
                    onChange={(e) => setPerfSubjectId(e.target.value)}
                    placeholder="Optional Subject UUID"
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div className="sm:col-span-3 flex justify-end">
                  <button
                    type="submit"
                    disabled={perfLoading}
                    className="rounded-md bg-brand-700 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 disabled:opacity-50"
                  >
                    {perfLoading ? "Generating..." : "Generate Performance Report"}
                  </button>
                </div>
              </form>
            </div>

            {perfError && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                {perfError}
              </div>
            )}

            {perfReport && (
              <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
                <div className="flex items-center justify-between pb-3 border-b border-slate-100">
                  <div>
                    <h3 className="font-bold text-slate-900 text-lg">
                      {perfReport.class_name} Assessment Performance
                    </h3>
                    <p className="text-xs text-slate-500">
                      Distribution of scores across graded assessments
                    </p>
                  </div>
                  <span className="text-xs font-semibold text-brand-700 bg-brand-50 px-2.5 py-1 rounded">
                    {perfReport.assessments.length} Assessments Evaluated
                  </span>
                </div>

                {perfReport.assessments.length === 0 ? (
                  <p className="text-xs text-slate-400 py-8 text-center">
                    No assessments found matching the selected parameters.
                  </p>
                ) : (
                  <div className="overflow-x-auto">
                    <table className="w-full min-w-[760px] text-left text-xs">
                      <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
                        <tr>
                          <th className="py-2.5 px-3 font-semibold">Date</th>
                          <th className="py-2.5 px-3 font-semibold">Assessment Title</th>
                          <th className="py-2.5 px-3 font-semibold">Subject</th>
                          <th className="py-2.5 px-3 font-semibold">Type</th>
                          <th className="py-2.5 px-3 font-semibold">Submissions</th>
                          <th className="py-2.5 px-3 font-semibold">Average</th>
                          <th className="py-2.5 px-3 font-semibold">Highest</th>
                          <th className="py-2.5 px-3 font-semibold">Lowest</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {perfReport.assessments.map((a) => (
                          <tr key={a.assessment_id} className="hover:bg-slate-50/50">
                            <td className="py-2.5 px-3 text-slate-600">{a.assessment_date}</td>
                            <td className="py-2.5 px-3 font-bold text-slate-900">{a.assessment_title}</td>
                            <td className="py-2.5 px-3 text-slate-700">
                              {a.subject_name}{" "}
                              <span className="text-[11px] text-slate-400 font-mono">({a.subject_code})</span>
                            </td>
                            <td className="py-2.5 px-3 text-slate-500">{a.assessment_type}</td>
                            <td className="py-2.5 px-3 font-medium text-slate-700">{a.results_count}</td>
                            <td className="py-2.5 px-3">
                              <span className="font-bold text-brand-700">{a.average_score}</span>
                              <span className="text-slate-400"> / {a.max_score}</span>
                              {a.average_percentage != null && (
                                <span className="ml-1.5 text-[10px] text-slate-500 font-semibold">
                                  ({a.average_percentage.toFixed(1)}%)
                                </span>
                              )}
                            </td>
                            <td className="py-2.5 px-3 font-semibold text-emerald-700">{a.highest_score}</td>
                            <td className="py-2.5 px-3 font-semibold text-rose-700">{a.lowest_score}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Tab 4: Student Report Card */}
        {activeTab === "report-card" && (
          <div className="space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <h3 className="font-bold text-slate-900 text-base mb-3">Generate Student Report Card</h3>
              <form onSubmit={handleGenerateReportCard} className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
                <div>
                  <label className="block text-slate-600 font-medium mb-1">Student ID (UUID) *</label>
                  <input
                    type="text"
                    value={rcStudentId}
                    onChange={(e) => setRcStudentId(e.target.value)}
                    placeholder="Enter Student UUID"
                    required
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div>
                  <label className="block text-slate-600 font-medium mb-1">Term ID (UUID) *</label>
                  <input
                    type="text"
                    value={rcTermId}
                    onChange={(e) => setRcTermId(e.target.value)}
                    placeholder="Enter Term UUID"
                    required
                    className="w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                  />
                </div>

                <div className="sm:col-span-2 flex justify-end">
                  <button
                    type="submit"
                    disabled={rcLoading}
                    className="rounded-md bg-brand-700 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 disabled:opacity-50"
                  >
                    {rcLoading ? "Generating..." : "Generate Official Report Card"}
                  </button>
                </div>
              </form>
            </div>

            {rcError && (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
                {rcError}
              </div>
            )}

            {reportCard && (
              <div className="rounded-xl border border-slate-300 bg-white p-8 shadow-md space-y-6">
                {/* School Header */}
                <div className="text-center pb-6 border-b-2 border-slate-200">
                  <h2 className="text-2xl font-black tracking-tight text-slate-900">
                    SCHOOLHUB INSTITUTIONAL REPORT CARD
                  </h2>
                  <p className="text-xs uppercase tracking-widest text-slate-500 font-semibold mt-1">
                    Academic Year {reportCard.academic_year_name} &middot; {reportCard.term_name}
                  </p>
                </div>

                {/* Student Info Box */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 p-4 rounded-lg bg-slate-50 border border-slate-200 text-xs">
                  <div>
                    <span className="text-slate-400 block font-medium">Student Name</span>
                    <span className="font-bold text-slate-900 text-sm">
                      {reportCard.first_name} {reportCard.last_name}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-400 block font-medium">Student Number</span>
                    <span className="font-mono font-bold text-slate-800 text-sm">
                      {reportCard.student_number}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-400 block font-medium">Class &amp; Level</span>
                    <span className="font-bold text-slate-900">
                      {reportCard.class_name} {reportCard.class_level ? `(${reportCard.class_level})` : ""}
                    </span>
                  </div>
                  <div>
                    <span className="text-slate-400 block font-medium">Overall Standing</span>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-sm font-extrabold text-brand-700">
                        {reportCard.overall_average_percentage != null
                          ? `${reportCard.overall_average_percentage.toFixed(1)}%`
                          : "N/A"}
                      </span>
                      {reportCard.overall_grade && (
                        <span className="px-2 py-0.5 rounded bg-brand-100 text-brand-800 font-bold text-xs">
                          Grade {reportCard.overall_grade}
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* Attendance Summary */}
                <div className="rounded-lg border border-slate-200 p-4 flex flex-wrap items-center justify-between gap-4 text-xs bg-white">
                  <div>
                    <span className="font-bold text-slate-800">Term Attendance: </span>
                    <span className="text-slate-600">
                      {reportCard.attendance.present_count} Present &middot;{" "}
                      {reportCard.attendance.absent_count} Absent &middot;{" "}
                      {reportCard.attendance.late_count} Late &middot;{" "}
                      {reportCard.attendance.excused_count} Excused
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-slate-500">Attendance Rate:</span>
                    <span className="font-bold text-emerald-700 text-sm">
                      {reportCard.attendance.attendance_rate.toFixed(1)}%
                    </span>
                  </div>
                </div>

                {/* Subject Performance Table */}
                <div>
                  <h4 className="font-bold text-slate-900 text-sm mb-3">Academic Subjects Evaluation</h4>
                  <div className="overflow-x-auto border rounded-lg border-slate-200">
                    <table className="w-full min-w-[700px] text-left text-xs">
                      <thead className="border-b border-slate-200 bg-slate-50 text-slate-700">
                        <tr>
                          <th className="py-2.5 px-3 font-semibold">Subject Code</th>
                          <th className="py-2.5 px-3 font-semibold">Subject Name</th>
                          <th className="py-2.5 px-3 font-semibold">Type</th>
                          <th className="py-2.5 px-3 font-semibold">Assessments</th>
                          <th className="py-2.5 px-3 font-semibold">Score / Max</th>
                          <th className="py-2.5 px-3 font-semibold">Average %</th>
                          <th className="py-2.5 px-3 font-semibold">Letter Grade</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {reportCard.subjects.map((sub) => (
                          <tr key={sub.subject_id} className="hover:bg-slate-50/50">
                            <td className="py-2.5 px-3 font-mono font-medium text-slate-700">
                              {sub.subject_code}
                            </td>
                            <td className="py-2.5 px-3 font-bold text-slate-900">{sub.subject_name}</td>
                            <td className="py-2.5 px-3 text-slate-500">{sub.subject_type}</td>
                            <td className="py-2.5 px-3 text-slate-600">{sub.assessments_taken}</td>
                            <td className="py-2.5 px-3 text-slate-700">
                              {sub.total_score} / {sub.total_max_score}
                            </td>
                            <td className="py-2.5 px-3 font-bold text-slate-800">
                              {sub.assessments_taken > 0 ? `${sub.average_percentage.toFixed(1)}%` : "—"}
                            </td>
                            <td className="py-2.5 px-3">
                              <span
                                className={`px-2 py-0.5 rounded text-[11px] font-bold ${
                                  sub.grade.startsWith("A")
                                    ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                    : sub.grade === "B"
                                    ? "bg-sky-50 text-sky-700 border border-sky-200"
                                    : sub.grade === "C"
                                    ? "bg-amber-50 text-amber-700 border border-amber-200"
                                    : sub.grade === "N/A"
                                    ? "bg-slate-100 text-slate-500"
                                    : "bg-rose-50 text-rose-700 border border-rose-200"
                                }`}
                              >
                                {sub.grade}
                              </span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="pt-4 border-t border-slate-200 flex justify-between items-center text-xs text-slate-400">
                  <span>SchoolHub Official Academic Document</span>
                  <button
                    onClick={() => window.print()}
                    className="rounded border border-slate-300 bg-white px-3 py-1 font-semibold text-slate-700 shadow-sm hover:bg-slate-50"
                  >
                    Print Report Card
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
