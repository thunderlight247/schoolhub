"use client";

// Departments & Positions (M24, part 3): reference-data management for the
// staff domain. Read side is staff.view; creation is staff.update (matching
// the backend route gates). Tables + create modals on the M23 kit.

import { useCallback, useEffect, useState, type FormEvent } from "react";
import Link from "next/link";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import { createDepartment, createPosition, listDepartments, listPositions } from "@/services/staff";
import { Department, POSITION_CATEGORIES, StaffPosition } from "@/types/staff";

type Tab = "departments" | "positions";

function StaffSetup() {
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission("staff.update");

  const [tab, setTab] = useState<Tab>("departments");
  const [departments, setDepartments] = useState<Department[]>([]);
  const [positions, setPositions] = useState<StaffPosition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [d, p] = await Promise.all([listDepartments(), listPositions()]);
      setDepartments(d);
      setPositions(p);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load reference data.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // create department state
  const [showDept, setShowDept] = useState(false);
  const [deptForm, setDeptForm] = useState({ name: "", description: "" });
  const [deptError, setDeptError] = useState<string | null>(null);
  const submitDepartment = async (e: FormEvent) => {
    e.preventDefault();
    setDeptError(null);
    try {
      await createDepartment(deptForm.name.trim(), deptForm.description.trim() || undefined);
      setShowDept(false);
      setDeptForm({ name: "", description: "" });
      load();
    } catch (err) {
      setDeptError(err instanceof ApiError ? err.message : "Could not create department.");
    }
  };

  // create position state
  const [showPos, setShowPos] = useState(false);
  const [posForm, setPosForm] = useState({ title: "", category: "TEACHING", description: "" });
  const [posError, setPosError] = useState<string | null>(null);
  const submitPosition = async (e: FormEvent) => {
    e.preventDefault();
    setPosError(null);
    try {
      await createPosition(
        posForm.title.trim(),
        posForm.category,
        posForm.description.trim() || undefined,
      );
      setShowPos(false);
      setPosForm({ title: "", category: "TEACHING", description: "" });
      load();
    } catch (err) {
      setPosError(err instanceof ApiError ? err.message : "Could not create position.");
    }
  };

  const deptColumns: Column<Department>[] = [
    { key: "name", header: "Department", className: "font-medium text-slate-800" },
    { key: "description", header: "Description", render: (d) => d.description ?? "—" },
  ];

  const posColumns: Column<StaffPosition>[] = [
    { key: "title", header: "Position", className: "font-medium text-slate-800" },
    { key: "category", header: "Category", render: (p) => p.category },
    { key: "description", header: "Description", render: (p) => p.description ?? "—" },
  ];

  return (
    <>
      <div className="space-y-6">
        <div>
          <Link href="/staff" className="text-xs font-medium text-slate-500 hover:text-brand-700">
            ← Staff Directory
          </Link>
          <h1 className="mt-2 text-2xl font-bold tracking-tight text-slate-900">
            Departments &amp; Positions
          </h1>
          <p className="mt-1 text-sm text-slate-500">
            Reference data used when assigning staff members. Creation is available to staff
            managers.
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-1 rounded-lg border border-slate-200 bg-white p-1 shadow-sm w-fit">
          {(["departments", "positions"] as Tab[]).map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className={`rounded-md px-4 py-1.5 text-sm font-medium capitalize ${
                tab === t ? "bg-brand-50 text-brand-900" : "text-slate-600 hover:bg-slate-50"
              }`}
            >
              {t}
            </button>
          ))}
        </div>

        {error && (
          <div className="flex items-center justify-between rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
            <span>{error}</span>
            <button onClick={load} className="text-xs font-semibold underline hover:text-red-900">
              Try Again
            </button>
          </div>
        )}

        {tab === "departments" ? (
          <>
            <div className="flex justify-end">
              {canUpdate && (
                <button
                  onClick={() => setShowDept(true)}
                  className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
                >
                  + Add department
                </button>
              )}
            </div>
            {loading ? (
              <div className="flex h-40 items-center justify-center">
                <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
              </div>
            ) : (
              <DataTable
                columns={deptColumns}
                rows={departments}
                getRowKey={(d) => d.id}
                emptyMessage="No departments yet."
              />
            )}
          </>
        ) : (
          <>
            <div className="flex justify-end">
              {canUpdate && (
                <button
                  onClick={() => setShowPos(true)}
                  className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
                >
                  + Add position
                </button>
              )}
            </div>
            {loading ? (
              <div className="flex h-40 items-center justify-center">
                <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
              </div>
            ) : (
              <DataTable
                columns={posColumns}
                rows={positions}
                getRowKey={(p) => p.id}
                emptyMessage="No positions yet."
              />
            )}
          </>
        )}
      </div>

      {/* Create department modal */}
      <Modal open={showDept} onClose={() => setShowDept(false)} title="Add department">
        <form onSubmit={submitDepartment} className="space-y-4">
          <TextField
            label="Name *"
            id="department-name"
            required
            value={deptForm.name}
            onChange={(e) => setDeptForm({ ...deptForm, name: e.target.value })}
            placeholder="Mathematics"
          />
          <TextAreaField
            label="Description"
            value={deptForm.description}
            onChange={(e) => setDeptForm({ ...deptForm, description: e.target.value })}
          />
          <FormActions error={deptError} submitting={false} label="Create department" onCancel={() => setShowDept(false)} />
        </form>
      </Modal>

      {/* Create position modal */}
      <Modal open={showPos} onClose={() => setShowPos(false)} title="Add staff position">
        <form onSubmit={submitPosition} className="space-y-4">
          <TextField
            label="Title *"
            required
            value={posForm.title}
            onChange={(e) => setPosForm({ ...posForm, title: e.target.value })}
            placeholder="Mathematics Teacher"
          />
          <SelectField
            label="Category *"
            value={posForm.category}
            onChange={(e) => setPosForm({ ...posForm, category: e.target.value })}
          >
            {POSITION_CATEGORIES.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </SelectField>
          <TextAreaField
            label="Description"
            value={posForm.description}
            onChange={(e) => setPosForm({ ...posForm, description: e.target.value })}
          />
          <FormActions error={posError} submitting={false} label="Create position" onCancel={() => setShowPos(false)} />
        </form>
      </Modal>
    </>
  );
}

export default function StaffSetupPage() {
  return (
    <ProtectedRoute requiredPermission="staff.view">
      <AppShell title="Departments & Positions">
        <StaffSetup />
      </AppShell>
    </ProtectedRoute>
  );
}
