"use client";

// Enrollments (M27, part 1): the school-wide enrollment register. Filters
// (status, class, academic year, search), pagination, enrolling a student
// into a class (search-and-pick like the guardian linker), and status
// transitions (ENROLLED / COMPLETED / TRANSFERRED / WITHDRAWN) per row.
// Permissions mirror the M9 route gates: view (any staff), create on
// class.create, transitions on class.update.

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { PersonCell } from "@/components/ui/stat-card";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import { listEnrollments, createEnrollment, setEnrollmentStatus } from "@/services/enrollments";
import { listAcademicYears, listClasses } from "@/services/academics";
import { listStudents } from "@/services/students";
import {
  ENROLLMENT_STATUSES,
  Enrollment,
  EnrollmentStatus,
} from "@/types/enrollments";
import { AcademicYear, Class } from "@/types/academics";

const PAGE_SIZE = 10;

function statusBadge(status: string) {
  const styles: Record<string, string> = {
    ENROLLED: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
    COMPLETED: "bg-sky-50 text-sky-700 ring-sky-600/20",
    TRANSFERRED: "bg-amber-50 text-amber-700 ring-amber-600/20",
    WITHDRAWN: "bg-slate-100 text-slate-600 ring-slate-500/20",
  };
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${styles[status] ?? styles.WITHDRAWN}`}
    >
      {status.charAt(0) + status.slice(1).toLowerCase()}
    </span>
  );
}

function EnrollmentsDirectory() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("class.create");
  const canTransition = hasPermission("class.update");

  const [rows, setRows] = useState<Enrollment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [classId, setClassId] = useState("");
  const [yearId, setYearId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [years, setYears] = useState<AcademicYear[]>([]);
  const [classes, setClasses] = useState<Class[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listEnrollments({
        search,
        status: status || undefined,
        class_id: classId || undefined,
        academic_year_id: yearId || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setRows(res.enrollments);
      setTotal(res.pagination.total);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load enrollments.");
    } finally {
      setLoading(false);
    }
  }, [search, status, classId, yearId, page]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    listAcademicYears()
      .then(setYears)
      .catch(() => {});
    listClasses()
      .then(setClasses)
      .catch(() => {});
  }, []);

  const transition = async (enrollment: Enrollment, next: EnrollmentStatus) => {
    if (next === enrollment.status) return;
    setError(null);
    setNotice(null);
    try {
      const updated = await setEnrollmentStatus(enrollment.id, next);
      setRows((rs) => rs.map((r) => (r.id === updated.id ? updated : r)));
      setNotice(
        `${updated.student_name ?? "Student"}'s enrollment marked ${next.toLowerCase()}.`,
      );
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update the enrollment.");
      load(); // re-sync with server state
    }
  };

  // --- enroll modal ---
  const [showEnroll, setShowEnroll] = useState(false);
  const [studentQuery, setStudentQuery] = useState("");
  const [studentResults, setStudentResults] = useState<
    Array<{ id: string; first_name: string; last_name: string; student_number: string }>
  >([]);
  const [studentSearchLoading, setStudentSearchLoading] = useState(false);
  const [enrollForm, setEnrollForm] = useState({
    student_id: "",
    academic_year_id: "",
    class_id: "",
    enrollment_date: new Date().toISOString().slice(0, 10),
    notes: "",
  });
  const [enrollError, setEnrollError] = useState<string | null>(null);
  const [enrollSubmitting, setEnrollSubmitting] = useState(false);

  useEffect(() => {
    if (!showEnroll) return;
    const q = studentQuery.trim();
    setStudentSearchLoading(true);
    const timer = setTimeout(async () => {
      try {
        const res = await listStudents({ search: q || undefined, page: 1, page_size: 5 });
        setStudentResults(res.students);
      } catch {
        setStudentResults([]);
      } finally {
        setStudentSearchLoading(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [studentQuery, showEnroll]);

  const openEnroll = () => {
    setEnrollForm((f) => ({
      ...f,
      student_id: "",
      academic_year_id: years.find((y) => y.is_current)?.id ?? years[0]?.id ?? "",
      class_id: "",
      notes: "",
    }));
    setStudentQuery("");
    setStudentResults([]);
    setEnrollError(null);
    setShowEnroll(true);
  };

  const submitEnroll = async (e: FormEvent) => {
    e.preventDefault();
    if (!enrollForm.student_id) {
      setEnrollError("Select a student to enroll.");
      return;
    }
    if (!enrollForm.class_id) {
      setEnrollError("Select a class.");
      return;
    }
    setEnrollSubmitting(true);
    setEnrollError(null);
    try {
      const created = await createEnrollment({
        student_id: enrollForm.student_id,
        academic_year_id: enrollForm.academic_year_id,
        class_id: enrollForm.class_id,
        enrollment_date: enrollForm.enrollment_date || undefined,
        notes: enrollForm.notes.trim() || undefined,
      });
      setShowEnroll(false);
      setNotice(
        `${created.student_name ?? "Student"} enrolled in ${created.class_name ?? "class"}.`,
      );
      setPage(1);
      load();
    } catch (err) {
      setEnrollError(err instanceof ApiError ? err.message : "Could not enroll the student.");
    } finally {
      setEnrollSubmitting(false);
    }
  };

  const columns: Column<Enrollment>[] = [
    {
      key: "student",
      header: "Student",
      render: (r) => {
        const [firstName, ...rest] = (r.student_name ?? "Unknown").split(" ");
        return (
          <Link href={`/students/${r.student_id}`} className="block">
            <PersonCell
              firstName={firstName}
              lastName={rest.join(" ") || "—"}
              subtitle={[r.student_number, r.class_name].filter(Boolean).join(" · ")}
            />
          </Link>
        );
      },
    },
    {
      key: "academic_year_name",
      header: "Year",
      hideBelow: "md",
      render: (r) => r.academic_year_name ?? "—",
    },
    { key: "status", header: "Status", render: (r) => statusBadge(r.status) },
    {
      key: "enrollment_date",
      header: "Enrolled",
      hideBelow: "lg",
      render: (r) => r.enrollment_date,
    },
    {
      key: "exit_date",
      header: "Exit date",
      hideBelow: "xl",
      render: (r) => r.exit_date ?? "—",
    },
    ...(canTransition
      ? [
          {
            key: "actions",
            header: "Status",
            render: (r: Enrollment) => (
              <select
                value={r.status}
                onChange={(e) => transition(r, e.target.value as EnrollmentStatus)}
                aria-label={`Change enrollment status for ${r.student_name ?? "student"}`}
                className="rounded-md border border-slate-200 bg-white px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-brand-600"
              >
                {ENROLLMENT_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            ),
          } as Column<Enrollment>,
        ]
      : []),
  ];

  const filterControls = (
    <>
      <SearchInput
        placeholder="Search student…"
        onSearch={(q) => {
          setSearch(q);
          setPage(1);
        }}
        className="max-w-xs"
      />
      <select
        value={status}
        onChange={(e) => {
          setStatus(e.target.value);
          setPage(1);
        }}
        aria-label="Filter by status"
        className="rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 focus:outline-none focus:ring-brand-600"
      >
        <option value="">Any status</option>
        {ENROLLMENT_STATUSES.map((s) => (
          <option key={s} value={s}>
            {s}
          </option>
        ))}
      </select>
      <select
        value={classId}
        onChange={(e) => {
          setClassId(e.target.value);
          setPage(1);
        }}
        aria-label="Filter by class"
        className="rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 focus:outline-none focus:ring-brand-600"
      >
        <option value="">All classes</option>
        {classes.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
      <select
        value={yearId}
        onChange={(e) => {
          setYearId(e.target.value);
          setPage(1);
        }}
        aria-label="Filter by academic year"
        className="rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 focus:outline-none focus:ring-brand-600"
      >
        <option value="">All years</option>
        {years.map((y) => (
          <option key={y.id} value={y.id}>
            {y.name}
          </option>
        ))}
      </select>
    </>
  );

  return (
    <ProtectedRoute requiredPermission="class.view">
      <AppShell
        title="Enrollments"
        actions={
          canCreate && (
            <button
              onClick={openEnroll}
              className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
            >
              + Enroll student
            </button>
          )
        }
      >
        <div className="space-y-6">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                Enrollment Register
              </h1>
              <p className="mt-1 text-sm text-slate-500">
                {total} enrollment record{total === 1 ? "" : "s"} — class membership by academic
                year.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">{filterControls}</div>
          </div>

          {(error || notice) && (
            <p
              role="status"
              className={`rounded-lg border px-3 py-2 text-xs font-medium ${
                error
                  ? "border-red-200 bg-red-50 text-red-700"
                  : "border-emerald-200 bg-emerald-50 text-emerald-700"
              }`}
            >
              {error ?? notice}
            </p>
          )}

          {loading ? (
            <div className="flex h-56 items-center justify-center rounded-xl border border-slate-200 bg-white">
              <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
            </div>
          ) : (
            <DataTable columns={columns} rows={rows} getRowKey={(r) => r.id} />
          )}

          <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
        </div>

        {/* Enroll modal */}
        <Modal open={showEnroll} onClose={() => setShowEnroll(false)} title="Enroll a student in a class">
          <form onSubmit={submitEnroll} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
                Find student
              </label>
              <input
                type="search"
                value={studentQuery}
                onChange={(e) => setStudentQuery(e.target.value)}
                placeholder="Search by name…"
                className="mt-1 w-full rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 outline-none focus:ring-brand-600"
              />
              {studentSearchLoading && <p className="mt-2 text-xs text-slate-400">Searching…</p>}
              <div className="mt-2 space-y-1">
                {studentResults.map((s) => (
                  <button
                    type="button"
                    key={s.id}
                    onClick={() => setEnrollForm((f) => ({ ...f, student_id: s.id }))}
                    className={`flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm ${
                      enrollForm.student_id === s.id
                        ? "border-brand-600 bg-brand-50"
                        : "border-slate-200 hover:bg-slate-50"
                    }`}
                  >
                    <span className="font-medium text-slate-800">
                      {s.first_name} {s.last_name}
                    </span>
                    <span className="font-mono text-xs text-slate-400">{s.student_number}</span>
                  </button>
                ))}
                {!studentSearchLoading && studentResults.length === 0 && (
                  <p className="rounded-lg border border-dashed border-slate-200 p-3 text-xs text-slate-500">
                    No students found. Enroll them first from the{" "}
                    <Link href="/students" className="font-semibold text-brand-700 underline">
                      Students
                    </Link>{" "}
                    page.
                  </p>
                )}
              </div>
            </div>
            <SelectField
              label="Academic year *"
              required
              value={enrollForm.academic_year_id}
              onChange={(e) => setEnrollForm({ ...enrollForm, academic_year_id: e.target.value })}
            >
              {years.map((y) => (
                <option key={y.id} value={y.id}>
                  {y.name}
                  {y.is_current ? " (current)" : ""}
                </option>
              ))}
            </SelectField>
            <SelectField
              label="Class *"
              required
              value={enrollForm.class_id}
              onChange={(e) => setEnrollForm({ ...enrollForm, class_id: e.target.value })}
            >
              <option value="">— select class —</option>
              {classes
                .filter((c) => !enrollForm.academic_year_id || c.academic_year_id === enrollForm.academic_year_id)
                .map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                    {c.academic_year_name ? ` (${c.academic_year_name})` : ""}
                  </option>
                ))}
            </SelectField>
            <TextField
              label="Enrollment date"
              type="date"
              value={enrollForm.enrollment_date}
              onChange={(e) => setEnrollForm({ ...enrollForm, enrollment_date: e.target.value })}
            />
            <TextAreaField
              label="Notes"
              value={enrollForm.notes}
              onChange={(e) => setEnrollForm({ ...enrollForm, notes: e.target.value })}
            />
            <FormActions
              error={enrollError}
              submitting={enrollSubmitting}
              label="Enroll student"
              onCancel={() => setShowEnroll(false)}
            />
          </form>
        </Modal>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function EnrollmentsPage() {
  return <EnrollmentsDirectory />;
}
