"use client";

import { useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import {
  AssignedClass,
  StudentRosterItem,
  TeacherOverviewData,
} from "@/types/teacher_portal";
import { fetchClassRoster, fetchTeacherOverview } from "@/services/teacher_portal";

export default function TeacherPortalPage() {
  const [data, setData] = useState<TeacherOverviewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Class roster selection state
  const [selectedClass, setSelectedClass] = useState<AssignedClass | null>(null);
  const [roster, setRoster] = useState<StudentRosterItem[]>([]);
  const [rosterLoading, setRosterLoading] = useState(false);
  const [rosterError, setRosterError] = useState<string | null>(null);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetchTeacherOverview();
      setData(res);
      if (res.assigned_classes && res.assigned_classes.length > 0 && !selectedClass) {
        setSelectedClass(res.assigned_classes[0]);
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("Failed to load teacher portal. Please make sure you are logged in as a teacher.");
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    if (!selectedClass) return;
    const loadRoster = async () => {
      setRosterLoading(true);
      setRosterError(null);
      try {
        const list = await fetchClassRoster(selectedClass.class_id);
        setRoster(list);
      } catch (err: unknown) {
        if (err instanceof Error) {
          setRosterError(err.message);
        } else {
          setRosterError("Failed to load class roster.");
        }
      } finally {
        setRosterLoading(false);
      }
    };
    loadRoster();
  }, [selectedClass]);

  const contextChips = (
    <>
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
    </>
  );

  return (
    <ProtectedRoute requiredRole="TEACHER">
      <AppShell title="Teacher Portal" actions={contextChips}>
        <div className="space-y-8">
        {/* Teacher Welcome & Profile Banner */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="inline-flex items-center gap-2 mb-2">
              <span className="text-xs font-bold uppercase tracking-wider text-brand-700 bg-brand-50 px-2 py-0.5 rounded border border-brand-100">
                Teacher Workspace
              </span>
              <span className="text-xs text-slate-400">
                Staff ID: {data?.profile?.staff_number ?? "..."}
              </span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">
              Welcome, {data?.profile ? `${data.profile.first_name} ${data.profile.last_name}` : "Teacher"}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              {data?.profile?.position_title ?? "Instructor"} &middot; {data?.profile?.department_name ?? "Academics Department"}
            </p>
          </div>

          <div className="flex flex-wrap gap-2 text-xs">
            <span className="inline-flex items-center rounded-md bg-emerald-50 px-3 py-1.5 font-medium text-emerald-700 border border-emerald-100">
              Status: {data?.profile?.employment_status ?? "ACTIVE"}
            </span>
            {data?.profile?.email && (
              <span className="inline-flex items-center rounded-md bg-slate-100 px-3 py-1.5 font-medium text-slate-600 border border-slate-200">
                {data.profile.email}
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

        {/* KPI Metrics */}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Classes</div>
            <div className="mt-2 text-2xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.total_classes ?? 0)}
            </div>
            <div className="mt-1 text-[11px] text-slate-400">Assigned cohorts</div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Subjects</div>
            <div className="mt-2 text-2xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.total_subjects ?? 0)}
            </div>
            <div className="mt-1 text-[11px] text-slate-400">Curriculum areas</div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Students</div>
            <div className="mt-2 text-2xl font-bold text-brand-700">
              {loading ? "..." : (data?.counts?.total_students ?? 0)}
            </div>
            <div className="mt-1 text-[11px] text-slate-400">Active learners</div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Attendance</div>
            <div className="mt-2 text-2xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.total_attendance_sessions ?? 0)}
            </div>
            <div className="mt-1 text-[11px] text-slate-400">Registers captured</div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm col-span-2 sm:col-span-1">
            <div className="text-xs font-medium uppercase tracking-wider text-slate-500">Assessments</div>
            <div className="mt-2 text-2xl font-bold text-slate-900">
              {loading ? "..." : (data?.counts?.total_assessments ?? 0)}
            </div>
            <div className="mt-1 text-[11px] text-slate-400">Graded tasks</div>
          </div>
        </div>

        {/* Assigned Classes Grid */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-bold text-slate-900">Assigned Classes &amp; Subjects</h2>
              <p className="text-xs text-slate-500">Teaching assignments for the current academic cycle</p>
            </div>
          </div>

          {(!data?.assigned_classes || data.assigned_classes.length === 0) ? (
            <div className="rounded-xl border border-dashed border-slate-300 bg-white p-8 text-center text-xs text-slate-500">
              No classes currently assigned to your teacher account.
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {data.assigned_classes.map((cls) => {
                const isSelected = selectedClass?.assignment_id === cls.assignment_id;
                return (
                  <div
                    key={cls.assignment_id}
                    onClick={() => setSelectedClass(cls)}
                    className={`cursor-pointer rounded-xl border p-5 transition-all ${
                      isSelected
                        ? "border-brand-600 bg-brand-50/20 shadow-md ring-1 ring-brand-600"
                        : "border-slate-200 bg-white hover:border-slate-300 shadow-sm"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-bold text-slate-900 text-base">{cls.class_name}</span>
                      <span className="text-[10px] font-semibold uppercase px-2 py-0.5 rounded bg-slate-100 text-slate-600">
                        {cls.subject_type}
                      </span>
                    </div>

                    <div className="mt-2 flex items-center gap-2">
                      <span className="text-sm font-semibold text-brand-700">{cls.subject_name}</span>
                      <span className="text-xs text-slate-400">({cls.subject_code})</span>
                    </div>

                    <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500">
                      <span>Enrolled: {cls.enrolled_students_count} / {cls.class_capacity}</span>
                      <span className="font-semibold text-brand-700 hover:underline">
                        {isSelected ? "● Selected Roster" : "View Roster &rarr;"}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Selected Class Student Roster with Guardian Contact Details */}
        {selectedClass && (
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 border-b border-slate-100 pb-4">
              <div>
                <div className="inline-flex items-center gap-2">
                  <span className="text-base font-bold text-slate-900">
                    Class Roster: {selectedClass.class_name}
                  </span>
                  <span className="rounded bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700">
                    {selectedClass.subject_name}
                  </span>
                </div>
                <p className="text-xs text-slate-500 mt-0.5">
                  Enrolled student profiles and direct primary guardian contact info
                </p>
              </div>

              <div className="text-xs text-slate-500">
                Total students: <span className="font-bold text-slate-800">{roster.length}</span>
              </div>
            </div>

            {rosterLoading ? (
              <div className="py-8 text-center text-xs text-slate-400">Loading student roster...</div>
            ) : rosterError ? (
              <div className="py-6 text-center text-xs text-red-600">{rosterError}</div>
            ) : roster.length === 0 ? (
              <div className="py-8 text-center text-xs text-slate-400">
                No active students enrolled in this class.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 text-slate-600 font-semibold border-b border-slate-200">
                    <tr>
                      <th className="py-3 px-3">Student</th>
                      <th className="py-3 px-3">Student ID</th>
                      <th className="py-3 px-3">Gender</th>
                      <th className="py-3 px-3">DOB</th>
                      <th className="py-3 px-3">Status</th>
                      <th className="py-3 px-3">Guardian Name</th>
                      <th className="py-3 px-3">Relationship</th>
                      <th className="py-3 px-3">Guardian Contact</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {/* The roster query LEFT JOINs student_guardians, so a student
                        with two linked guardians produces one row per guardian —
                        key on the row index, not on student_id, to avoid React
                        duplicate-key warnings. */}
                    {roster.map((s, idx) => (
                      <tr key={`${s.student_id}-${idx}`} className="hover:bg-slate-50/60">
                        <td className="py-3 px-3 font-semibold text-slate-900">
                          {s.first_name} {s.last_name}
                        </td>
                        <td className="py-3 px-3 text-slate-500">{s.student_number}</td>
                        <td className="py-3 px-3 text-slate-500">{s.gender ?? "—"}</td>
                        <td className="py-3 px-3 text-slate-500">{s.date_of_birth ?? "—"}</td>
                        <td className="py-3 px-3">
                          <span className="rounded bg-emerald-50 px-2 py-0.5 text-[10px] font-semibold text-emerald-700">
                            {s.enrollment_status}
                          </span>
                        </td>
                        <td className="py-3 px-3 font-medium text-slate-800">
                          {s.guardian_name ?? "—"}
                        </td>
                        <td className="py-3 px-3 text-slate-500">
                          {s.guardian_relationship ?? "—"}
                        </td>
                        <td className="py-3 px-3">
                          <div className="flex flex-col gap-0.5 text-[11px]">
                            {s.guardian_phone && (
                              <a href={`tel:${s.guardian_phone}`} className="text-brand-700 hover:underline">
                                📞 {s.guardian_phone}
                              </a>
                            )}
                            {s.guardian_email && (
                              <a href={`mailto:${s.guardian_email}`} className="text-slate-500 hover:underline">
                                ✉️ {s.guardian_email}
                              </a>
                            )}
                            {!s.guardian_phone && !s.guardian_email && (
                              <span className="text-slate-400">None listed</span>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}

        {/* Attendance & Assessments Dual View */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          {/* Recent Attendance Registers */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-base font-bold text-slate-900">Recent Attendance Registers</h3>
                <p className="text-xs text-slate-500">Registers recorded for your classes</p>
              </div>
              <span className="text-xs font-semibold text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                M11 Domain
              </span>
            </div>

            {(!data?.recent_attendance || data.recent_attendance.length === 0) ? (
              <div className="py-6 text-center text-xs text-slate-400">
                No attendance registers recorded recently.
              </div>
            ) : (
              <div className="space-y-3">
                {data.recent_attendance.map((att) => (
                  <div key={att.session_id} className="rounded-lg border border-slate-100 bg-slate-50/50 p-3.5 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-slate-900">
                        {att.class_name} &middot; {att.subject_name}
                      </span>
                      <span className="text-slate-400">{att.session_date}</span>
                    </div>

                    <div className="mt-2.5 flex items-center gap-3">
                      <span className="text-emerald-700 font-medium">✓ {att.present_count} Present</span>
                      <span className="text-amber-700 font-medium">⏱ {att.late_count} Late</span>
                      <span className="text-red-700 font-medium">✕ {att.absent_count} Absent</span>
                      <span className="text-sky-700 font-medium">ℹ {att.excused_count} Excused</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Recent Assessments */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-base font-bold text-slate-900">Recent Graded Assessments</h3>
                <p className="text-xs text-slate-500">Score recording &amp; test status</p>
              </div>
              <span className="text-xs font-semibold text-brand-700 bg-brand-50 px-2 py-0.5 rounded">
                M12 Domain
              </span>
            </div>

            {(!data?.recent_assessments || data.recent_assessments.length === 0) ? (
              <div className="py-6 text-center text-xs text-slate-400">
                No assessments scheduled or recorded yet.
              </div>
            ) : (
              <div className="space-y-3">
                {data.recent_assessments.map((ass) => (
                  <div key={ass.assessment_id} className="rounded-lg border border-slate-100 bg-slate-50/50 p-3.5 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-slate-900">{ass.title}</span>
                      <span className="rounded bg-brand-50 px-1.5 py-0.5 font-bold uppercase text-[10px] text-brand-700">
                        {ass.assessment_type}
                      </span>
                    </div>

                    <div className="mt-1 flex items-center justify-between text-slate-500">
                      <span>{ass.class_name} &middot; {ass.subject_name}</span>
                      <span>Max score: {ass.max_score}</span>
                    </div>

                    <div className="mt-2.5 pt-2 border-t border-slate-100 flex items-center justify-between text-slate-500">
                      <span>Scores recorded: {ass.results_count} / {ass.total_students_count}</span>
                      <span className="text-slate-400">{ass.assessment_date}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Teacher Announcements Feed */}
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-base font-bold text-slate-900">Notices &amp; Announcements</h3>
              <p className="text-xs text-slate-500">School-wide communications and role bulletins</p>
            </div>
            <span className="text-xs font-semibold text-brand-700 bg-brand-50 px-2 py-0.5 rounded">
              M13 Domain
            </span>
          </div>

          {(!data?.announcements || data.announcements.length === 0) ? (
            <div className="py-6 text-center text-xs text-slate-400">
              No notices published for your classes or role.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {data.announcements.map((ann) => (
                <div key={ann.id} className="rounded-lg border border-slate-100 bg-slate-50/50 p-3.5 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-slate-900">{ann.title}</span>
                    <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded uppercase ${
                      ann.priority === 'URGENT'
                        ? 'bg-red-100 text-red-800'
                        : ann.priority === 'HIGH'
                        ? 'bg-amber-100 text-amber-800'
                        : 'bg-slate-100 text-slate-700'
                    }`}>
                      {ann.priority}
                    </span>
                  </div>
                  <p className="mt-1.5 text-slate-600 line-clamp-2">{ann.content}</p>
                  <div className="mt-2 text-right text-[11px] text-slate-400">
                    By {ann.author_name}
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
