"use client";

// Staff profile (M24, part 2): one staff member's full record — personal
// details edit (PATCH), department/position assignment (full-replace PUT),
// employment status (staff.deactivate), and login account management
// (create account / replace roles). Gated per-action with hasPermission so
// the UI mirrors the backend's permission split.

import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  createStaffAccount,
  getStaff,
  listDepartments,
  listPositions,
  listRoles,
  setAccountRoles,
  setEmploymentStatus,
  setStaffAssignment,
  updateStaff,
} from "@/services/staff";
import {
  Department,
  EMPLOYMENT_STATUSES,
  GENDERS,
  POSITION_CATEGORIES,
  Role,
  StaffDetail,
  StaffPosition,
  SYSTEM_ROLES,
} from "@/types/staff";

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium text-slate-800">{value || "—"}</dd>
    </div>
  );
}

function StaffProfile() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { hasPermission, hasRole } = useAuth();
  const canUpdate = hasPermission("staff.update");
  const canDeactivate = hasPermission("staff.deactivate");
  // Account management rides the backend's ADMIN-gated RBAC routes (/roles).
  const isAdmin = hasRole("ADMIN");

  const [staff, setStaff] = useState<StaffDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [departments, setDepartments] = useState<Department[]>([]);
  const [positions, setPositions] = useState<StaffPosition[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);

  // shared action-error state per section
  const [personalError, setPersonalError] = useState<string | null>(null);
  const [assignmentError, setAssignmentError] = useState<string | null>(null);
  const [accountError, setAccountError] = useState<string | null>(null);
  const [accountSuccess, setAccountSuccess] = useState<string | null>(null);

  // edit modals
  const [showPersonal, setShowPersonal] = useState(false);
  const [showAssignment, setShowAssignment] = useState(false);
  const [showAccount, setShowAccount] = useState(false);
  const [showRoles, setShowRoles] = useState(false);

  const loadStaff = useCallback(async () => {
    try {
      setLoadError(null);
      const s = await getStaff(id);
      setStaff(s);
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : "Could not load staff member.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadStaff();
  }, [loadStaff]);

  useEffect(() => {
    listDepartments()
      .then(setDepartments)
      .catch(() => {});
    listPositions()
      .then(setPositions)
      .catch(() => {});
    if (isAdmin) {
      listRoles()
        .then(setRoles)
        .catch(() => {});
    }
  }, [isAdmin]);

  // --- personal edit state ---
  const [personal, setPersonal] = useState({
    first_name: "",
    last_name: "",
    other_names: "",
    gender: "",
    date_of_birth: "",
    phone: "",
    email: "",
    address: "",
  });
  const openPersonal = () => {
    if (!staff) return;
    setPersonal({
      first_name: staff.first_name,
      last_name: staff.last_name,
      other_names: staff.other_names ?? "",
      gender: staff.gender ?? "",
      date_of_birth: staff.date_of_birth?.slice(0, 10) ?? "",
      phone: staff.phone ?? "",
      email: staff.email ?? "",
      address: staff.address ?? "",
    });
    setPersonalError(null);
    setShowPersonal(true);
  };
  const submitPersonal = async (e: FormEvent) => {
    e.preventDefault();
    setPersonalError(null);
    try {
      const updated = await updateStaff(id, {
        first_name: personal.first_name.trim(),
        last_name: personal.last_name.trim(),
        other_names: personal.other_names.trim() || null,
        gender: personal.gender || undefined,
        date_of_birth: personal.date_of_birth || undefined,
        phone: personal.phone.trim() || null,
        email: personal.email.trim() || null,
        address: personal.address.trim() || null,
      });
      setStaff(updated);
      setShowPersonal(false);
    } catch (err) {
      setPersonalError(err instanceof ApiError ? err.message : "Could not save changes.");
    }
  };

  // --- assignment edit state ---
  const [assignment, setAssignment] = useState({ department_id: "", position_id: "", hired_date: "" });
  const openAssignment = () => {
    if (!staff) return;
    setAssignment({
      department_id: staff.department_id ?? "",
      position_id: staff.position_id ?? "",
      hired_date: staff.hired_date?.slice(0, 10) ?? "",
    });
    setAssignmentError(null);
    setShowAssignment(true);
  };
  const submitAssignment = async (e: FormEvent) => {
    e.preventDefault();
    setAssignmentError(null);
    try {
      const updated = await setStaffAssignment(id, {
        department_id: assignment.department_id || null,
        position_id: assignment.position_id || null,
        hired_date: assignment.hired_date || null,
      });
      setStaff(updated);
      setShowAssignment(false);
    } catch (err) {
      setAssignmentError(err instanceof ApiError ? err.message : "Could not update assignment.");
    }
  };

  // --- account state ---
  const [accountForm, setAccountForm] = useState({ email: "", password: "", role: "TEACHER" });
  const submitAccount = async (e: FormEvent) => {
    e.preventDefault();
    setAccountError(null);
    try {
      await createStaffAccount(id, {
        email: accountForm.email.trim(),
        password: accountForm.password,
        roles: [accountForm.role],
      });
      setShowAccount(false);
      setAccountSuccess("Account created.");
      loadStaff();
    } catch (err) {
      setAccountError(err instanceof ApiError ? err.message : "Could not create the account.");
    }
  };

  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const openRoles = () => {
    if (!staff?.account) return;
    setSelectedRoles(staff.account.roles);
    setAccountError(null);
    setShowRoles(true);
  };
  const submitRoles = async (e: FormEvent) => {
    e.preventDefault();
    setAccountError(null);
    try {
      await setAccountRoles(id, selectedRoles);
      setShowRoles(false);
      setAccountSuccess("Roles updated.");
      loadStaff();
    } catch (err) {
      setAccountError(err instanceof ApiError ? err.message : "Could not update roles.");
    }
  };

  const changeStatus = async (status: string) => {
    if (!staff || status === staff.employment_status) return;
    setAccountError(null);
    try {
      const updated = await setEmploymentStatus(id, status);
      setStaff(updated);
    } catch (err) {
      setAccountError(err instanceof ApiError ? err.message : "Could not change status.");
      loadStaff(); // re-sync the select with the server state
    }
  };

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
      </div>
    );
  }

  if (loadError || !staff) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
        {loadError ?? "Staff member not found."}{" "}
        <Link href="/staff" className="font-semibold underline">
          Back to directory
        </Link>
      </div>
    );
  }

  const roleOptions = roles.length > 0 ? roles.map((r) => r.name) : [...SYSTEM_ROLES];

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <Link href="/staff" className="text-xs font-medium text-slate-500 hover:text-brand-700">
            ← Staff Directory
          </Link>
          <div className="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                {staff.first_name} {staff.last_name}
              </h1>
              <p className="mt-1 flex items-center gap-3 text-sm text-slate-500">
                <span className="font-mono text-xs">{staff.staff_number}</span>
                <span
                  className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${
                    staff.employment_status === "ACTIVE"
                      ? "bg-emerald-50 text-emerald-700 ring-emerald-600/20"
                      : "bg-amber-50 text-amber-700 ring-amber-600/20"
                  }`}
                >
                  {staff.employment_status.replace("_", " ")}
                </span>
              </p>
            </div>
            {canDeactivate && (
              <div className="flex items-center gap-2">
                <label className="text-xs font-medium text-slate-500">Employment status</label>
                <select
                  value={staff.employment_status}
                  onChange={(e) => changeStatus(e.target.value)}
                  className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-brand-600 focus:outline-none"
                >
                  {EMPLOYMENT_STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {s.replace("_", " ")}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>
        </div>

        {accountSuccess && (
          <p className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs font-medium text-emerald-700">
            {accountSuccess}
          </p>
        )}

        {/* Details grid */}
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
          {/* Personal details */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-bold text-slate-900">Personal details</h2>
              {canUpdate && (
                <button
                  onClick={openPersonal}
                  className="text-xs font-semibold text-brand-700 hover:underline"
                >
                  Edit
                </button>
              )}
            </div>
            <dl className="grid grid-cols-2 gap-4">
              <Field label="First name" value={staff.first_name} />
              <Field label="Last name" value={staff.last_name} />
              <Field label="Other names" value={staff.other_names ?? ""} />
              <Field label="Gender" value={staff.gender ?? ""} />
              <Field label="Date of birth" value={staff.date_of_birth?.slice(0, 10) ?? ""} />
              <Field label="Phone" value={staff.phone ?? ""} />
              <Field label="Email" value={staff.email ?? ""} />
              <Field label="Address" value={staff.address ?? ""} />
            </dl>
          </section>

          {/* Employment */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-bold text-slate-900">Employment &amp; assignment</h2>
              {canUpdate && (
                <button
                  onClick={openAssignment}
                  className="text-xs font-semibold text-brand-700 hover:underline"
                >
                  Edit
                </button>
              )}
            </div>
            <dl className="grid grid-cols-2 gap-4">
              <Field label="Department" value={staff.department_name ?? ""} />
              <Field label="Position" value={staff.position_title ?? ""} />
              <Field label="Category" value={staff.position_category ?? ""} />
              <Field label="Hired date" value={staff.hired_date?.slice(0, 10) ?? ""} />
            </dl>
          </section>
        </div>

        {/* Account */}
        <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-bold text-slate-900">Login account</h2>
            {isAdmin && (
              staff.account ? (
                <button
                  onClick={openRoles}
                  className="text-xs font-semibold text-brand-700 hover:underline"
                >
                  Edit roles
                </button>
              ) : (
                <button
                  onClick={() => {
                    setAccountForm({ email: staff.email ?? "", password: "", role: "TEACHER" });
                    setShowAccount(true);
                  }}
                  className="text-xs font-semibold text-brand-700 hover:underline"
                >
                  Create account
                </button>
              )
            )}
          </div>
          {accountError && (
            <p className="mb-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
              {accountError}
            </p>
          )}
          {staff.account ? (
            <dl className="grid grid-cols-3 gap-4">
              <Field label="Email" value={staff.account.email} />
              <Field label="Status" value={staff.account.status} />
              <Field label="Roles" value={staff.account.roles.join(", ")} />
            </dl>
          ) : (
            <p className="text-sm text-slate-500">
              No login account yet. Creating one lets this staff member sign in to their
              workspace.
            </p>
          )}
        </section>
      </div>

      {/* Personal edit modal */}
      <Modal open={showPersonal} onClose={() => setShowPersonal(false)} title="Edit personal details">
        <form onSubmit={submitPersonal} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="First name *"
              required
              value={personal.first_name}
              onChange={(e) => setPersonal({ ...personal, first_name: e.target.value })}
            />
            <TextField
              label="Last name *"
              required
              value={personal.last_name}
              onChange={(e) => setPersonal({ ...personal, last_name: e.target.value })}
            />
            <TextField
              label="Other names"
              value={personal.other_names}
              onChange={(e) => setPersonal({ ...personal, other_names: e.target.value })}
            />
            <SelectField
              label="Gender"
              value={personal.gender}
              onChange={(e) => setPersonal({ ...personal, gender: e.target.value })}
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
              value={personal.date_of_birth}
              onChange={(e) => setPersonal({ ...personal, date_of_birth: e.target.value })}
            />
            <TextField
              label="Phone"
              value={personal.phone}
              onChange={(e) => setPersonal({ ...personal, phone: e.target.value })}
            />
          </div>
          <TextField
            label="Email"
            type="email"
            value={personal.email}
            onChange={(e) => setPersonal({ ...personal, email: e.target.value })}
          />
          <TextAreaField
            label="Address"
            value={personal.address}
            onChange={(e) => setPersonal({ ...personal, address: e.target.value })}
          />
          <FormActions error={personalError} submitting={false} onCancel={() => setShowPersonal(false)} />
        </form>
      </Modal>

      {/* Assignment edit modal */}
      <Modal open={showAssignment} onClose={() => setShowAssignment(false)} title="Edit assignment">
        <form onSubmit={submitAssignment} className="space-y-4">
          <SelectField
            label="Department"
            value={assignment.department_id}
            onChange={(e) => setAssignment({ ...assignment, department_id: e.target.value })}
          >
            <option value="">— none —</option>
            {departments.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </SelectField>
          <SelectField
            label="Position"
            value={assignment.position_id}
            onChange={(e) => setAssignment({ ...assignment, position_id: e.target.value })}
          >
            <option value="">— none —</option>
            {positions.map((p) => (
              <option key={p.id} value={p.id}>
                {p.title}
              </option>
            ))}
          </SelectField>
          <TextField
            label="Hired date"
            type="date"
            value={assignment.hired_date}
            onChange={(e) => setAssignment({ ...assignment, hired_date: e.target.value })}
          />
          <p className="text-xs text-slate-400">
            Assignment is a full replace: leaving a field as “— none —” clears it.
          </p>
          <FormActions error={assignmentError} submitting={false} onCancel={() => setShowAssignment(false)} />
        </form>
      </Modal>

      {/* Create account modal */}
      <Modal open={showAccount} onClose={() => setShowAccount(false)} title="Create login account">
        <form onSubmit={submitAccount} className="space-y-4">
          <TextField
            label="Email *"
            type="email"
            required
            value={accountForm.email}
            onChange={(e) => setAccountForm({ ...accountForm, email: e.target.value })}
          />
          <TextField
            label="Password *"
            type="password"
            required
            minLength={8}
            value={accountForm.password}
            onChange={(e) => setAccountForm({ ...accountForm, password: e.target.value })}
          />
          <p className="text-xs text-slate-400">
            Minimum 8 characters. Share it securely — it is hashed server-side. The
            staff member also receives an email invitation to sign up with this
            address when Clerk is configured.
          </p>
          <SelectField
            label="Initial role *"
            value={accountForm.role}
            onChange={(e) => setAccountForm({ ...accountForm, role: e.target.value })}
          >
            {roleOptions.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </SelectField>
          <FormActions error={null} submitting={false} label="Create account" onCancel={() => setShowAccount(false)} />
        </form>
      </Modal>

      {/* Edit roles modal */}
      <Modal open={showRoles} onClose={() => setShowRoles(false)} title="Edit account roles">
        <form onSubmit={submitRoles} className="space-y-4">
          <fieldset className="space-y-2">
            {roleOptions.map((r) => (
              <label key={r} className="flex items-center gap-2 text-sm text-slate-700">
                <input
                  type="checkbox"
                  checked={selectedRoles.includes(r)}
                  onChange={(e) =>
                    setSelectedRoles(
                      e.target.checked
                        ? [...selectedRoles, r]
                        : selectedRoles.filter((x) => x !== r),
                    )
                  }
                  className="h-4 w-4 rounded border-slate-300 text-brand-700 focus:ring-brand-600"
                />
                {r}
              </label>
            ))}
          </fieldset>
          <p className="text-xs text-slate-400">
            Roles are replaced wholesale — unchecking everything removes all roles.
          </p>
          <FormActions error={null} submitting={false} label="Save roles" onCancel={() => setShowRoles(false)} />
        </form>
      </Modal>
    </>
  );
}

export default function StaffProfilePage() {
  return (
    <ProtectedRoute requiredPermission="staff.view">
      <AppShell title="Staff Profile">
        <StaffProfile />
      </AppShell>
    </ProtectedRoute>
  );
}
