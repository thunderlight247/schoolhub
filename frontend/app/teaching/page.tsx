"use client";

// Teaching Assignments (M27, part 2): which teacher teaches which subject in
// which class. Create validates through the backend chain (subject must be
// in the class curriculum via class_subjects — M8) and each teacher/class/
// subject triple is unique. Permissions mirror M10: class.view to read,
// class.update to create/remove.

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { PersonCell } from "@/components/ui/stat-card";
import { SelectField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  createTeacherAssignment,
  deleteTeacherAssignment,
  listTeacherAssignments,
} from "@/services/enrollments";
import { listClassSubjects, listClasses, listSubjects } from "@/services/academics";
import { listStaff } from "@/services/staff";
import { TeacherAssignment } from "@/types/enrollments";
import { Class, ClassSubject, Subject } from "@/types/academics";
import { StaffListItem } from "@/types/staff";

const PAGE_SIZE = 10;

function TeachingAssignments() {
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission("class.update");

  const [rows, setRows] = useState<TeacherAssignment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [classId, setClassId] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [classes, setClasses] = useState<Class[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [staff, setStaff] = useState<StaffListItem[]>([]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listTeacherAssignments({
        search,
        class_id: classId || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setRows(res.assignments);
      setTotal(res.pagination.total);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load teaching assignments.");
    } finally {
      setLoading(false);
    }
  }, [search, classId, page]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    listClasses()
      .then(setClasses)
      .catch(() => {});
    listSubjects()
      .then(setSubjects)
      .catch(() => {});
    listStaff({ page: 1, page_size: 100 })
      .then((res) => setStaff(res.staff))
      .catch(() => {});
  }, []);

  const remove = async (assignment: TeacherAssignment) => {
    setError(null);
    setNotice(null);
    try {
      await deleteTeacherAssignment(assignment.id);
      setNotice(
        `${assignment.teacher_name ?? "Teacher"} unassigned from ${assignment.class_name ?? "class"}.`,
      );
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not remove the assignment.");
    }
  };

  // --- create modal ---
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ teacher_id: "", class_id: "", subject_id: "" });
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [curriculum, setCurriculum] = useState<ClassSubject[]>([]);

  useEffect(() => {
    if (!form.class_id) {
      setCurriculum([]);
      return;
    }
    listClassSubjects(form.class_id)
      .then(setCurriculum)
      .catch(() => setCurriculum([]));
  }, [form.class_id]);

  const openCreate = () => {
    setForm({ teacher_id: "", class_id: "", subject_id: "" });
    setFormError(null);
    setShowCreate(true);
  };

  const submitCreate = async () => {
    if (!form.teacher_id || !form.class_id || !form.subject_id) {
      setFormError("Teacher, class, and subject are all required.");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await createTeacherAssignment({
        teacher_id: form.teacher_id,
        class_id: form.class_id,
        subject_id: form.subject_id,
      });
      setShowCreate(false);
      setNotice("Teacher assigned.");
      setPage(1);
      load();
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not create the assignment.");
    } finally {
      setSubmitting(false);
    }
  };

  const columns: Column<TeacherAssignment>[] = [
    {
      key: "teacher",
      header: "Teacher",
      render: (r) => {
        const [firstName, ...rest] = (r.teacher_name ?? "Unknown").split(" ");
        return (
          <Link href={`/staff/${r.teacher_id}`} className="block">
            <PersonCell
              firstName={firstName}
              lastName={rest.join(" ") || "—"}
              subtitle={r.staff_number}
            />
          </Link>
        );
      },
    },
    { key: "class_name", header: "Class", render: (r) => r.class_name ?? "—" },
    {
      key: "subject_name",
      header: "Subject",
      render: (r) => (
        <span className="text-sm text-slate-700">
          <span className="mr-2 font-mono text-xs text-slate-400">{r.subject_code}</span>
          {r.subject_name ?? "—"}
        </span>
      ),
    },
    {
      key: "academic_year_name",
      header: "Year",
      hideBelow: "md",
      render: (r) => r.academic_year_name ?? "—",
    },
    ...(canUpdate
      ? [
          {
            key: "actions",
            header: "Actions",
            render: (r: TeacherAssignment) => (
              <button
                onClick={() => remove(r)}
                title="Remove assignment"
                aria-label={`Unassign ${r.teacher_name ?? "teacher"} from ${r.class_name ?? "class"}`}
                className="flex h-7 w-7 items-center justify-center rounded-full bg-red-50 text-red-600 hover:bg-red-100"
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src="/icons/delete.png" alt="" width={13} height={13} />
              </button>
            ),
          } as Column<TeacherAssignment>,
        ]
      : []),
  ];

  return (
    <ProtectedRoute requiredPermission="class.view">
      <AppShell
        title="Teaching Assignments"
        actions={
          canUpdate && (
            <button
              onClick={openCreate}
              className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
            >
              + Assign teacher
            </button>
          )
        }
      >
        <div className="space-y-6">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                Teaching Assignments
              </h1>
              <p className="mt-1 text-sm text-slate-500">
                {total} assignment{total === 1 ? "" : "s"} — teacher → class → subject coverage.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <SearchInput
                placeholder="Search teacher or class…"
                onSearch={(q) => {
                  setSearch(q);
                  setPage(1);
                }}
                className="max-w-xs"
              />
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
            </div>
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

        {/* Assign teacher modal */}
        <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Assign a teacher">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              submitCreate();
            }}
            className="space-y-4"
          >
            <SelectField
              label="Teacher *"
              required
              value={form.teacher_id}
              onChange={(e) => setForm({ ...form, teacher_id: e.target.value })}
            >
              <option value="">— select teacher —</option>
              {staff.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.first_name} {s.last_name} ({s.staff_number})
                  {s.position_title ? ` — ${s.position_title}` : ""}
                </option>
              ))}
            </SelectField>
            <SelectField
              label="Class *"
              required
              value={form.class_id}
              onChange={(e) => setForm({ ...form, class_id: e.target.value, subject_id: "" })}
            >
              <option value="">— select class —</option>
              {classes.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                  {c.academic_year_name ? ` (${c.academic_year_name})` : ""}
                </option>
              ))}
            </SelectField>
            <SelectField
              label="Subject * (class curriculum)"
              required
              value={form.subject_id}
              onChange={(e) => setForm({ ...form, subject_id: e.target.value })}
            >
              <option value="">— select subject —</option>
              {curriculum.map((cs) => (
                <option key={cs.id} value={cs.id}>
                  {cs.code} — {cs.name}
                </option>
              ))}
            </SelectField>
            <p className="text-xs text-slate-400">
              Only subjects in the class&apos;s curriculum can be assigned, and a teacher can hold
              each class-subject pairing once.
            </p>
            <FormActions
              error={formError}
              submitting={submitting}
              label="Assign teacher"
              onCancel={() => setShowCreate(false)}
            />
          </form>
        </Modal>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function TeachingAssignmentsPage() {
  return <TeachingAssignments />;
}
