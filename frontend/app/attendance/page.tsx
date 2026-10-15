"use client";

// Attendance Register Workspace (M28)
// Full teacher/admin register capture, sessions directory, per-record
// mark corrections, and class attendance summary inspection.
// Gated under attendance.view (directory/summaries), attendance.record (capture),
// and attendance.update (corrections).

import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { StatCard } from "@/components/ui/stat-card";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  listAttendanceSessions,
  getAttendanceSession,
  createAttendanceSession,
  updateAttendanceRecord,
  getClassAttendanceSummary,
} from "@/services/attendance";
import { listClasses, listSubjects, listTerms, listClassSubjects } from "@/services/academics";
import { listEnrollments } from "@/services/enrollments";
import type {
  AttendanceSession,
  AttendanceRecord,
  AttendanceStatus,
  AttendanceSummary,
} from "@/types/attendance";
import { ATTENDANCE_STATUSES } from "@/types/attendance";
import type { Class, Subject, Term, ClassSubject } from "@/types/academics";

const PAGE_SIZE = 10;

function statusBadge(status: AttendanceStatus) {
  switch (status) {
    case "PRESENT":
      return (
        <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700 ring-1 ring-inset ring-emerald-600/20">
          Present
        </span>
      );
    case "ABSENT":
      return (
        <span className="inline-flex items-center rounded-full bg-rose-50 px-2 py-0.5 text-xs font-semibold text-rose-700 ring-1 ring-inset ring-rose-600/20">
          Absent
        </span>
      );
    case "LATE":
      return (
        <span className="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-xs font-semibold text-amber-700 ring-1 ring-inset ring-amber-600/20">
          Late
        </span>
      );
    case "EXCUSED":
      return (
        <span className="inline-flex items-center rounded-full bg-sky-50 px-2 py-0.5 text-xs font-semibold text-sky-700 ring-1 ring-inset ring-sky-600/20">
          Excused
        </span>
      );
    default:
      return (
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600">
          {status}
        </span>
      );
  }
}

interface RosterStudent {
  student_id: string;
  student_name: string;
  student_number: string;
  status: AttendanceStatus;
  notes: string;
}

function AttendanceWorkspace() {
  const { hasPermission } = useAuth();
  const canRecord = hasPermission("attendance.record");
  const canUpdate = hasPermission("attendance.update");

  // Sessions list state
  const [sessions, setSessions] = useState<AttendanceSession[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  // Filters
  const [filterClassId, setFilterClassId] = useState("");
  const [filterSubjectId, setFilterSubjectId] = useState("");
  const [filterTermId, setFilterTermId] = useState("");
  const [filterDateFrom, setFilterDateFrom] = useState("");
  const [filterDateTo, setFilterDateTo] = useState("");
  const [search, setSearch] = useState("");

  // Metadata dropdowns
  const [classes, setClasses] = useState<Class[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [terms, setTerms] = useState<Term[]>([]);

  // Take Register Modal state
  const [showTakeRegister, setShowTakeRegister] = useState(false);
  const [takeClassId, setTakeClassId] = useState("");
  const [takeSubjectId, setTakeSubjectId] = useState("");
  const [takeTermId, setTakeTermId] = useState("");
  const [takeDate, setTakeDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [classSubjects, setClassSubjects] = useState<ClassSubject[]>([]);
  const [roster, setRoster] = useState<RosterStudent[]>([]);
  const [rosterLoading, setRosterLoading] = useState(false);
  const [takeSubmitting, setTakeSubmitting] = useState(false);
  const [takeError, setTakeError] = useState<string | null>(null);

  // Detail / Correction Modal state
  const [detailModalOpen, setDetailModalOpen] = useState(false);
  const [selectedSession, setSelectedSession] = useState<AttendanceSession | null>(null);
  const [sessionRecords, setSessionRecords] = useState<AttendanceRecord[]>([]);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [savingRecordId, setSavingRecordId] = useState<string | null>(null);
  const [recordFeedback, setRecordFeedback] = useState<Record<string, string>>({});

  // Class Summary Modal state
  const [summaryModalOpen, setSummaryModalOpen] = useState(false);
  const [summaryTargetClass, setSummaryTargetClass] = useState<Class | null>(null);
  const [classSummary, setClassSummary] = useState<AttendanceSummary | null>(null);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryError, setSummaryError] = useState<string | null>(null);

  // Load metadata options
  useEffect(() => {
    listClasses().then(setClasses).catch(() => {});
    listSubjects().then(setSubjects).catch(() => {});
    listTerms().then(setTerms).catch(() => {});
  }, []);

  // Fetch session directory
  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listAttendanceSessions({
        class_id: filterClassId || undefined,
        subject_id: filterSubjectId || undefined,
        term_id: filterTermId || undefined,
        date_from: filterDateFrom || undefined,
        date_to: filterDateTo || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setSessions(res.sessions || []);
      setTotal(res.pagination?.total || 0);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load attendance sessions.");
    } finally {
      setLoading(false);
    }
  }, [filterClassId, filterSubjectId, filterTermId, filterDateFrom, filterDateTo, page]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  // Client-side search filtering over current page
  const filteredSessions = useMemo(() => {
    if (!search.trim()) return sessions;
    const q = search.toLowerCase();
    return sessions.filter(
      (s) =>
        s.class_name?.toLowerCase().includes(q) ||
        s.subject_name?.toLowerCase().includes(q) ||
        s.subject_code?.toLowerCase().includes(q) ||
        s.term_name?.toLowerCase().includes(q) ||
        s.session_date.includes(q),
    );
  }, [sessions, search]);

  // Available terms for the selected class in Take Register
  const selectedClass = useMemo(
    () => classes.find((c) => c.id === takeClassId),
    [classes, takeClassId],
  );

  const availableClassTerms = useMemo(() => {
    if (!selectedClass) return terms;
    const matching = terms.filter((t) => t.academic_year_id === selectedClass.academic_year_id);
    return matching.length > 0 ? matching : terms;
  }, [selectedClass, terms]);

  // Handle class selection in Take Register modal
  const handleTakeClassChange = async (cid: string) => {
    setTakeClassId(cid);
    setTakeSubjectId("");
    setTakeError(null);
    setRoster([]);
    if (!cid) return;

    const cls = classes.find((c) => c.id === cid);
    if (cls) {
      const matchedTerms = terms.filter((t) => t.academic_year_id === cls.academic_year_id);
      const cur = matchedTerms.find((t) => t.is_current) || matchedTerms[0] || terms[0];
      if (cur) setTakeTermId(cur.id);
    }

    setRosterLoading(true);
    try {
      const [subjectsRes, enrollRes] = await Promise.all([
        listClassSubjects(cid),
        listEnrollments({ class_id: cid, status: "ENROLLED", page_size: 100 }),
      ]);
      setClassSubjects(subjectsRes || []);
      if (subjectsRes && subjectsRes.length > 0) {
        setTakeSubjectId(subjectsRes[0].id);
      }

      const initialRoster: RosterStudent[] = (enrollRes.enrollments || []).map((e) => ({
        student_id: e.student_id,
        student_name: e.student_name || "Enrolled Student",
        student_number: e.student_number || "",
        status: "PRESENT",
        notes: "",
      }));
      setRoster(initialRoster);
    } catch (err) {
      setTakeError(
        err instanceof ApiError ? err.message : "Failed to load class curriculum or student roster.",
      );
    } finally {
      setRosterLoading(false);
    }
  };

  // Bulk status helpers in Take Register
  const markAllStatus = (status: AttendanceStatus) => {
    setRoster((prev) => prev.map((s) => ({ ...s, status })));
  };

  const updateRosterStudentStatus = (studentId: string, status: AttendanceStatus) => {
    setRoster((prev) =>
      prev.map((s) => (s.student_id === studentId ? { ...s, status } : s)),
    );
  };

  const updateRosterStudentNotes = (studentId: string, notes: string) => {
    setRoster((prev) =>
      prev.map((s) => (s.student_id === studentId ? { ...s, notes } : s)),
    );
  };

  // Submit new attendance session
  const handleTakeRegisterSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!takeClassId || !takeSubjectId || !takeTermId || !takeDate) {
      setTakeError("Please select class, subject, term, and date.");
      return;
    }
    if (roster.length === 0) {
      setTakeError("Cannot record attendance for a class with zero active enrollees.");
      return;
    }

    setTakeSubmitting(true);
    setTakeError(null);
    try {
      await createAttendanceSession({
        class_id: takeClassId,
        subject_id: takeSubjectId,
        term_id: takeTermId,
        session_date: takeDate,
        records: roster.map((r) => ({
          student_id: r.student_id,
          status: r.status,
          notes: r.notes.trim() ? r.notes.trim() : null,
        })),
      });

      setShowTakeRegister(false);
      setNotice("Attendance register successfully recorded.");
      setTakeClassId("");
      setTakeSubjectId("");
      setTakeTermId("");
      setRoster([]);
      loadSessions();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "ALREADY_RECORDED" || err.status === 409) {
          setTakeError("Attendance has already been captured for this class, subject, term, and date.");
        } else {
          setTakeError(err.message);
        }
      } else {
        setTakeError("Failed to record attendance session.");
      }
    } finally {
      setTakeSubmitting(false);
    }
  };

  // Inspect & Correct Modal
  const handleOpenDetail = async (session: AttendanceSession) => {
    setSelectedSession(session);
    setDetailModalOpen(true);
    setDetailLoading(true);
    setDetailError(null);
    setRecordFeedback({});
    try {
      const res = await getAttendanceSession(session.id);
      setSelectedSession(res.session);
      setSessionRecords(res.records || []);
    } catch (err) {
      setDetailError(err instanceof ApiError ? err.message : "Could not load session records.");
    } finally {
      setDetailLoading(false);
    }
  };

  const handleUpdateRecord = async (recordId: string, status: AttendanceStatus, notes?: string | null) => {
    setSavingRecordId(recordId);
    setRecordFeedback((prev) => ({ ...prev, [recordId]: "" }));
    try {
      const updated = await updateAttendanceRecord(recordId, {
        status,
        notes: notes !== undefined ? notes : null,
      });

      setSessionRecords((prev) =>
        prev.map((r) => (r.id === recordId ? { ...r, status: updated.status, notes: updated.notes } : r)),
      );
      setRecordFeedback((prev) => ({ ...prev, [recordId]: "Saved!" }));
      setTimeout(() => {
        setRecordFeedback((prev) => {
          const next = { ...prev };
          delete next[recordId];
          return next;
        });
      }, 2500);
    } catch (err) {
      setDetailError(err instanceof ApiError ? err.message : "Could not update record mark.");
    } finally {
      setSavingRecordId(null);
    }
  };

  // Session summary counts within Inspect modal
  const sessionCounts = useMemo(() => {
    const counts = { PRESENT: 0, ABSENT: 0, LATE: 0, EXCUSED: 0, total: sessionRecords.length };
    for (const r of sessionRecords) {
      if (r.status in counts) {
        counts[r.status as AttendanceStatus]++;
      }
    }
    return counts;
  }, [sessionRecords]);

  // Open Class Summary Modal
  const handleOpenClassSummary = async (cls: Class) => {
    setSummaryTargetClass(cls);
    setSummaryModalOpen(true);
    setSummaryLoading(true);
    setSummaryError(null);
    try {
      const sum = await getClassAttendanceSummary(cls.id, {
        term_id: filterTermId || undefined,
        subject_id: filterSubjectId || undefined,
      });
      setClassSummary(sum);
    } catch (err) {
      setSummaryError(err instanceof ApiError ? err.message : "Could not load class attendance summary.");
    } finally {
      setSummaryLoading(false);
    }
  };

  // Table columns
  const columns: Column<AttendanceSession>[] = [
    {
      key: "session_date",
      header: "Session Date",
      render: (s) => (
        <span className="font-semibold text-slate-900 font-mono text-xs sm:text-sm">
          {s.session_date}
        </span>
      ),
    },
    {
      key: "class_name",
      header: "Class",
      render: (s) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">
          {s.class_name || "—"}
        </span>
      ),
    },
    {
      key: "subject",
      header: "Subject",
      render: (s) => (
        <div className="flex flex-col">
          <span className="font-medium text-slate-800">{s.subject_name || "—"}</span>
          {s.subject_code && (
            <span className="text-[11px] font-mono text-slate-400">{s.subject_code}</span>
          )}
        </div>
      ),
    },
    {
      key: "term_name",
      header: "Term",
      hideBelow: "sm",
      render: (s) => <span className="text-xs text-slate-600">{s.term_name || "—"}</span>,
    },
    {
      key: "academic_year_name",
      header: "Academic Year",
      hideBelow: "md",
      render: (s) => (
        <span className="text-xs text-slate-500">{s.academic_year_name || "—"}</span>
      ),
    },
    {
      key: "actions",
      header: "Actions",
      className: "text-right",
      render: (s) => (
        <div className="flex items-center justify-end gap-2">
          <button
            onClick={() => handleOpenDetail(s)}
            className="rounded-lg border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 hover:text-brand-700 transition-colors"
          >
            Inspect &amp; Correct
          </button>
          {s.class_id && (
            <button
              onClick={() => {
                const targetCls = classes.find((c) => c.id === s.class_id) || {
                  id: s.class_id,
                  name: s.class_name || "Class",
                  academic_year_id: s.academic_year_id || "",
                };
                handleOpenClassSummary(targetCls as Class);
              }}
              className="rounded-lg border border-brand-100 bg-brand-50/50 px-2.5 py-1 text-xs font-medium text-brand-700 shadow-sm hover:bg-brand-50 transition-colors"
            >
              Summary
            </button>
          )}
        </div>
      ),
    },
  ];

  const resetFilters = () => {
    setFilterClassId("");
    setFilterSubjectId("");
    setFilterTermId("");
    setFilterDateFrom("");
    setFilterDateTo("");
    setSearch("");
    setPage(1);
  };

  const hasActiveFilters = Boolean(
    filterClassId || filterSubjectId || filterTermId || filterDateFrom || filterDateTo || search,
  );

  return (
    <AppShell
      title="Attendance Register"
      actions={
        canRecord && (
          <button
            onClick={() => {
              setShowTakeRegister(true);
              setTakeError(null);
            }}
            className="inline-flex items-center gap-1.5 rounded-lg bg-brand-700 px-3.5 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 transition-colors"
          >
            <span>+</span> Take Register
          </button>
        )
      }
    >
      <div className="space-y-6">
        {/* KPI Stat Cards */}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <StatCard
            label="Total Sessions Recorded"
            value={total}
            tone="sky"
            chip="All Time"
          />
          <StatCard
            label="Active Classes"
            value={classes.length}
            tone="purple"
            chip="Enrolled"
          />
          <StatCard
            label="Active Terms"
            value={terms.length}
            tone="yellow"
            chip="Calendar"
          />
          <StatCard
            label="Subjects Taught"
            value={subjects.length}
            tone="green"
            chip="Curriculum"
          />
        </div>

        {/* Notice alert */}
        {notice && (
          <div className="flex items-center justify-between rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
            <span>{notice}</span>
            <button
              onClick={() => setNotice(null)}
              className="text-xs font-semibold text-emerald-600 hover:text-emerald-800"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Directory Card */}
        <div className="space-y-4 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-base font-bold text-slate-900">Attendance Directory</h2>
              <p className="text-xs text-slate-500">
                Filter and inspect class registers across terms, subjects, and dates
              </p>
            </div>
            <div className="flex items-center gap-2">
              <SearchInput
                onSearch={setSearch}
                placeholder="Search class, subject, date..."
              />
              {hasActiveFilters && (
                <button
                  onClick={resetFilters}
                  className="rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50"
                >
                  Reset
                </button>
              )}
            </div>
          </div>

          {/* Filter Bar */}
          <div className="grid grid-cols-1 gap-3 pt-2 sm:grid-cols-2 lg:grid-cols-5">
            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Class
              </label>
              <select
                value={filterClassId}
                onChange={(e) => {
                  setFilterClassId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Classes</option>
                {classes.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Subject
              </label>
              <select
                value={filterSubjectId}
                onChange={(e) => {
                  setFilterSubjectId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Subjects</option>
                {subjects.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.code} — {s.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Term
              </label>
              <select
                value={filterTermId}
                onChange={(e) => {
                  setFilterTermId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Terms</option>
                {terms.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                From Date
              </label>
              <input
                type="date"
                value={filterDateFrom}
                onChange={(e) => {
                  setFilterDateFrom(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              />
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                To Date
              </label>
              <input
                type="date"
                value={filterDateTo}
                onChange={(e) => {
                  setFilterDateTo(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              />
            </div>
          </div>

          {/* Table */}
          {error && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-4 text-xs text-rose-700">
              {error}
            </div>
          )}

          {loading ? (
            <div className="py-12 text-center text-sm text-slate-400">Loading attendance registers…</div>
          ) : (
            <>
              <DataTable
                columns={columns}
                rows={filteredSessions}
                getRowKey={(s) => s.id}
                emptyMessage="No attendance sessions found matching the current filters."
              />
              <Pagination
                page={page}
                totalItems={total}
                pageSize={PAGE_SIZE}
                onPageChange={(newPage) => setPage(newPage)}
              />
            </>
          )}
        </div>
      </div>

      {/* Take Register Modal */}
      <Modal
        open={showTakeRegister}
        onClose={() => setShowTakeRegister(false)}
        title="Take Attendance Register"
        maxWidth="max-w-3xl"
      >
        <form onSubmit={handleTakeRegisterSubmit} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SelectField
              id="take-class"
              label="Select Class *"
              value={takeClassId}
              onChange={(e) => handleTakeClassChange(e.target.value)}
              required
            >
              <option value="">Choose a class…</option>
              {classes.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </SelectField>

            <SelectField
              id="take-subject"
              label="Subject *"
              value={takeSubjectId}
              onChange={(e) => setTakeSubjectId(e.target.value)}
              disabled={!takeClassId || classSubjects.length === 0}
              required
            >
              <option value="">
                {!takeClassId
                  ? "Select class first"
                  : classSubjects.length === 0
                  ? "No subjects assigned to class"
                  : "Choose subject…"}
              </option>
              {classSubjects.map((cs) => (
                <option key={cs.id} value={cs.id}>
                  {cs.code} — {cs.name}
                </option>
              ))}
            </SelectField>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SelectField
              id="take-term"
              label="Academic Term *"
              value={takeTermId}
              onChange={(e) => setTakeTermId(e.target.value)}
              disabled={!takeClassId}
              required
            >
              <option value="">Choose term…</option>
              {availableClassTerms.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} {t.is_current ? "(Current)" : ""}
                </option>
              ))}
            </SelectField>

            <TextField
              id="take-date"
              label="Session Date *"
              type="date"
              value={takeDate}
              onChange={(e) => setTakeDate(e.target.value)}
              required
            />
          </div>

          {/* Roster Container */}
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b border-slate-200 pb-3 mb-3">
              <div>
                <h3 className="text-xs font-bold uppercase tracking-wider text-slate-700">
                  Active Enrollees Roster
                </h3>
                <p className="text-[11px] text-slate-500">
                  {roster.length} student{roster.length === 1 ? "" : "s"} enrolled in this class
                </p>
              </div>

              {roster.length > 0 && (
                <div className="flex flex-wrap items-center gap-1.5 text-xs">
                  <span className="text-[11px] font-medium text-slate-500 mr-1">Quick Mark:</span>
                  <button
                    type="button"
                    onClick={() => markAllStatus("PRESENT")}
                    className="rounded bg-emerald-100 px-2 py-0.5 text-[11px] font-semibold text-emerald-800 hover:bg-emerald-200"
                  >
                    All Present
                  </button>
                  <button
                    type="button"
                    onClick={() => markAllStatus("ABSENT")}
                    className="rounded bg-rose-100 px-2 py-0.5 text-[11px] font-semibold text-rose-800 hover:bg-rose-200"
                  >
                    All Absent
                  </button>
                  <button
                    type="button"
                    onClick={() => markAllStatus("LATE")}
                    className="rounded bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-800 hover:bg-amber-200"
                  >
                    All Late
                  </button>
                </div>
              )}
            </div>

            {rosterLoading ? (
              <div className="py-8 text-center text-xs text-slate-400">Loading enrolled roster…</div>
            ) : !takeClassId ? (
              <div className="py-8 text-center text-xs text-slate-400">
                Please select a class to load its active students.
              </div>
            ) : roster.length === 0 ? (
              <div className="py-8 text-center text-xs text-amber-600">
                No actively enrolled students found in this class.
              </div>
            ) : (
              <div className="max-h-72 overflow-y-auto space-y-2 pr-1">
                {roster.map((student) => (
                  <div
                    key={student.student_id}
                    className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-xs sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-semibold text-slate-900">
                        {student.student_name}
                      </p>
                      <p className="text-[11px] font-mono text-slate-400">
                        {student.student_number || "No student #"}
                      </p>
                    </div>

                    <div className="flex items-center gap-1">
                      {ATTENDANCE_STATUSES.map((st) => {
                        const active = student.status === st;
                        const activeStyles = {
                          PRESENT: "bg-emerald-600 text-white font-bold shadow-xs",
                          ABSENT: "bg-rose-600 text-white font-bold shadow-xs",
                          LATE: "bg-amber-600 text-white font-bold shadow-xs",
                          EXCUSED: "bg-sky-600 text-white font-bold shadow-xs",
                        }[st];
                        const inactiveStyles =
                          "bg-slate-100 text-slate-600 hover:bg-slate-200";

                        return (
                          <button
                            key={st}
                            type="button"
                            onClick={() => updateRosterStudentStatus(student.student_id, st)}
                            className={`rounded-md px-2 py-1 text-[11px] transition-colors ${
                              active ? activeStyles : inactiveStyles
                            }`}
                          >
                            {st}
                          </button>
                        );
                      })}
                    </div>

                    <div className="sm:w-44">
                      <input
                        type="text"
                        placeholder="Optional note…"
                        value={student.notes}
                        onChange={(e) => updateRosterStudentNotes(student.student_id, e.target.value)}
                        className="w-full rounded border border-slate-200 bg-white px-2 py-1 text-xs text-slate-700 placeholder:text-slate-300 focus:border-brand-500 focus:outline-none"
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <FormActions
            error={takeError}
            submitting={takeSubmitting}
            label="Save Attendance Register"
            onCancel={() => setShowTakeRegister(false)}
          />
        </form>
      </Modal>

      {/* Inspect & Correct Modal */}
      <Modal
        open={detailModalOpen}
        onClose={() => setDetailModalOpen(false)}
        title="Inspect & Correct Attendance Register"
        maxWidth="max-w-4xl"
      >
        <div className="space-y-4">
          {/* Header Metadata */}
          {selectedSession && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-base font-bold text-slate-900">
                    {selectedSession.class_name || "Class"}
                  </span>
                  <span className="rounded bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700 border border-brand-100">
                    {selectedSession.subject_code} — {selectedSession.subject_name}
                  </span>
                </div>
                <p className="text-xs text-slate-500">
                  {selectedSession.term_name} &bull; {selectedSession.academic_year_name} &bull; Session Date:{" "}
                  <strong className="text-slate-800 font-mono">{selectedSession.session_date}</strong>
                </p>
              </div>

              {/* Attendance metrics pills */}
              <div className="flex items-center gap-2 text-xs">
                <span className="rounded-lg bg-emerald-50 border border-emerald-200 px-2.5 py-1 font-semibold text-emerald-800">
                  Present: {sessionCounts.PRESENT}
                </span>
                <span className="rounded-lg bg-rose-50 border border-rose-200 px-2.5 py-1 font-semibold text-rose-800">
                  Absent: {sessionCounts.ABSENT}
                </span>
                <span className="rounded-lg bg-amber-50 border border-amber-200 px-2.5 py-1 font-semibold text-amber-800">
                  Late: {sessionCounts.LATE}
                </span>
                <span className="rounded-lg bg-sky-50 border border-sky-200 px-2.5 py-1 font-semibold text-sky-800">
                  Excused: {sessionCounts.EXCUSED}
                </span>
                <span className="rounded-lg bg-slate-100 px-2.5 py-1 font-bold text-slate-700">
                  Rate:{" "}
                  {sessionCounts.total > 0
                    ? `${Math.round((sessionCounts.PRESENT / sessionCounts.total) * 100)}%`
                    : "0%"}
                </span>
              </div>
            </div>
          )}

          {detailError && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
              {detailError}
            </div>
          )}

          {detailLoading ? (
            <div className="py-12 text-center text-xs text-slate-400">Loading session records…</div>
          ) : sessionRecords.length === 0 ? (
            <div className="py-8 text-center text-xs text-slate-400">No attendance records in this session.</div>
          ) : (
            <div className="max-h-[50vh] overflow-y-auto space-y-2 pr-1">
              {sessionRecords.map((rec) => {
                const feedback = recordFeedback[rec.id];
                const isSaving = savingRecordId === rec.id;

                return (
                  <div
                    key={rec.id}
                    className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-xs sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="font-semibold text-xs text-slate-900">
                          {rec.student_name || "Student"}
                        </p>
                        {statusBadge(rec.status)}
                      </div>
                      <p className="text-[11px] font-mono text-slate-400">
                        {rec.student_number || "No student #"}
                      </p>
                    </div>

                    {canUpdate ? (
                      <div className="flex flex-wrap items-center gap-2">
                        <div className="flex items-center gap-1">
                          {ATTENDANCE_STATUSES.map((st) => (
                            <button
                              key={st}
                              type="button"
                              onClick={() => handleUpdateRecord(rec.id, st, rec.notes)}
                              disabled={isSaving}
                              className={`rounded px-2 py-1 text-[11px] font-medium transition-colors ${
                                rec.status === st
                                  ? "bg-brand-700 text-white font-bold"
                                  : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                              }`}
                            >
                              {st}
                            </button>
                          ))}
                        </div>

                        <input
                          type="text"
                          defaultValue={rec.notes || ""}
                          placeholder="Note…"
                          onBlur={(e) => {
                            if (e.target.value !== (rec.notes || "")) {
                              handleUpdateRecord(rec.id, rec.status, e.target.value);
                            }
                          }}
                          className="w-32 rounded border border-slate-200 px-2 py-1 text-xs text-slate-700 focus:border-brand-500 focus:outline-none"
                        />

                        {feedback && (
                          <span className="text-[11px] font-bold text-emerald-600">
                            {feedback}
                          </span>
                        )}
                        {isSaving && (
                          <span className="text-[11px] text-slate-400">Saving…</span>
                        )}
                      </div>
                    ) : (
                      <div className="text-xs text-slate-500 italic">{rec.notes || "No notes"}</div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </Modal>

      {/* Class Attendance Summary Modal */}
      <Modal
        open={summaryModalOpen}
        onClose={() => setSummaryModalOpen(false)}
        title={summaryTargetClass ? `Attendance Summary: ${summaryTargetClass.name}` : "Class Summary"}
        maxWidth="max-w-md"
      >
        <div className="space-y-4">
          {summaryError && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
              {summaryError}
            </div>
          )}

          {summaryLoading ? (
            <div className="py-12 text-center text-xs text-slate-400">Loading class summary…</div>
          ) : !classSummary ? (
            <div className="py-8 text-center text-xs text-slate-400">No summary data found.</div>
          ) : (
            <div className="space-y-4">
              <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-center">
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-400">
                  Total Records Logged
                </span>
                <p className="mt-1 text-3xl font-extrabold text-slate-900">
                  {classSummary.total_records}
                </p>
                {classSummary.total_records > 0 && (
                  <p className="mt-1 text-xs font-medium text-emerald-700">
                    Overall Present Rate:{" "}
                    {Math.round(
                      ((classSummary.counts.PRESENT || 0) / classSummary.total_records) * 100,
                    )}
                    %
                  </p>
                )}
              </div>

              {/* Counts Grid */}
              <div className="grid grid-cols-2 gap-3">
                <div className="rounded-lg border border-emerald-200 bg-emerald-50/60 p-3">
                  <span className="text-[11px] font-bold uppercase text-emerald-700">Present</span>
                  <p className="text-xl font-bold text-emerald-900">
                    {classSummary.counts.PRESENT || 0}
                  </p>
                </div>
                <div className="rounded-lg border border-rose-200 bg-rose-50/60 p-3">
                  <span className="text-[11px] font-bold uppercase text-rose-700">Absent</span>
                  <p className="text-xl font-bold text-rose-900">
                    {classSummary.counts.ABSENT || 0}
                  </p>
                </div>
                <div className="rounded-lg border border-amber-200 bg-amber-50/60 p-3">
                  <span className="text-[11px] font-bold uppercase text-amber-700">Late</span>
                  <p className="text-xl font-bold text-amber-900">
                    {classSummary.counts.LATE || 0}
                  </p>
                </div>
                <div className="rounded-lg border border-sky-200 bg-sky-50/60 p-3">
                  <span className="text-[11px] font-bold uppercase text-sky-700">Excused</span>
                  <p className="text-xl font-bold text-sky-900">
                    {classSummary.counts.EXCUSED || 0}
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </Modal>
    </AppShell>
  );
}

export default function AttendancePage() {
  return (
    <ProtectedRoute requiredPermission="attendance.view">
      <AttendanceWorkspace />
    </ProtectedRoute>
  );
}
