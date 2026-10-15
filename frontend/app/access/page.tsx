"use client";

// Access Control (expert pass follow-up). Two sections:
//
//  1. "Administrators" (MASTER_ADMIN only — backend gates /admins on
//     admin.manage + the MASTER_ADMIN role): every admin with a
//     Remove-admin action, guarded against self-demotion and
//     last-admin mistakes server-side; the UI mirrors those rules.
//  2. "Account access by role" (admins with account.block): everyone
//     holding TEACHER/STAFF/PARENT/STUDENT with a Block/Unblock toggle.
//     Blocking suspends users.status, which auth.ValidateSession re-checks
//     on every request — the lockout is immediate, even mid-session.
//
// Backend errors (LAST_ADMIN, SELF_DEMOTION, SELF_BLOCK,
// ADMIN_NOT_BLOCKABLE) surface verbatim in the row action.

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { Modal } from "@/components/ui/modal";
import { TextField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  blockAccount,
  blockInvitee,
  createAdmin,
  listAccountsByRole,
  listAdmins,
  listMyInvitees,
  revokeAdminRole,
  unblockAccount,
  unblockInvitee,
} from "@/services/access";
import {
  BLOCKABLE_ROLES,
  AdminAccountRow,
  BlockableRole,
  InviteeRow,
  RoleAccountRow,
} from "@/types/access";

const ROLE_LABELS: Record<BlockableRole, string> = {
  TEACHER: "Teachers",
  STAFF: "Staff",
  PARENT: "Parents",
  STUDENT: "Students",
};

function StatusChip({ status }: { status: string }) {
  const active = status === "ACTIVE";
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
        active
          ? "bg-emerald-50 text-emerald-700 ring-emerald-600/20"
          : "bg-red-50 text-red-700 ring-red-600/20"
      }`}
    >
      {status}
    </span>
  );
}

function AccessControlPage() {
  const { hasRole, hasPermission, user } = useAuth();
  const meId = user?.id;
  const canManageAdmins = hasPermission("admin.manage") && hasRole("MASTER_ADMIN");
  const canBlock = hasPermission("account.block");

  const [admins, setAdmins] = useState<AdminAccountRow[]>([]);
  const [role, setRole] = useState<BlockableRole>("PARENT");
  const [accounts, setAccounts] = useState<RoleAccountRow[]>([]);
  // M34: "every inviter can see invitees" — loads for EVERY signed-in
  // user (teachers included); admins simply also see the sections below.
  const [invitees, setInvitees] = useState<InviteeRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Per-row action state so buttons show busy/error without a page reload.
  const [busyId, setBusyId] = useState<string | null>(null);
  const [rowError, setRowError] = useState<{ id: string; message: string } | null>(null);

  // M34: master-admin "Add admin" composer. No password field — the
  // backend provisions the account and the Clerk invitation email lets the
  // recipient choose their own password.
  const [showAddAdmin, setShowAddAdmin] = useState(false);
  const [newAdmin, setNewAdmin] = useState({ email: "", first_name: "", last_name: "" });
  const [addBusy, setAddBusy] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);
  const [addSuccess, setAddSuccess] = useState<string | null>(null);

  const handleAddAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    setAddBusy(true);
    setAddError(null);
    setAddSuccess(null);
    try {
      await createAdmin(newAdmin);
      setAddSuccess(
        `${newAdmin.email} added. They'll receive a Clerk email to create their password.`,
      );
      setNewAdmin({ email: "", first_name: "", last_name: "" });
      await loadAdmins();
    } catch (err) {
      setAddError(err instanceof ApiError ? err.message : "Could not add the admin.");
    } finally {
      setAddBusy(false);
    }
  };

  const loadAdmins = useCallback(async () => {
    if (!canManageAdmins) return;
    try {
      const res = await listAdmins();
      setAdmins(res.admins);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load admins.");
    }
  }, [canManageAdmins]);

  const loadAccounts = useCallback(async () => {
    if (!canBlock) return;
    try {
      const res = await listAccountsByRole(role);
      setAccounts(res.accounts);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load accounts.");
    }
  }, [canBlock, role]);

  const loadInvitees = useCallback(async () => {
    try {
      const res = await listMyInvitees();
      setInvitees(res.invitees);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load your invitees.");
    }
  }, []);

  useEffect(() => {
    const run = async () => {
      setLoading(true);
      setError(null);
      await Promise.all([loadAdmins(), loadAccounts(), loadInvitees()]);
      setLoading(false);
    };
    run();
  }, [loadAdmins, loadAccounts, loadInvitees]);

  const withRow = async (id: string, action: () => Promise<void>) => {
    setBusyId(id);
    setRowError(null);
    try {
      await action();
      await Promise.all([loadAdmins(), loadAccounts(), loadInvitees()]);
    } catch (err) {
      setRowError({
        id,
        message: err instanceof ApiError ? err.message : "Action failed.",
      });
    } finally {
      setBusyId(null);
    }
  };

  const toggleAccount = (row: RoleAccountRow) =>
    withRow(row.id, () =>
      row.status === "SUSPENDED" ? unblockAccount(row.id) : blockAccount(row.id),
    );

  const demoteAdmin = (row: AdminAccountRow) => withRow(row.id, () => revokeAdminRole(row.id));

  // M34: inviter-side suspend/restore — strictly scoped to the rows the
  // backend returned (they carry users.invited_by = the caller).
  const toggleInvitee = (row: InviteeRow) =>
    withRow(row.id, () =>
      row.status === "SUSPENDED" ? unblockInvitee(row.id) : blockInvitee(row.id),
    );

  // M34: rows from GET /my-invitees — the caller's own invitees. Actions
  // hit PUT /my-invitees/:id/status, so no account.block permission needed
  // (the backend scopes every row to users.invited_by = the caller).
  const inviteeColumns: Column<InviteeRow>[] = [
    {
      key: "name",
      header: "Invited account",
      render: (r) => (
        <div>
          <div className="font-medium text-slate-800">
            {r.first_name} {r.last_name}
            <span className="ml-2 rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold text-slate-600">
              {r.roles.includes(",") ? r.roles.split(",").join(" · ") : r.roles || "pending roles"}
            </span>
          </div>
          <div className="text-xs text-slate-500">{r.email}</div>
        </div>
      ),
    },
    { key: "status", header: "Status", render: (r) => <StatusChip status={r.status} /> },
    {
      key: "actions",
      header: "",
      className: "text-right",
      render: (r) => (
        <button
          disabled={busyId === r.id}
          onClick={() => toggleInvitee(r)}
          className={`rounded-md border px-2.5 py-1 text-xs font-semibold disabled:opacity-50 ${
            r.status === "SUSPENDED"
              ? "border-emerald-200 text-emerald-700 hover:bg-emerald-50"
              : "border-red-200 text-red-600 hover:bg-red-50"
          }`}
        >
          {busyId === r.id ? "Working…" : r.status === "SUSPENDED" ? "Restore access" : "Block"}
        </button>
      ),
    },
  ];

  const adminColumns: Column<AdminAccountRow>[] = [
    {
      key: "name",
      header: "Admin",
      render: (r) => (
        <div>
          <div className="font-medium text-slate-800">
            {r.first_name} {r.last_name}
            {r.is_master && (
              <span className="ml-2 rounded-full bg-violet-100 px-1.5 py-0.5 text-[10px] font-bold uppercase text-violet-700">
                Master
              </span>
            )}
          </div>
          <div className="text-xs text-slate-500">{r.email}</div>
        </div>
      ),
    },
    { key: "status", header: "Status", render: (r) => <StatusChip status={r.status} /> },
    {
      key: "last_login",
      header: "Last sign-in",
      hideBelow: "md",
      render: (r) =>
        r.last_login_at ? new Date(r.last_login_at).toLocaleString() : "Never",
    },
    {
      key: "actions",
      header: "",
      className: "text-right",
      render: (r) =>
        r.is_master || r.id === meId ? null : (
          <button
            disabled={busyId === r.id}
            onClick={() => demoteAdmin(r)}
            className="rounded-md border border-red-200 px-2.5 py-1 text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
          >
            {busyId === r.id ? "Working…" : "Remove admin"}
          </button>
        ),
    },
  ];  const accountColumns: Column<RoleAccountRow>[] = [
    {
      key: "name",
      header: "Name",
      render: (r) => {
        // Parents link back to the guardians directory; the rest to staff.
        const base = role === "PARENT" ? "/guardians" : "/staff";
        return (
          <Link href={`${base}/${r.person_id}`} className="block hover:text-brand-700">
            <div className="font-medium text-slate-800">
              {r.first_name} {r.last_name}
            </div>
            <div className="text-xs text-slate-500">{r.email}</div>
          </Link>
        );
      },
    },
    {
      key: "status", header: "Account status", render: (r) => <StatusChip status={r.status} /> },
    {
      key: "last_login",
      header: "Last sign-in",
      hideBelow: "md",
      render: (r) =>
        r.last_login_at ? new Date(r.last_login_at).toLocaleString() : "Never",
    },
    {
      key: "block",
      header: "",
      className: "text-right",
      render: (r) => (
        <button
          disabled={busyId === r.id}
          onClick={() => toggleAccount(r)}
          className={`rounded-md border px-2.5 py-1 text-xs font-semibold disabled:opacity-50 ${
            r.status === "SUSPENDED"
              ? "border-emerald-200 text-emerald-700 hover:bg-emerald-50"
              : "border-red-200 text-red-600 hover:bg-red-50"
          }`}
        >
          {busyId === r.id
            ? "Working…"
            : r.status === "SUSPENDED"
              ? "Unblock"
              : "Block"}
        </button>
      ),
    },
  ];

  return (
    <ProtectedRoute>
      <AppShell title="Access Control">
        <div className="space-y-8">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">Access Control</h1>
              <p className="mt-1 text-sm text-slate-500">
                Manage who can sign in. Blocking suspends the account and ends its
                current session on the next request — even if the user is signed in
                right now.
              </p>
            </div>
          </div>

          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {error}
            </div>
          )}

          {/* M34: "every inviter can see invitees and block account" —
              shown to whoever signs in; the backend scopes rows to
              u.invited_by = the caller, so a teacher sees the parents they
              invited, a master admin sees the admins they minted. */}
          <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
            <div className="border-b border-slate-100 px-6 py-4">
              <h2 className="text-base font-semibold text-slate-900">People you invited</h2>
              <p className="text-xs text-slate-500">
                Accounts YOU provisioned. You can suspend (block) or restore them at
                any time — the lockout ends their live session on the next request.
              </p>
            </div>
            <div className="p-4">
              <DataTable
                columns={inviteeColumns}
                rows={invitees}
                getRowKey={(r) => r.id}
                emptyMessage="No accounts invited yet — invite a colleague, staff member, or parent and they will show up here."
              />
            </div>
          </section>

          {canManageAdmins && (
            <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
              <div className="flex flex-col gap-3 border-b border-slate-100 px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 className="text-base font-semibold text-slate-900">Administrators</h2>
                  <p className="text-xs text-slate-500">
                    Master admins manage this list. Removing an admin revokes their ADMIN
                    role — they lose admin access but keep any portals their person record
                    supports.
                  </p>
                </div>
                <button
                  onClick={() => {
                    setShowAddAdmin(true);
                    setAddError(null);
                    setAddSuccess(null);
                  }}
                  className="whitespace-nowrap rounded-lg bg-brand-700 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-brand-600"
                >
                  + Add admin
                </button>
              </div>
              <div className="p-4">
                <DataTable
                  columns={adminColumns}
                  rows={admins}
                  getRowKey={(r) => r.id}
                  emptyMessage="No admin accounts registered yet."
                />
              </div>
            </section>
          )}

          <Modal
            open={showAddAdmin}
            onClose={() => setShowAddAdmin(false)}
            title="Add admin"
          >
            <form onSubmit={handleAddAdmin} className="space-y-4">
              <p className="text-xs text-slate-500">
                The new admin is created immediately and receives a Clerk email to
                choose their own password. Never send them a password yourself.
              </p>
              <TextField
                label="Email"
                type="email"
                required
                value={newAdmin.email}
                onChange={(e) => setNewAdmin((f) => ({ ...f, email: e.target.value }))}
                placeholder="name@yourschool.com"
              />
              <TextField
                label="First name"
                required
                value={newAdmin.first_name}
                onChange={(e) => setNewAdmin((f) => ({ ...f, first_name: e.target.value }))}
              />
              <TextField
                label="Last name"
                required
                value={newAdmin.last_name}
                onChange={(e) => setNewAdmin((f) => ({ ...f, last_name: e.target.value }))}
              />
              <FormActions
                error={addError}
                success={addSuccess}
                submitting={addBusy}
                label="Add admin"
                onCancel={() => setShowAddAdmin(false)}
              />
            </form>
          </Modal>

          {canBlock && (
            <section className="rounded-xl border border-slate-200 bg-white shadow-sm">
              <div className="flex flex-col gap-3 border-b border-slate-100 px-6 py-4 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <h2 className="text-base font-semibold text-slate-900">
                    Parent &amp; teacher access
                  </h2>
                  <p className="text-xs text-slate-500">
                    Block or restore sign-in for any {ROLE_LABELS[role].toLowerCase()}{" "}
                    account, regardless of whether they are signed in.
                  </p>
                </div>
                <div className="flex gap-1 rounded-lg bg-slate-100 p-1">
                  {BLOCKABLE_ROLES.map((r) => (
                    <button
                      key={r}
                      onClick={() => setRole(r)}
                      className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                        role === r
                          ? "bg-white text-brand-800 shadow-sm"
                          : "text-slate-500 hover:text-slate-800"
                      }`}
                    >
                      {ROLE_LABELS[r]}
                    </button>
                  ))}
                </div>
              </div>

              {rowError && (
                <p className="mx-4 mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
                  {rowError.message}
                </p>
              )}

              <div className="p-4">
                <DataTable
                  columns={accountColumns}
                  rows={accounts}
                  getRowKey={(r) => r.id}
                  emptyMessage={`No ${ROLE_LABELS[role].toLowerCase()} have login accounts yet.`}
                />
              </div>
            </section>
          )}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}

export default function AccessPage() {
  return <AccessControlPage />;
}
