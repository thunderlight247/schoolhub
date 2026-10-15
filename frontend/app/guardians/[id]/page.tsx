"use client";

// Guardian profile (M25): contact record, edit, and the linked-children
// list with relationship + enrollment status. Parents' portal access is
// derived from these links, so each child card links back to the student.

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { Avatar } from "@/components/ui/stat-card";
import { ApiError } from "@/lib/api";
import { createGuardianAccount, getGuardian, updateGuardian } from "@/services/students";
import { GuardianDetail } from "@/types/students";
import { GENDERS } from "@/types/staff";

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium text-slate-800">{value || "—"}</dd>
    </div>
  );
}

function GuardianProfile() {
  const { id } = useParams<{ id: string }>();
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission("student.update");
  // M34: parent-access provisioning rides guardian.create — the inviter
  // role (teacher or admin) mints the parent login (backend route gate).
  const canProvision = hasPermission("guardian.create");

  const [guardian, setGuardian] = useState<GuardianDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [showEdit, setShowEdit] = useState(false);
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
  const [editError, setEditError] = useState<string | null>(null);

  // Parent-access (Create account) state.
  const [showAccount, setShowAccount] = useState(false);
  const [accountForm, setAccountForm] = useState({ email: "", password: "" });
  const [accountError, setAccountError] = useState<string | null>(null);
  const [accountCreated, setAccountCreated] = useState<string | null>(null);

  const loadGuardian = useCallback(async () => {
    try {
      setLoadError(null);
      setGuardian(await getGuardian(id));
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : "Could not load the guardian.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadGuardian();
  }, [loadGuardian]);

  const openEdit = () => {
    if (!guardian) return;
    setForm({
      first_name: guardian.first_name,
      last_name: guardian.last_name,
      other_names: guardian.other_names ?? "",
      gender: guardian.gender ?? "",
      date_of_birth: guardian.date_of_birth?.slice(0, 10) ?? "",
      phone: guardian.phone ?? "",
      email: guardian.email ?? "",
      address: guardian.address ?? "",
      occupation: guardian.occupation ?? "",
    });
    setEditError(null);
    setShowEdit(true);
  };

  const submitEdit = async (e: FormEvent) => {
    e.preventDefault();
    setEditError(null);
    try {
      const updated = await updateGuardian(id, {
        first_name: form.first_name.trim(),
        last_name: form.last_name.trim(),
        other_names: form.other_names.trim() || null,
        gender: form.gender || undefined,
        date_of_birth: form.date_of_birth || undefined,
        phone: form.phone.trim() || null,
        email: form.email.trim() || null,
        address: form.address.trim() || null,
        occupation: form.occupation.trim() || null,
      });
      setGuardian(updated);
      setShowEdit(false);
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : "Could not save changes.");
    }
  };

  // --- Create parent access ---
  const openAccount = () => {
    if (!guardian) return;
    setAccountForm({ email: guardian.email ?? "", password: "" });
    setAccountError(null);
    setShowAccount(true);
  };

  const submitAccount = async (e: FormEvent) => {
    e.preventDefault();
    setAccountError(null);
    try {
      await createGuardianAccount(id, {
        email: accountForm.email.trim(),
        password: accountForm.password,
      });
      setShowAccount(false);
      setAccountCreated(
        `Parent access created. Share the username (${accountForm.email.trim()}) with ${guardian?.first_name ?? "the guardian"} — they sign up with it on the login page and an invitation email is on its way.`,
      );
    } catch (err) {
      setAccountError(err instanceof ApiError ? err.message : "Could not create the parent account.");
    }
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
      </div>
    );
  }

  if (loadError || !guardian) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
        {loadError ?? "Guardian not found."}{" "}
        <Link href="/guardians" className="font-semibold underline">
          Back to directory
        </Link>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <Link href="/guardians" className="text-xs font-medium text-slate-500 hover:text-brand-700">
            ← Guardians
          </Link>
          <div className="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-4">
              <Avatar firstName={guardian.first_name} lastName={guardian.last_name} size="lg" />
              <div>
                <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                  {guardian.first_name} {guardian.last_name}
                </h1>
                <p className="mt-1 text-sm text-slate-500">
                  {guardian.occupation ?? "Guardian"}
                  {guardian.students.length > 0 &&
                    ` · ${guardian.students.length} linked student${
                      guardian.students.length === 1 ? "" : "s"
                    }`}
                </p>
              </div>
            </div>
            {canUpdate && (
              <button
                onClick={openEdit}
                className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 shadow-sm hover:bg-slate-50"
              >
                Edit details
              </button>
            )}          </div>
        </div>

        {accountCreated && (
          <p className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm font-medium text-emerald-700">
            {accountCreated}
          </p>
        )}

        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
          {/* Contact */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-sm font-bold text-slate-900">Contact information</h2>
            <dl className="grid grid-cols-2 gap-4">
              <Field label="First name" value={guardian.first_name} />
              <Field label="Last name" value={guardian.last_name} />
              <Field label="Other names" value={guardian.other_names ?? ""} />
              <Field label="Gender" value={guardian.gender ?? ""} />
              <Field label="Phone" value={guardian.phone ?? ""} />
              <Field label="Email" value={guardian.email ?? ""} />
              <Field label="Occupation" value={guardian.occupation ?? ""} />
            </dl>
            <div className="mt-4 border-t border-slate-100 pt-4">
              <Field label="Address" value={guardian.address ?? ""} />
            </div>
          </section>

          {/* Linked children */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-bold text-slate-900">
                Linked students{" "}
                <span className="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600">
                  {guardian.students.length}
                </span>
              </h2>
            </div>
            {guardian.students.length === 0 ? (
              <p className="rounded-lg border border-dashed border-slate-200 p-4 text-sm text-slate-500">
                No students linked yet. Open a student&apos;s profile to link this guardian.
              </p>
            ) : (
              <ul className="space-y-3">
                {guardian.students.map((s) => (
                  <li
                    key={s.student_id}
                    className="flex items-center justify-between gap-3 rounded-lg border border-slate-100 p-3"
                  >
                    <div className="flex items-center gap-3">
                      <Avatar firstName={s.first_name} lastName={s.last_name} />
                      <div>
                        <Link
                          href={`/students/${s.student_id}`}
                          className="text-sm font-semibold text-slate-800 hover:text-brand-700 hover:underline"
                        >
                          {s.first_name} {s.last_name}
                        </Link>
                        <p className="text-xs text-slate-500">
                          <span className="font-mono">{s.student_number}</span> · {s.relationship}
                        </p>
                      </div>
                    </div>
                    {s.status === "ACTIVE" ? (
                      <span className="inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/20">
                        Active
                      </span>
                    ) : (
                      <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-inset ring-slate-500/20">
                        Archived
                      </span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
          {/* Parent portal access */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm lg:col-span-2">
            <div className="mb-4 flex items-center justify-between">
              <div>
                <h2 className="text-sm font-bold text-slate-900">Parent portal access</h2>
                <p className="text-xs text-slate-500">
                  Creates their login. They sign in with the email/username you choose,
                  and blocking later is done from Access Control.
                </p>
              </div>
              {canProvision && !guardian.account && (
                <button
                  onClick={openAccount}
                  className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
                >
                  + Create parent access
                </button>
              )}
            </div>
            {guardian.account ? (
              <div className="flex flex-wrap items-center gap-3">
                <span className="rounded-full bg-brand-50 px-2.5 py-1 text-xs font-semibold text-brand-800 ring-1 ring-inset ring-brand-100">
                  Username: {guardian.account.email}
                </span>
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
                    guardian.account.status === "ACTIVE"
                      ? "bg-emerald-50 text-emerald-700 ring-emerald-600/20"
                      : "bg-red-50 text-red-700 ring-red-600/20"
                  }`}
                >
                  {guardian.account.status}
                </span>
                <span className="text-xs text-slate-500">
                  Roles: {guardian.account.roles.join(", ")} · manage/block from{" "}
                  <Link href="/access" className="font-semibold text-brand-700 hover:underline">
                    Access Control
                  </Link>
                </span>
              </div>
            ) : (
              <p className="text-sm text-slate-500">
                No parent login yet. After creating one, the guardian appears under
                Access Control → Parents, where you can also block or unblock them.
              </p>
            )}
          </section>
        </div>
      </div>

      {/* Account modal */}
      <Modal open={showAccount} onClose={() => setShowAccount(false)} title="Create parent access">
        <form onSubmit={submitAccount} className="space-y-4">
          <TextField
            label="Username (their email) *"
            type="email"
            required
            value={accountForm.email}
            onChange={(e) => setAccountForm({ ...accountForm, email: e.target.value })}
          />
          <p className="text-xs text-slate-400">
            This is the username the parent will use to sign up and log in. A sign-up
            invitation is emailed to this address when Clerk is configured.
          </p>
          <TextField
            label="Temporary password *"
            type="password"
            required
            minLength={8}
            value={accountForm.password}
            onChange={(e) => setAccountForm({ ...accountForm, password: e.target.value })}
          />
          <p className="text-xs text-slate-400">
            Minimum 8 characters. Share it securely — it is hashed server-side and
            never emailed.
          </p>
          <FormActions
            error={accountError}
            submitting={false}
            label="Create access"
            onCancel={() => setShowAccount(false)}
          />
        </form>
      </Modal>

      {/* Edit modal */}
      <Modal open={showEdit} onClose={() => setShowEdit(false)} title="Edit guardian details">
        <form onSubmit={submitEdit} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <TextField
              label="First name *"
              required
              value={form.first_name}
              onChange={(e) => setForm({ ...form, first_name: e.target.value })}
            />
            <TextField
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
          <TextAreaField
            label="Address"
            value={form.address}
            onChange={(e) => setForm({ ...form, address: e.target.value })}
          />
          <FormActions error={editError} submitting={false} onCancel={() => setShowEdit(false)} />
        </form>
      </Modal>
    </>
  );
}

export default function GuardianProfilePage() {
  return (
    <ProtectedRoute requiredPermission="student.view">
      <AppShell title="Guardian Profile">
        <GuardianProfile />
      </AppShell>
    </ProtectedRoute>
  );
}
