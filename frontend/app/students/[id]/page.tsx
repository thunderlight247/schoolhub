"use client";

// Student profile (M25): full record with personal edit (PATCH), archive /
// re-activate (student.archive), and guardian relationship management
// (link existing guardians / unlink, student.update). The guardians card
// renders contact channels (phone / email) as the reference design does.

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
import {
  getStudent,
  linkGuardian,
  setStudentStatus,
  unlinkGuardian,
  updateStudent,
} from "@/services/students";
import {
  createEnrollment,
  listStudentEnrollments,
  setEnrollmentStatus,
} from "@/services/enrollments";
import {
  ENROLLMENT_STATUSES,
  Enrollment,
  EnrollmentStatus,
} from "@/types/enrollments";
import { listAcademicYears, listClasses } from "@/services/academics";
import {
  AcademicYear,
  Class,
} from "@/types/academics";
import {
  GUARDIAN_RELATIONSHIPS,

  GuardianLink,
  StudentDetail,
} from "@/types/students";
import { GENDERS } from "@/types/staff";

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</dt>
      <dd className="mt-0.5 text-sm font-medium text-slate-800">{value || "—"}</dd>
    </div>
  );
}

function StudentProfile() {
  const { id } = useParams<{ id: string }>();
  const { hasPermission } = useAuth();
  const canUpdate = hasPermission("student.update");
  const canArchive = hasPermission("student.archive");

  const [student, setStudent] = useState<StudentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  // edit modal
  const [showEdit, setShowEdit] = useState(false);
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
  const [personalError, setPersonalError] = useState<string | null>(null);

  // link guardian modal
  const [showLink, setShowLink] = useState(false);
  const [linkForm, setLinkForm] = useState({ guardian_id: "", relationship: "GUARDIAN" });
  const [linkError, setLinkError] = useState<string | null>(null);
  const [linkSubmitting, setLinkSubmitting] = useState(false);

  // link-existing guardian search
  const [guardianQuery, setGuardianQuery] = useState("");
  const [guardianResults, setGuardianResults] = useState<
    Array<{ id: string; first_name: string; last_name: string; student_count?: number }>
  >([]);
  const [guardianSearchLoading, setGuardianSearchLoading] = useState(false);

  // enrollment history (M27)
  const [enrollments, setEnrollments] = useState<Enrollment[]>([]);
  const [years, setYears] = useState<AcademicYear[]>([]);
  const [classes, setClasses] = useState<Class[]>([]);
  const [showEnroll, setShowEnroll] = useState(false);
  const [enrollForm, setEnrollForm] = useState({
    academic_year_id: "",
    class_id: "",
    enrollment_date: new Date().toISOString().slice(0, 10),
    notes: "",
  });
  const [enrollError, setEnrollError] = useState<string | null>(null);
  const [enrollSubmitting, setEnrollSubmitting] = useState(false);

  const loadEnrollments = useCallback(async () => {
    try {
      const res = await listStudentEnrollments(id);
      setEnrollments(res.enrollments);
    } catch {
      setEnrollments([]);
    }
  }, [id]);

  const loadStudent = useCallback(async () => {
    try {
      setLoadError(null);
      setStudent(await getStudent(id));
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : "Could not load the student.");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadStudent();
  }, [loadStudent]);

  const openEdit = () => {
    if (!student) return;
    setPersonal({
      first_name: student.first_name,
      last_name: student.last_name,
      other_names: student.other_names ?? "",
      gender: student.gender ?? "",
      date_of_birth: student.date_of_birth?.slice(0, 10) ?? "",
      phone: student.phone ?? "",
      email: student.email ?? "",
      address: student.address ?? "",
    });
    setPersonalError(null);
    setShowEdit(true);
  };

  const submitEdit = async (e: FormEvent) => {
    e.preventDefault();
    setPersonalError(null);
    try {
      const updated = await updateStudent(id, {
        first_name: personal.first_name.trim(),
        last_name: personal.last_name.trim(),
        other_names: personal.other_names.trim() || null,
        gender: personal.gender || undefined,
        date_of_birth: personal.date_of_birth || undefined,
        phone: personal.phone.trim() || null,
        email: personal.email.trim() || null,
        address: personal.address.trim() || null,
      });
      setStudent(updated);
      setShowEdit(false);
    } catch (err) {
      setPersonalError(err instanceof ApiError ? err.message : "Could not save changes.");
    }
  };

  const toggleArchive = async () => {
    if (!student) return;
    const next = student.status === "ACTIVE" ? "ARCHIVED" : "ACTIVE";
    setActionError(null);
    setActionSuccess(null);
    try {
      const updated = await setStudentStatus(id, next);
      setStudent(updated);
      setActionSuccess(next === "ARCHIVED" ? "Student archived." : "Student re-activated.");
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Could not change status.");
    }
  };

  useEffect(() => {
    loadEnrollments();
    listAcademicYears()
      .then(setYears)
      .catch(() => {});
    listClasses()
      .then(setClasses)
      .catch(() => {});
  }, [loadEnrollments]);

  const changeEnrollmentStatus = async (enrollment: Enrollment, next: EnrollmentStatus) => {
    if (next === enrollment.status) return;
    setActionError(null);
    setActionSuccess(null);
    try {
      const updated = await setEnrollmentStatus(enrollment.id, next);
      setEnrollments((es) => es.map((e) => (e.id === updated.id ? updated : e)));
      setActionSuccess(`Enrollment marked ${updated.status.toLowerCase()}.`);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Could not update the enrollment.");
      loadEnrollments(); // re-sync with server state
    }
  };

  const submitEnroll = async (e: FormEvent) => {
    e.preventDefault();
    if (!enrollForm.class_id) {
      setEnrollError("Select a class.");
      return;
    }
    setEnrollSubmitting(true);
    setEnrollError(null);
    try {
      await createEnrollment({
        student_id: id,
        academic_year_id: enrollForm.academic_year_id,
        class_id: enrollForm.class_id,
        enrollment_date: enrollForm.enrollment_date || undefined,
        notes: enrollForm.notes.trim() || undefined,
      });
      setShowEnroll(false);
      setActionSuccess("Student enrolled.");
      loadEnrollments();
    } catch (err) {
      setEnrollError(err instanceof ApiError ? err.message : "Could not enroll the student.");
    } finally {
      setEnrollSubmitting(false);
    }
  };

  // --- guardian linking ---
  useEffect(() => {
    if (!showLink) return;
    const q = guardianQuery.trim();
    setGuardianSearchLoading(true);
    const timer = setTimeout(async () => {
      try {
        const { listGuardians } = await import("@/services/students");
        const res = await listGuardians({ search: q || undefined, page: 1, page_size: 5 });
        setGuardianResults(res.guardians);
      } catch {
        setGuardianResults([]);
      } finally {
        setGuardianSearchLoading(false);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [guardianQuery, showLink]);

  const openLink = () => {
    setLinkForm({ guardian_id: "", relationship: "GUARDIAN" });
    setGuardianQuery("");
    setGuardianResults([]);
    setLinkError(null);
    setShowLink(true);
  };

  const submitLink = async (e: FormEvent) => {
    e.preventDefault();
    if (!linkForm.guardian_id) {
      setLinkError("Select a guardian to link.");
      return;
    }
    setLinkSubmitting(true);
    setLinkError(null);
    try {
      const guardians = await linkGuardian(id, linkForm.guardian_id, linkForm.relationship);
      setStudent((s) => (s ? { ...s, guardians } : s));
      setShowLink(false);
      setActionSuccess("Guardian linked.");
    } catch (err) {
      setLinkError(err instanceof ApiError ? err.message : "Could not link the guardian.");
    } finally {
      setLinkSubmitting(false);
    }
  };

  const unlink = async (guardianId: string, name: string) => {
    setActionError(null);
    setActionSuccess(null);
    try {
      const guardians = await unlinkGuardian(id, guardianId);
      setStudent((s) => (s ? { ...s, guardians } : s));
      setActionSuccess(`Unlinked ${name}.`);
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : "Could not unlink the guardian.");
    }
  };

  const [confirmUnlink, setConfirmUnlink] = useState<GuardianLink | null>(null);

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
      </div>
    );
  }

  if (loadError || !student) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
        {loadError ?? "Student not found."}{" "}
        <Link href="/students" className="font-semibold underline">
          Back to directory
        </Link>
      </div>
    );
  }

  const isArchived = student.status === "ARCHIVED";

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <div>
          <Link href="/students" className="text-xs font-medium text-slate-500 hover:text-brand-700">
            ← Students
          </Link>
          <div className="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-4">
              <Avatar firstName={student.first_name} lastName={student.last_name} size="lg" />
              <div>
                <h1 className="text-2xl font-bold tracking-tight text-slate-900">
                  {student.first_name} {student.last_name}
                </h1>
                <p className="mt-1 flex items-center gap-3 text-sm text-slate-500">
                  <span className="font-mono text-xs">{student.student_number}</span>
                  {isArchived ? (
                    <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-inset ring-slate-500/20">
                      Archived
                    </span>
                  ) : (
                    <span className="inline-flex rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 ring-1 ring-inset ring-emerald-600/20">
                      Active
                    </span>
                  )}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              {canArchive && (
                <button
                  onClick={toggleArchive}
                  className={`rounded-md px-3 py-1.5 text-xs font-semibold shadow-sm ${
                    isArchived
                      ? "bg-brand-700 text-white hover:bg-brand-600"
                      : "border border-red-200 bg-white text-red-700 hover:bg-red-50"
                  }`}
                >
                  {isArchived ? "Re-activate" : "Archive student"}
                </button>
              )}
              {canUpdate && (
                <button
                  onClick={openEdit}
                  className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 shadow-sm hover:bg-slate-50"
                >
                  Edit details
                </button>
              )}
            </div>
          </div>
        </div>

        {(actionError || actionSuccess) && (
          <p
            role="status"
            className={`rounded-lg border px-3 py-2 text-xs font-medium ${
              actionError
                ? "border-red-200 bg-red-50 text-red-700"
                : "border-emerald-200 bg-emerald-50 text-emerald-700"
            }`}
          >
            {actionError ?? actionSuccess}
          </p>
        )}

        {/* Details */}
        <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="mb-4 text-sm font-bold text-slate-900">Personal details</h2>
            <dl className="grid grid-cols-2 gap-4">
              <Field label="First name" value={student.first_name} />
              <Field label="Last name" value={student.last_name} />
              <Field label="Other names" value={student.other_names ?? ""} />
              <Field label="Gender" value={student.gender ?? ""} />
              <Field label="Date of birth" value={student.date_of_birth?.slice(0, 10) ?? ""} />
              <Field label="Admission date" value={student.admission_date?.slice(0, 10) ?? ""} />
              <Field label="Phone" value={student.phone ?? ""} />
              <Field label="Email" value={student.email ?? ""} />
            </dl>
            <div className="mt-4 border-t border-slate-100 pt-4">
              <Field label="Address" value={student.address ?? ""} />
            </div>
          </section>

          {/* Guardians */}
          <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-4 flex items-center justify-between">
              <h2 className="text-sm font-bold text-slate-900">
                Guardians{" "}
                <span className="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600">
                  {student.guardians.length}
                </span>
              </h2>
              {canUpdate && (
                <button
                  onClick={openLink}
                  className="text-xs font-semibold text-brand-700 hover:underline"
                >
                  + Link guardian
                </button>
              )}
            </div>

            {student.guardians.length === 0 ? (
              <p className="rounded-lg border border-dashed border-slate-200 p-4 text-sm text-slate-500">
                No guardians linked yet. Link one to give a parent portal access to this
                student&apos;s records.
              </p>
            ) : (
              <ul className="space-y-3">
                {student.guardians.map((g) => (
                  <li
                    key={g.guardian_id}
                    className="flex items-start justify-between gap-3 rounded-lg border border-slate-100 p-3"
                  >
                    <div className="flex items-start gap-3">
                      <Avatar firstName={g.first_name} lastName={g.last_name} />
                      <div className="min-w-0">
                        <Link
                          href={`/guardians/${g.guardian_id}`}
                          className="text-sm font-semibold text-slate-800 hover:text-brand-700 hover:underline"
                        >
                          {g.first_name} {g.last_name}
                        </Link>
                        <p className="text-xs text-slate-500">
                          {g.relationship}
                          {g.occupation ? ` · ${g.occupation}` : ""}
                        </p>
                        <p className="mt-1 flex flex-wrap items-center gap-3 text-xs text-slate-500">
                          {g.phone && (
                            <span className="inline-flex items-center gap-1">
                              {/* eslint-disable-next-line @next/next/no-img-element */}
                              <img src="/icons/phone.png" alt="" width={12} height={12} />
                              {g.phone}
                            </span>
                          )}
                          {g.email && (
                            <span className="inline-flex items-center gap-1">
                              {/* eslint-disable-next-line @next/next/no-img-element */}
                              <img src="/icons/mail.png" alt="" width={12} height={12} />
                              {g.email}
                            </span>
                          )}
                        </p>
                      </div>
                    </div>
                    {canUpdate && (
                      <button
                        onClick={() => setConfirmUnlink(g)}
                        className="rounded-md border border-red-200 bg-white px-2 py-1 text-xs font-medium text-red-600 hover:bg-red-50"
                      >
                        Unlink
                      </button>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </div>

        {/* Enrollment history (M27) */}
        <section className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-bold text-slate-900">
              Enrollment history{" "}
              <span className="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-600">
                {enrollments.length}
              </span>
            </h2>
            {hasPermission("class.create") && !isArchived && (
              <button
                onClick={() => {
                  setEnrollForm((f) => ({
                    ...f,
                    academic_year_id: years.find((y) => y.is_current)?.id ?? years[0]?.id ?? "",
                    class_id: "",
                  }));
                  setEnrollError(null);
                  setShowEnroll(true);
                }}
                className="text-xs font-semibold text-brand-700 hover:underline"
              >
                + Enroll in class
              </button>
            )}
          </div>
          {enrollments.length === 0 ? (
            <p className="rounded-lg border border-dashed border-slate-200 p-4 text-sm text-slate-500">
              No enrollment history yet. Enroll this student into a class for the current
              academic year.
            </p>
          ) : (
            <ul className="space-y-3">
              {enrollments.map((e) => (
                <li
                  key={e.id}
                  className="flex flex-col gap-2 rounded-lg border border-slate-100 p-3 sm:flex-row sm:items-center sm:justify-between"
                >
                  <div>
                    <p className="text-sm font-semibold text-slate-800">
                      {e.class_name ?? "Class"}
                      <span className="ml-2 text-xs font-normal text-slate-500">
                        {e.academic_year_name ?? ""}
                      </span>
                    </p>
                    <p className="mt-0.5 text-xs text-slate-500">
                      Enrolled {e.enrollment_date}
                      {e.exit_date ? ` · exited ${e.exit_date}` : ""}
                      {e.notes ? ` · ${e.notes}` : ""}
                    </p>
                  </div>
                  {hasPermission("class.update") ? (
                    <select
                      value={e.status}
                      onChange={(ev) =>
                        changeEnrollmentStatus(e, ev.target.value as EnrollmentStatus)
                      }
                      aria-label={`Change enrollment status for ${e.class_name ?? "class"}`}
                      className="rounded-md border border-slate-200 bg-white px-2 py-1 text-xs shadow-sm focus:outline-none focus:ring-brand-600"
                    >
                      {ENROLLMENT_STATUSES.map((s) => (
                        <option key={s} value={s}>
                          {s}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <span className="inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 ring-1 ring-inset ring-slate-500/20">
                      {e.status}
                    </span>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>

      {/* Enroll-in-class modal */}
      <Modal open={showEnroll} onClose={() => setShowEnroll(false)} title="Enroll in a class">
        <form onSubmit={submitEnroll} className="space-y-4">
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
              .filter(
                (c) =>
                  !enrollForm.academic_year_id ||
                  c.academic_year_id === enrollForm.academic_year_id,
              )
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

      {/* Edit personal details modal */}
      <Modal open={showEdit} onClose={() => setShowEdit(false)} title="Edit student details">
        <form onSubmit={submitEdit} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
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
          <FormActions error={personalError} submitting={false} onCancel={() => setShowEdit(false)} />
        </form>
      </Modal>

      {/* Link guardian modal */}
      <Modal open={showLink} onClose={() => setShowLink(false)} title="Link an existing guardian">
        <form onSubmit={submitLink} className="space-y-4">
          <div>
            <label className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
              Find guardian
            </label>
            <SearchGuardianInput value={guardianQuery} onChange={setGuardianQuery} />
            {guardianSearchLoading && (
              <p className="mt-2 text-xs text-slate-400">Searching…</p>
            )}
            <div className="mt-2 space-y-1">
              {guardianResults.map((g) => (
                <button
                  type="button"
                  key={g.id}
                  onClick={() => setLinkForm((f) => ({ ...f, guardian_id: g.id }))}
                  className={`flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left text-sm ${
                    linkForm.guardian_id === g.id
                      ? "border-brand-600 bg-brand-50"
                      : "border-slate-200 hover:bg-slate-50"
                  }`}
                >
                  <span className="font-medium text-slate-800">
                    {g.first_name} {g.last_name}
                  </span>
                  <span className="text-xs text-slate-400">
                    {g.student_count} linked student{g.student_count === 1 ? "" : "s"}
                  </span>
                </button>
              ))}
              {!guardianSearchLoading && guardianResults.length === 0 && (
                <p className="rounded-lg border border-dashed border-slate-200 p-3 text-xs text-slate-500">
                  No guardians found. Create one first from the{" "}
                  <Link href="/guardians" className="font-semibold text-brand-700 underline">
                    Guardians
                  </Link>{" "}
                  page, then link them here.
                </p>
              )}
            </div>
          </div>
          <SelectField
            label="Relationship *"
            value={linkForm.relationship}
            onChange={(e) => setLinkForm({ ...linkForm, relationship: e.target.value })}
          >
            {GUARDIAN_RELATIONSHIPS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </SelectField>
          <FormActions
            error={linkError}
            submitting={linkSubmitting}
            label="Link guardian"
            onCancel={() => setShowLink(false)}
          />
        </form>
      </Modal>

      {/* Unlink confirmation modal */}
      <Modal
        open={confirmUnlink !== null}
        onClose={() => setConfirmUnlink(null)}
        title="Unlink guardian"
      >
        <p className="text-sm text-slate-600">
          Unlink{" "}
          <span className="font-semibold text-slate-800">
            {confirmUnlink?.first_name} {confirmUnlink?.last_name}
          </span>{" "}
          from {student.first_name} {student.last_name}? The guardian record is kept — only the
          relationship is removed. Their portal access to this student ends immediately.
        </p>
        <div className="mt-5 flex justify-end gap-2">
          <button
            onClick={() => setConfirmUnlink(null)}
            className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50"
          >
            Cancel
          </button>
          <button
            onClick={async () => {
              if (confirmUnlink) {
                await unlink(
                  confirmUnlink.guardian_id,
                  `${confirmUnlink.first_name} ${confirmUnlink.last_name}`,
                );
              }
              setConfirmUnlink(null);
            }}
            className="rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-red-500"
          >
            Unlink
          </button>
        </div>
      </Modal>
    </>
  );
}

// Local wrapper keeps the modal's search box compact.
function SearchGuardianInput({
  value,
  onChange,
}: {
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <input
      type="search"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="Search by name…"
      className="mt-1 w-full rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 outline-none focus:ring-brand-600"
    />
  );
}

export default function StudentProfilePage() {
  return (
    <ProtectedRoute requiredPermission="student.view">
      <AppShell title="Student Profile">
        <StudentProfile />
      </AppShell>
    </ProtectedRoute>
  );
}
