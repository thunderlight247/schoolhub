"use client";

// Staff Directory (M24, part 1): the staff list screen — search, department /
// status / category filters, pagination, and staff creation. Built entirely
// on the M23 scaffolding: AppShell, ProtectedRoute (staff.view), DataTable,
// SearchInput, Pagination, Modal, and the form primitives.

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  createStaff,
  listDepartments,
  listPositions,
  listStaff,
} from "@/services/staff";
import {
  Department,
  EMPLOYMENT_STATUSES,
  GENDERS,
  POSITION_CATEGORIES,
  StaffListItem,
  StaffPosition,
} from "@/types/staff";

const PAGE_SIZE = 10;

function statusBadge(status: string) {
  const styles: Record<string, string> = {
    ACTIVE: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
    INACTIVE: "bg-slate-100 text-slate-600 ring-slate-500/20",
    SUSPENDED: "bg-red-50 text-red-700 ring-red-600/20",
    ON_LEAVE: "bg-amber-50 text-amber-700 ring-amber-600/20",
  };
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
        styles[status] ?? styles.INACTIVE
      }`}
    >
      {status.replace("_", " ")}
    </span>
  );
}

function StaffDirectory() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("staff.create");
  const canManageReference = hasPermission("staff.update");

  // List state
  const [rows, setRows] = useState<StaffListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [departmentId, setDepartmentId] = useState("");
  const [employmentStatus, setEmploymentStatus] = useState("");
  const [positionCategory, setPositionCategory] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Reference data for filters + create form
  const [departments, setDepartments] = useState<Department[]>([]);
  const [positions, setPositions] = useState<StaffPosition[]>([]);

  // Create modal state
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({
    staff_number: "",
    first_name: "",
    last_name: "",
    other_names: "",
    gender: "",
    date_of_birth: "",
    phone: "",
    email: "",
    address: "",
    department_id: "",
    position_id: "",
    hired_date: "",
  });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listStaff({
        search,
        department_id: departmentId || undefined,
        employment_status: employmentStatus || undefined,
        position_category: positionCategory || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setRows(res.staff);
      setTotal(res.pagination.total);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load staff.");
    } finally {
      setLoading(false);
    }
  }, [search, departmentId, employmentStatus, positionCategory, page]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    Promise.all([listDepartments(), listPositions()])
      .then(([d, p]) => {
        setDepartments(d);
        setPositions(p);
      })
      .catch(() => {
        // Filters and the create form degrade to manual entry; the list still works.
      });
  }, []);

  const resetFilters = () => {
    setSearch("");
    setDepartmentId("");
    setEmploymentStatus("");
    setPositionCategory("");
    setPage(1);
  };

  const submitCreate = async (e: FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      await createStaff({
        staff_number: form.staff_number.trim(),
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        other_names: form.other_names.trim() || undefined,
        gender: form.gender || undefined,
        date_of_birth: form.date_of_birth || undefined,
        phone: form.phone.trim() || undefined,
        email: form.email.trim() || undefined,
        address: form.address.trim() || undefined,
        department_id: form.department_id || undefined,
        position_id: form.position_id || undefined,
        hired_date: form.hired_date || undefined,
      });
      setShowCreate(false);
      setForm({
        staff_number: "",
        first_name: "",
        last_name: "",
        other_names: "",
        gender: "",
        date_of_birth: "",
        phone: "",
        email: "",
        address: "",
        department_id: "",
        position_id: "",
        hired_date: "",
      });
      load();
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Could not create staff member.");
    } finally {
      setCreating(false);
    }
  };

  const columns: Column<StaffListItem>[] = [
    { key: "staff_number", header: "Staff №", className: "font-mono text-xs" },
    {
      key: "name",
      header: "Name",
      render: (r) => (
        <Link
          href={`/staff/${r.id}`}
          className="font-medium text-brand-800 hover:text-brand-600 hover:underline"
        >
          {r.first_name} {r.last_name}
        </Link>
      ),
    },
    { key: "department_name", header: "Department", render: (r) => r.department_name ?? "—" },
    { key: "position_title", header: "Position", render: (r) => r.position_title ?? "—" },
    {
      key: "position_category",
      header: "Category",
      render: (r) => r.position_category ?? "—",
      className: "text-xs text-slate-500",
    },
    { key: "employment_status", header: "Status", render: (r) => statusBadge(r.employment_status) },
    {
      key: "has_account",
      header: "Account",
      render: (r) =>
        r.has_account ? (
          <span className="text-emerald-600">✓</span>
        ) : (
          <span className="text-slate-300">—</span>
        ),
    },
  ];

  const filterControls = (
    <>
      <SearchInput
        placeholder="Search name or staff number…"
        onSearch={(q) => {
          setSearch(q);
          setPage(1);
        }}
        className="w-64"
      />
      <select
        value={departmentId}
        onChange={(e) => {
          setDepartmentId(e.target.value);
          setPage(1);
        }}
        className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-brand-600 focus:outline-none"
      >
        <option value="">All departments</option>
        {departments.map((d) => (
          <option key={d.id} value={d.id}>
            {d.name}
          </option>
        ))}
      </select>
      <select
        value={employmentStatus}
        onChange={(e) => {
          setEmploymentStatus(e.target.value);
          setPage(1);
        }}
        className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-brand-600 focus:outline-none"
      >
        <option value="">Any status</option>
        {EMPLOYMENT_STATUSES.map((s) => (
          <option key={s} value={s}>
            {s.replace("_", " ")}
          </option>
        ))}
      </select>
      <select
        value={positionCategory}
        onChange={(e) => {
          setPositionCategory(e.target.value);
          setPage(1);
        }}
        className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-brand-600 focus:outline-none"
      >
        <option value="">Any category</option>
        {POSITION_CATEGORIES.map((c) => (
          <option key={c} value={c}>
            {c}
          </option>
        ))}
      </select>
    </>
  );

  return (
    <ProtectedRoute requiredPermission="staff.view">
      <AppShell
        title="Staff Directory"
        actions={
          <>
            {canManageReference && (
              <Link
                href="/staff/setup"
                className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50"
              >
                Departments &amp; Positions
              </Link>
            )}
            {canCreate && (
              <button
                onClick={() => setShowCreate(true)}
                className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
              >
                + Add staff member
              </button>
            )}
          </>
        }
      >
        <div className="space-y-6">
          {/* Header */}
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">Staff Directory</h1>
              <p className="mt-1 text-sm text-slate-500">
                {total} staff member{total === 1 ? "" : "s"} — search, filter, and manage
                employment records.
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

          {/* Active filter summary */}
          {(search || departmentId || employmentStatus || positionCategory) && !loading && (
            <p className="text-xs text-slate-500">
              Filters active.{" "}
              <button onClick={resetFilters} className="font-semibold text-brand-700 hover:underline">
                Clear all
              </button>
            </p>
          )}

          {/* Table */}
          {loading ? (
            <div className="flex h-48 items-center justify-center">
              <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
            </div>
          ) : (
            <DataTable columns={columns} rows={rows} getRowKey={(r) => r.id} />
          )}

          <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
        </div>

        {/* Create staff modal */}
        <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Add staff member">
          <form onSubmit={submitCreate} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <TextField
                label="Staff number *"
                id="staff-number"
                required
                value={form.staff_number}
                onChange={(e) => setForm({ ...form, staff_number: e.target.value })}
                placeholder="STF-010"
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
                label="First name *"
                id="first-name"
                required
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
              />
              <TextField
                label="Last name *"
                id="last-name"
                required
                value={form.last_name}
                onChange={(e) => setForm({ ...form, last_name: e.target.value })}
              />
              <TextField
                label="Other names"
                value={form.other_names}
                onChange={(e) => setForm({ ...form, other_names: e.target.value })}
              />
              <TextField
                label="Date of birth"
                type="date"
                value={form.date_of_birth}
                onChange={(e) => setForm({ ...form, date_of_birth: e.target.value })}
              />
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
              <SelectField
                label="Department"
                value={form.department_id}
                onChange={(e) => setForm({ ...form, department_id: e.target.value })}
              >
                <option value="">—</option>
                {departments.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </SelectField>
              <SelectField
                label="Position"
                value={form.position_id}
                onChange={(e) => setForm({ ...form, position_id: e.target.value })}
              >
                <option value="">—</option>
                {positions.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.title}
                  </option>
                ))}
              </SelectField>
            </div>
            <TextField
              label="Hired date"
              type="date"
              value={form.hired_date}
              onChange={(e) => setForm({ ...form, hired_date: e.target.value })}
            />
            <TextAreaField
              label="Address"
              value={form.address}
              onChange={(e) => setForm({ ...form, address: e.target.value })}
            />
            <FormActions
              error={createError}
              submitting={creating}
              label="Create staff member"
              onCancel={() => setShowCreate(false)}
            />
          </form>
        </Modal>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function StaffPage() {
  return <StaffDirectory />;
}
