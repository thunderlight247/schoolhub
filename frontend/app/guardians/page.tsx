"use client";

// Guardians Directory (M25): list + create, mirroring the students
// directory's professional patterns. Guardian creation is the prerequisite
// step before linking a guardian to a student (the backend keeps those
// operations deliberately separate).

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
import { TextField, SelectField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import { createGuardian, listGuardians } from "@/services/students";
import { GuardianListItem } from "@/types/students";
import { GENDERS } from "@/types/staff";

const PAGE_SIZE = 10;

function GuardiansDirectory() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("guardian.create");

  const [rows, setRows] = useState<GuardianListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({
    first_name: "",
    last_name: "",
    other_names: "",
    gender: "",
    date_of_birth: "",
    phone: "",
    email: "",
    address: "",
    occupation: "",
  });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listGuardians({ search, page, page_size: PAGE_SIZE });
      setRows(res.guardians);
      setTotal(res.pagination.total);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load guardians.");
    } finally {
      setLoading(false);
    }
  }, [search, page]);

  useEffect(() => {
    load();
  }, [load]);

  const submitCreate = async (e: FormEvent) => {
    e.preventDefault();
    setCreating(true);
    setCreateError(null);
    try {
      await createGuardian({
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        other_names: form.other_names.trim() || undefined,
        gender: form.gender || undefined,
        date_of_birth: form.date_of_birth || undefined,
        phone: form.phone.trim() || undefined,
        email: form.email.trim() || undefined,
        address: form.address.trim() || undefined,
        occupation: form.occupation.trim() || undefined,
      });
      setShowCreate(false);
      setForm((f) => ({
        ...f,
        first_name: "",
        last_name: "",
        other_names: "",
        gender: "",
        date_of_birth: "",
        phone: "",
        email: "",
        address: "",
        occupation: "",
      }));
      load();
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Could not register the guardian.");
    } finally {
      setCreating(false);
    }
  };

  const columns: Column<GuardianListItem>[] = [
    {
      key: "info",
      header: "Guardian",
      render: (r) => (
        <Link href={`/guardians/${r.id}`} className="block">
          <PersonCell
            firstName={r.first_name}
            lastName={r.last_name}
            subtitle={r.occupation ?? undefined}
          />
        </Link>
      ),
    },
    {
      key: "phone",
      header: "Phone",
      hideBelow: "md",
      render: (r) => r.phone ?? "—",
    },
    {
      key: "email",
      header: "Email",
      hideBelow: "lg",
      render: (r) => r.email ?? "—",
      className: "text-xs",
    },
    {
      key: "student_count",
      header: "Children",
      render: (r) => (
        <span className="inline-flex rounded-full bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-800 ring-1 ring-inset ring-brand-100">
          {r.student_count}
        </span>
      ),
    },
  ];

  return (
    <ProtectedRoute requiredPermission="student.view">
      <AppShell
        title="Guardians"
        actions={
          canCreate && (
            <button
              onClick={() => setShowCreate(true)}
              className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
            >
              + Register guardian
            </button>
          )
        }
      >
        <div className="space-y-6">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                Guardians &amp; Parents
              </h1>
              <p className="mt-1 text-sm text-slate-500">
                Registered guardians and parents. Link them to students from a student&apos;s
                profile.
              </p>
            </div>
            <SearchInput
              placeholder="Search guardians…"
              onSearch={(q) => {
                setSearch(q);
                setPage(1);
              }}
              className="max-w-xs"
            />
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

        {/* Register guardian modal */}
        <Modal open={showCreate} onClose={() => setShowCreate(false)} title="Register a new guardian">
          <form onSubmit={submitCreate} className="space-y-4">
            <p className="text-xs font-medium text-slate-400">Personal information</p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <TextField
                id="guardian-first-name"
                label="First name *"
                required
                value={form.first_name}
                onChange={(e) => setForm({ ...form, first_name: e.target.value })}
              />
              <TextField
                id="guardian-last-name"
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
              <TextField
                label="Occupation"
                value={form.occupation}
                onChange={(e) => setForm({ ...form, occupation: e.target.value })}
              />
            </div>
            <TextField
              label="Address"
              value={form.address}
              onChange={(e) => setForm({ ...form, address: e.target.value })}
            />
            <FormActions
              error={createError}
              submitting={creating}
              label="Register guardian"
              onCancel={() => setShowCreate(false)}
            />
          </form>
        </Modal>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function GuardiansPage() {
  return <GuardiansDirectory />;
}
