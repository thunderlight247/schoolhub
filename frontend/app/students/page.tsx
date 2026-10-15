"use client";

// Students Directory (M25): list, search, status filter, pagination, and
// enrollment of new students. First screen built at the professional quality
// bar (see remember.md): responsive column hiding, zebra rows, avatar info
// cells, numbered pagination, and real loading/empty/error states.

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { StatCard } from "@/components/ui/stat-card";
import { PersonCell } from "@/components/ui/stat-card";
import { TextField, SelectField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import { createStudent, listStudents } from "@/services/students";
import {

  STUDENT_STATUSES,
  StudentListItem,
} from "@/types/students";
import { GENDERS } from "@/types/staff";

const PAGE_SIZE = 10;

function statusBadge(status: string) {
  return status === "ACTIVE" ? (
    <span className="inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/20">
      Active
    </span>
  ) : (
    <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-inset ring-slate-500/20">
      Archived
    </span>
  );
}

function StudentsDirectory() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("student.create");

  const [rows, setRows] = useState<StudentListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [activeCount, setActiveCount] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create modal state
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({
    student_number: "",
    first_name: "",
    last_name: "",
    other_names: "",
    gender: "",
    date_of_birth: "",
    phone: "",
    email: "",
    address: "",
    admission_date: new Date().toISOString().slice(0, 10),
  });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listStudents({
        search,
        status: status || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setRows(res.students);
      setTotal(res.pagination.total);
      // Active count = total minus archived on the current filter; when
      // unfiltered, ask once for the archived-only page size to compute it.
      if (!status && !search) {
        const archived = await listStudents({ status: "ARCHIVED", page: 1, page_size: 1 });
        setActiveCount(res.pagination.total - archived.pagination.total);
      } else {
        setActiveCount(res.pagination.total);
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load students.");
    } finally {
      setLoading(false);
    }
  }, [search, status, page]);

  useEffect(() => {
    load();
  }, [load]);

  const submitCreate = async (e: FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      await createStudent({
        student_number: form.student_number.trim(),
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        other_names: form.other_names.trim() || undefined,
        gender: form.gender || undefined,
        date_of_birth: form.date_of_birth || undefined,
        phone: form.phone.trim() || undefined,
        email: form.email.trim() || undefined,
        address: form.address.trim() || undefined,
        admission_date: form.admission_date || undefined,
      });
      setShowCreate(false);
      setForm((f) => ({
        ...f,
        student_number: "",
        first_name: "",
        last_name: "",
        other_names: "",
        gender: "",
        date_of_birth: "",
        phone: "",
        email: "",
        address: "",
      }));
      load();
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Could not enroll the student.");
    } finally {
      setCreating(false);
    }
  };

  const columns: Column<StudentListItem>[] = [
    {
      key: "info",
      header: "Student",
      render: (r) => (
        <Link href={`/students/${r.id}`} className="block">
          <PersonCell
            firstName={r.first_name}
            lastName={r.last_name}
            subtitle={`Admitted ${r.admission_date?.slice(0, 10) ?? "—"}`}
          />
        </Link>
      ),
    },
    {
      key: "student_number",
      header: "Student №",
      className: "font-mono text-xs",
      hideBelow: "md",
      render: (r) => <span className="font-mono text-xs">{r.student_number}</span>,
    },
    { key: "status", header: "Status", render: (r) => statusBadge(r.status) },
    {
      key: "admission_date",
      header: "Admitted",
      hideBelow: "lg",
      render: (r) => r.admission_date?.slice(0, 10) ?? "—",
    },
    {
      key: "created_at",
      header: "Registered",
      hideBelow: "xl",
      render: (r) => new Date(r.created_at).toLocaleDateString(),
    },
  ];

  const filterControls = (
    <>
      <SearchInput
        placeholder="Search name or student number…"
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
        <option value="">All statuses</option>
        {STUDENT_STATUSES.map((s) => (
          <option key={s} value={s}>
            {s === "ACTIVE" ? "Active" : "Archived"}
          </option>
        ))}
      </select>
    </>
  );

  return (
    <ProtectedRoute requiredPermission="student.view">
      <AppShell
        title="Students"
        actions={
          canCreate && (
            <button
              onClick={() => setShowCreate(true)}
              className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
            >
              + Enroll student
            </button>
          )
        }
      >
        <div className="space-y-6">
          {/* Stat cards (reference-repo pattern) */}
          <div className="flex flex-wrap gap-4">
            <StatCard label="active students" value={activeCount} tone="purple" />
            <StatCard label="total records" value={total} tone="sky" />
            <StatCard label="archived" value={Math.max(0, total - activeCount)} tone="yellow" />
          </div>

          {/* Filter bar */}
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                Student Records
              </h1>
              <p className="mt-1 text-sm text-slate-500">
                Search, review, and manage student enrollment records.
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">{filterControls}</div>
          </div>

          {error && (
            <div className="flex items-center justify-between rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              <span>{error}</span>
              <button onClick={load} className="text-xs font-semibold underline hover:text-red-900">
                Try Again
              </button>
            </div>
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

        {/* Enroll student modal */}
        <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Enroll a new student">
          <form onSubmit={submitCreate} className="space-y-4">
            <p className="text-xs font-medium text-slate-400">Personal information</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <TextField
                id="student-number"
                label="Student number *"
                required
                value={form.student_number}
                onChange={(e) => setForm({ ...form, student_number: e.target.value })}
                placeholder="STU-2026-013"
              />
              <TextField
                id="student-first-name"
                label="First name *"
                required
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
              />
              <TextField
                id="student-last-name"
                label="Last name *"
                required
                value={form.last_name}
                onChange={(e) => setForm({ ...form, last_name: e.target.value })}
              />
              <TextField
                label="Other names"
                value={form.other_names}
                onChange={(e) => setForm({ ...form, other_names: e.target.value })}
              />
              <SelectField
                label="Gender"
                value={form.gender}
                onChange={(e) => setForm({ ...form, gender: e.target.value })}
              >
                <option value="">—</option>
                {GENDERS.map((g) => (
                  <option key={g} value={g}>
                    {g}
                  </option>
                ))}
              </SelectField>
              <TextField
                label="Date of birth"
                type="date"
                value={form.date_of_birth}
                onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })}
              />
            </div>
            <p className="pt-1 text-xs font-medium text-slate-400">Contact</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <TextField
                label="Phone"
                value={form.phone}
                onChange={(e) => setForm({ ...form, phone: e.target.value })}
              />
              <TextField
                label="Email"
                type="email"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
              />
            </div>
            <TextField
              label="Address"
              value={form.address}
              onChange={(e) => setForm({ ...form, address: e.target.value })}
            />
            <TextField
              label="Admission date"
              type="date"
              value={form.admission_date}
              onChange={(e) => setForm({ ...form, admission_date: e.target.value })}
            />
            <FormActions
              error={createError}
              submitting={creating}
              label="Enroll student"
              onCancel={() => setShowCreate(false)}
            />
          </form>
        </Modal>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function StudentsPage() {
  return <StudentsDirectory />;
}
