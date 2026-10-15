"use client";

// Academic Structure (M26): the calendar and curriculum backbone — academic
// years & terms with their unique "current" flags, programmes, subjects,
// classes, and per-class curriculum (class_subjects). Gated on class.view;
// creation on class.create; edits/assignments on class.update.
//
// The "set current" actions move the single current year/term — the backend
// enforces uniqueness, and this UI reflects the change by reloading.

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { Tabs } from "@/components/ui/tabs";
import { DataTable, type Column } from "@/components/ui/table";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  assignSubjectToClass,
  createAcademicYear,
  createClass,
  createProgramme,
  createSubject,
  createTerm,
  listAcademicYears,
  listClassSubjects,
  listClasses,
  listProgrammes,
  listSubjects,
  listTerms,
  removeSubjectFromClass,
  setCurrentAcademicYear,
  setCurrentTerm,
} from "@/services/academics";
import {
  AcademicYear,
  Class,
  ClassSubject,
  Programme,
  SUBJECT_TYPES,
  Subject,
  Term,
} from "@/types/academics";

type Tab = "years" | "terms" | "programmes" | "subjects" | "classes" | "curriculum";

const TABS: { key: Tab; label: string }[] = [
  { key: "years", label: "Academic Years" },
  { key: "terms", label: "Terms" },
  { key: "programmes", label: "Programmes" },
  { key: "subjects", label: "Subjects" },
  { key: "classes", label: "Classes" },
  { key: "curriculum", label: "Class Curriculum" },
];

function CurrentBadge() {
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700 ring-1 ring-inset ring-brand-100">
      <span className="h-1.5 w-1.5 rounded-full bg-brand-600" />
      Current
    </span>
  );
}

function AcademicStructure() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("class.create");
  const canUpdate = hasPermission("class.update");

  const [tab, setTab] = useState<Tab>("years");
  const [years, setYears] = useState<AcademicYear[]>([]);
  const [terms, setTerms] = useState<Term[]>([]);
  const [programmes, setProgrammes] = useState<Programme[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [classes, setClasses] = useState<Class[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [y, t, p, s, c] = await Promise.all([
        listAcademicYears(),
        listTerms(),
        listProgrammes(),
        listSubjects(),
        listClasses(),
      ]);
      setYears(y);
      setTerms(t);
      setProgrammes(p);
      setSubjects(s);
      setClasses(c);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load academic structure.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAll();
  }, [loadAll]);

  const flash = (msg: string) => {
    setNotice(msg);
    setError(null);
  };

  // --- modal plumbing (one open modal at a time) ---
  const [modal, setModal] = useState<null | { kind: string; editId?: string }>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const close = () => {
    setModal(null);
    setFormError(null);
  };

  // --- year form ---
  const [yearForm, setYearForm] = useState({ name: "", start_date: "", end_date: "", is_current: false });
  const openYearModal = (y?: AcademicYear) => {
    setYearForm(
      y
        ? { name: y.name, start_date: y.start_date.slice(0, 10), end_date: y.end_date.slice(0, 10), is_current: y.is_current }
        : { name: "", start_date: "", end_date: "", is_current: false },
    );
    setModal({ kind: "year", editId: y?.id });
  };
  const submitYear = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      if (modal?.editId) {
        await import("@/services/academics").then((m) =>
          m.updateAcademicYear(modal.editId!, {
            name: yearForm.name.trim(),
            start_date: yearForm.start_date,
            end_date: yearForm.end_date,
          }),
        );
      } else {
        await createAcademicYear(
          yearForm.name.trim(),
          yearForm.start_date,
          yearForm.end_date,
          yearForm.is_current,
        );
      }
      close();
      await loadAll();
      flash("Academic year saved.");
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not save the academic year.");
    } finally {
      setSubmitting(false);
    }
  };

  // --- term form ---
  const [termForm, setTermForm] = useState({
    academic_year_id: "",
    name: "",
    start_date: "",
    end_date: "",
    is_current: false,
  });
  const openTermModal = (t?: Term) => {
    setTermForm(
      t
        ? {
            academic_year_id: t.academic_year_id,
            name: t.name,
            start_date: t.start_date.slice(0, 10),
            end_date: t.end_date.slice(0, 10),
            is_current: t.is_current,
          }
        : {
            academic_year_id: years.find((y) => y.is_current)?.id ?? years[0]?.id ?? "",
            name: "",
            start_date: "",
            end_date: "",
            is_current: false,
          },
    );
    setModal({ kind: "term", editId: t?.id });
  };
  const submitTerm = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      if (modal?.editId) {
        await import("@/services/academics").then((m) =>
          m.updateTerm(modal.editId!, {
            name: termForm.name.trim(),
            start_date: termForm.start_date,
            end_date: termForm.end_date,
          }),
        );
      } else {
        await createTerm(
          termForm.academic_year_id,
          termForm.name.trim(),
          termForm.start_date,
          termForm.end_date,
          termForm.is_current,
        );
      }
      close();
      await loadAll();
      flash("Term saved.");
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not save the term.");
    } finally {
      setSubmitting(false);
    }
  };

  // --- programme form ---
  const [progForm, setProgForm] = useState({ code: "", name: "", description: "" });
  const submitProgramme = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      await createProgramme(progForm.code.trim().toUpperCase(), progForm.name.trim(), progForm.description.trim() || undefined);
      close();
      await loadAll();
      flash("Programme created.");
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not create the programme.");
    } finally {
      setSubmitting(false);
    }
  };

  // --- subject form ---
  const [subjForm, setSubjForm] = useState({ code: "", name: "", type: "CORE", description: "" });
  const submitSubject = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      await createSubject(subjForm.code.trim().toUpperCase(), subjForm.name.trim(), subjForm.type, subjForm.description.trim() || undefined);
      close();
      await loadAll();
      flash("Subject created.");
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not create the subject.");
    } finally {
      setSubmitting(false);
    }
  };

  // --- class form ---
  const [classForm, setClassForm] = useState({
    academic_year_id: "",
    programme_id: "",
    name: "",
    level: "",
    capacity: "",
  });
  const openClassModal = () => {
    setClassForm((f) => ({
      ...f,
      academic_year_id: years.find((y) => y.is_current)?.id ?? years[0]?.id ?? "",
      programme_id: programmes[0]?.id ?? "",
      name: "",
      level: "",
      capacity: "",
    }));
    setModal({ kind: "class" });
  };
  const submitClass = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      await createClass({
        academic_year_id: classForm.academic_year_id,
        programme_id: classForm.programme_id || undefined,
        name: classForm.name.trim(),
        level: classForm.level.trim() || undefined,
        capacity: classForm.capacity ? Number(classForm.capacity) : undefined,
      });
      close();
      await loadAll();
      flash("Class created.");
    } catch (err) {
      setFormError(err instanceof ApiError ? err.message : "Could not create the class.");
    } finally {
      setSubmitting(false);
    }
  };

  // --- set current ---
  const setCurrent = async (kind: "year" | "term", id: string) => {
    setError(null);
    try {
      if (kind === "year") {
        await setCurrentAcademicYear(id);
        flash("Current academic year updated.");
      } else {
        await setCurrentTerm(id);
        flash("Current term updated.");
      }
      await loadAll();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not set current.");
    }
  };

  // --- curriculum tab state ---
  const [curriculumClassId, setCurriculumClassId] = useState("");
  const [curriculum, setCurriculum] = useState<ClassSubject[]>([]);
  const [curriculumLoading, setCurriculumLoading] = useState(false);
  const [assignSubjectId, setAssignSubjectId] = useState("");
  const [curriculumError, setCurriculumError] = useState<string | null>(null);

  useEffect(() => {
    if (tab !== "curriculum") return;
    const classId = curriculumClassId || classes[0]?.id || "";
    if (!classId) {
      setCurriculum([]);
      return;
    }
    setCurriculumLoading(true);
    listClassSubjects(classId)
      .then(setCurriculum)
      .catch(() => setCurriculum([]))
      .finally(() => setCurriculumLoading(false));
  }, [tab, curriculumClassId, classes]);

  const activeCurriculumClassId = curriculumClassId || classes[0]?.id || "";

  const assign = async () => {
    if (!assignSubjectId) return;
    setCurriculumError(null);
    try {
      await assignSubjectToClass(activeCurriculumClassId, assignSubjectId);
      setAssignSubjectId("");
      setCurriculum(await listClassSubjects(activeCurriculumClassId));
      flash("Subject assigned to class.");
    } catch (err) {
      setCurriculumError(err instanceof ApiError ? err.message : "Could not assign the subject.");
    }
  };

  const unassign = async (subjectId: string) => {
    setCurriculumError(null);
    try {
      await removeSubjectFromClass(activeCurriculumClassId, subjectId);
      setCurriculum(await listClassSubjects(activeCurriculumClassId));
      flash("Subject removed from class.");
    } catch (err) {
      setCurriculumError(err instanceof ApiError ? err.message : "Could not remove the subject.");
    }
  };

  const yearColumns: Column<AcademicYear>[] = [
    { key: "name", header: "Year", className: "font-medium text-slate-800" },
    { key: "start_date", header: "Starts", render: (y) => y.start_date.slice(0, 10), hideBelow: "md" },
    { key: "end_date", header: "Ends", render: (y) => y.end_date.slice(0, 10), hideBelow: "md" },
    {
      key: "is_current",
      header: "Status",
      render: (y) => (y.is_current ? <CurrentBadge /> : <span className="text-xs text-slate-400">—</span>),
    },
    ...(canUpdate
      ? [
          {
            key: "actions",
            header: "Actions",
            render: (y: AcademicYear) => (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => openYearModal(y)}
                  title="Edit"
                  aria-label={`Edit ${y.name}`}
                  className="flex h-7 w-7 items-center justify-center rounded-full bg-sky-100 hover:bg-sky-200"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src="/icons/update.png" alt="" width={14} height={14} />
                </button>
                {!y.is_current && (
                  <button
                    onClick={() => setCurrent("year", y.id)}
                    title="Set as current year"
                    className="rounded-full bg-amber-100 px-2 py-1 text-xs font-semibold text-amber-800 hover:bg-amber-200"
                  >
                    Set current
                  </button>
                )}
              </div>
            ),
          } as Column<AcademicYear>,
        ]
      : []),
  ];

  const termColumns: Column<Term>[] = [
    { key: "name", header: "Term", className: "font-medium text-slate-800" },
    { key: "academic_year_name", header: "Year", hideBelow: "md", render: (t) => t.academic_year_name || "—" },
    { key: "start_date", header: "Starts", render: (t) => t.start_date.slice(0, 10), hideBelow: "lg" },
    { key: "end_date", header: "Ends", render: (t) => t.end_date.slice(0, 10), hideBelow: "lg" },
    {
      key: "is_current",
      header: "Status",
      render: (t) => (t.is_current ? <CurrentBadge /> : <span className="text-xs text-slate-400">—</span>),
    },
    ...(canUpdate
      ? [
          {
            key: "actions",
            header: "Actions",
            render: (t: Term) => (
              <div className="flex items-center gap-1">
                <button
                  onClick={() => openTermModal(t)}
                  title="Edit"
                  aria-label={`Edit ${t.name}`}
                  className="flex h-7 w-7 items-center justify-center rounded-full bg-sky-100 hover:bg-sky-200"
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src="/icons/update.png" alt="" width={14} height={14} />
                </button>
                {!t.is_current && (
                  <button
                    onClick={() => setCurrent("term", t.id)}
                    title="Set as current term"
                    className="rounded-full bg-amber-100 px-2 py-1 text-xs font-semibold text-amber-800 hover:bg-amber-200"
                  >
                    Set current
                  </button>
                )}
              </div>
            ),
          } as Column<Term>,
        ]
      : []),
  ];

  const programmeColumns: Column<Programme>[] = [
    { key: "code", header: "Code", className: "font-mono text-xs" },
    { key: "name", header: "Programme", className: "font-medium text-slate-800" },
    { key: "description", header: "Description", render: (p) => p.description ?? "—", hideBelow: "md" },
  ];

  const subjectColumns: Column<Subject>[] = [
    { key: "code", header: "Code", className: "font-mono text-xs" },
    { key: "name", header: "Subject", className: "font-medium text-slate-800" },
    { key: "type", header: "Type", render: (s) => s.type, hideBelow: "md" },
    {
      key: "department_name",
      header: "Department",
      hideBelow: "lg",
      render: (s) => s.department_name ?? "—",
    },
  ];

  const classColumns: Column<Class>[] = [
    { key: "name", header: "Class", className: "font-medium text-slate-800" },
    { key: "level", header: "Level", hideBelow: "md", render: (c) => c.level ?? "—" },
    {
      key: "programme_name",
      header: "Programme",
      hideBelow: "md",
      render: (c) => c.programme_name ?? "—",
    },
    { key: "academic_year_name", header: "Year", hideBelow: "lg", render: (c) => c.academic_year_name ?? "—" },
    { key: "capacity", header: "Capacity", hideBelow: "xl", render: (c) => c.capacity ?? "—" },
  ];

  const curriculumColumns: Column<ClassSubject>[] = [
    { key: "code", header: "Code", className: "font-mono text-xs" },
    { key: "name", header: "Subject", className: "font-medium text-slate-800" },
    { key: "type", header: "Type", hideBelow: "md" },
    {
      key: "assigned_at",
      header: "Assigned",
      hideBelow: "lg",
      render: (s) => s.assigned_at.slice(0, 10),
    },
    ...(canUpdate
      ? [
          {
            key: "actions",
            header: "Actions",
            render: (s: ClassSubject) => (
              <button
                onClick={() => unassign(s.id)}
                title="Remove from class curriculum"
                aria-label={`Remove ${s.name} from class curriculum`}
                className="flex h-7 w-7 items-center justify-center rounded-full bg-red-50 text-red-600 hover:bg-red-100"
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src="/icons/delete.png" alt="" width={13} height={13} />
              </button>
            ),
          } as Column<ClassSubject>,
        ]
      : []),
  ];

  const addIf = canCreate && (
    <button
      onClick={() => {
        if (tab === "years") openYearModal();
        else if (tab === "terms") openTermModal();
        else if (tab === "programmes") {
          setProgForm({ code: "", name: "", description: "" });
          setModal({ kind: "programme" });
        } else if (tab === "subjects") {
          setSubjForm({ code: "", name: "", type: "CORE", description: "" });
          setModal({ kind: "subject" });
        } else if (tab === "classes") openClassModal();
      }}
      className="rounded-md bg-brand-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600"
    >
      + Add{" "}
      {
        {
          years: "academic year",
          terms: "term",
          programmes: "programme",
          subjects: "subject",
          classes: "class",
          curriculum: "",
        }[tab]
      }
    </button>
  );

  return (
    <>
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">Academic Structure</h1>
          <p className="mt-1 text-sm text-slate-500">
            The calendar and curriculum backbone: years, terms, programmes, subjects, classes,
            and per-class curriculum.
          </p>
        </div>

        {/* Tabs */}
        <Tabs
          label="Academic structure sections"
          active={tab}
          onChange={setTab}
          variant="pills"
          items={TABS}
        />

        {(error || notice) && (
          <p
            role="status"
            className={`rounded-lg border px-3 py-2 text-xs font-medium ${
              error ? "border-red-200 bg-red-50 text-red-700" : "border-emerald-200 bg-emerald-50 text-emerald-700"
            }`}
          >
            {error ?? notice}
          </p>
        )}

        {tab === "curriculum" ? (
          <>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-2">
                <label className="text-xs font-medium text-slate-500">Class</label>
                <select
                  value={activeCurriculumClassId}
                  onChange={(e) => setCurriculumClassId(e.target.value)}
                  className="rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 focus:outline-none focus:ring-brand-600"
                >
                  {classes.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                      {c.academic_year_name ? ` (${c.academic_year_name})` : ""}
                    </option>
                  ))}
                </select>
              </div>
              {canUpdate && classes.length > 0 && (
                <div className="flex items-center gap-2">
                  <select
                    value={assignSubjectId}
                    onChange={(e) => setAssignSubjectId(e.target.value)}
                    aria-label="Subject to assign"
                    className="rounded-full bg-white px-3 py-2 text-sm shadow-sm ring-[1.5px] ring-slate-300 focus:outline-none focus:ring-brand-600"
                  >
                    <option value="">Assign a subject…</option>
                    {subjects
                      .filter((s) => !curriculum.some((cs) => cs.id === s.id))
                      .map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.code} — {s.name}
                        </option>
                      ))}
                  </select>
                  <button
                    onClick={assign}
                    disabled={!assignSubjectId}
                    className="rounded-md bg-brand-700 px-3 py-2 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 disabled:opacity-40"
                  >
                    Assign
                  </button>
                </div>
              )}
            </div>

            {curriculumError && (
              <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
                {curriculumError}
              </p>
            )}

            {curriculumLoading ? (
              <div className="flex h-40 items-center justify-center rounded-xl border border-slate-200 bg-white">
                <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
              </div>
            ) : (
              <DataTable
                columns={curriculumColumns}
                rows={curriculum}
                getRowKey={(s) => s.id}
                emptyMessage="No subjects assigned to this class yet."
              />
            )}
          </>
        ) : (
          <>
            <div className="flex justify-end">{addIf}</div>
            {loading ? (
              <div className="flex h-56 items-center justify-center rounded-xl border border-slate-200 bg-white">
                <span className="h-8 w-8 animate-spin rounded-full border-2 border-brand-200 border-t-brand-700" />
              </div>
            ) : tab === "years" ? (
              <DataTable columns={yearColumns} rows={years} getRowKey={(y) => y.id} />
            ) : tab === "terms" ? (
              <DataTable columns={termColumns} rows={terms} getRowKey={(t) => t.id} />
            ) : tab === "programmes" ? (
              <DataTable columns={programmeColumns} rows={programmes} getRowKey={(p) => p.id} />
            ) : tab === "subjects" ? (
              <DataTable columns={subjectColumns} rows={subjects} getRowKey={(s) => s.id} />
            ) : (
              <DataTable columns={classColumns} rows={classes} getRowKey={(c) => c.id} />
            )}
          </>
        )}

        {/* The unique current flag means only one year/term can be current. */}
        {(tab === "years" || tab === "terms") && canUpdate && (
          <p className="text-xs text-slate-400">
            Only one {tab === "years" ? "academic year" : "term"} can be current — setting another
            one moves the flag.
          </p>
        )}
      </div>

      {/* Year modal (create + edit) */}
      <Modal
        open={modal?.kind === "year"}
        onClose={close}
        title={modal?.editId ? "Edit academic year" : "Add academic year"}
      >
        <form onSubmit={submitYear} className="space-y-4">
          <TextField
            id="year-name"
            label="Name *"
            required
            value={yearForm.name}
            onChange={(e) => setYearForm({ ...yearForm, name: e.target.value })}
            placeholder="2027/2028"
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Start date *"
              type="date"
              required
              value={yearForm.start_date}
              onChange={(e) => setYearForm({ ...yearForm, start_date: e.target.value })}
            />
            <TextField
              label="End date *"
              type="date"
              required
              value={yearForm.end_date}
              onChange={(e) => setYearForm({ ...yearForm, end_date: e.target.value })}
            />
          </div>
          {!modal?.editId && (
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={yearForm.is_current}
                onChange={(e) => setYearForm({ ...yearForm, is_current: e.target.checked })}
                className="h-4 w-4 rounded border-slate-300 text-brand-700 focus:ring-brand-600"
              />
              Make this the current academic year
            </label>
          )}
          <FormActions error={formError} submitting={submitting} label="Save year" onCancel={close} />
        </form>
      </Modal>

      {/* Term modal (create + edit) */}
      <Modal
        open={modal?.kind === "term"}
        onClose={close}
        title={modal?.editId ? "Edit term" : "Add term"}
      >
        <form onSubmit={submitTerm} className="space-y-4">
          <SelectField
            label="Academic year *"
            required
            value={termForm.academic_year_id}
            onChange={(e) => setTermForm({ ...termForm, academic_year_id: e.target.value })}
          >
            {years.map((y) => (
              <option key={y.id} value={y.id}>
                {y.name}
              </option>
            ))}
          </SelectField>
          <TextField
            id="term-name"
            label="Name *"
            required
            value={termForm.name}
            onChange={(e) => setTermForm({ ...termForm, name: e.target.value })}
            placeholder="Term 2"
          />
          <div className="grid grid-cols-2 gap-3">
            <TextField
              label="Start date *"
              type="date"
              required
              value={termForm.start_date}
              onChange={(e) => setTermForm({ ...termForm, start_date: e.target.value })}
            />
            <TextField
              label="End date *"
              type="date"
              required
              value={termForm.end_date}
              onChange={(e) => setTermForm({ ...termForm, end_date: e.target.value })}
            />
          </div>
          {!modal?.editId && (
            <label className="flex items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={termForm.is_current}
                onChange={(e) => setTermForm({ ...termForm, is_current: e.target.checked })}
                className="h-4 w-4 rounded border-slate-300 text-brand-700 focus:ring-brand-600"
              />
              Make this the current term
            </label>
          )}
          <FormActions error={formError} submitting={submitting} label="Save term" onCancel={close} />
        </form>
      </Modal>

      {/* Programme modal */}
      <Modal open={modal?.kind === "programme"} onClose={close} title="Add programme">
        <form onSubmit={submitProgramme} className="space-y-4">
          <TextField
            id="programme-code"
            label="Code *"
            required
            value={progForm.code}
            onChange={(e) => setProgForm({ ...progForm, code: e.target.value })}
            placeholder="GS"
          />
          <TextField
            id="programme-name"
            label="Name *"
            required
            value={progForm.name}
            onChange={(e) => setProgForm({ ...progForm, name: e.target.value })}
            placeholder="General Science"
          />
          <TextAreaField
            label="Description"
            value={progForm.description}
            onChange={(e) => setProgForm({ ...progForm, description: e.target.value })}
          />
          <FormActions error={formError} submitting={submitting} label="Create programme" onCancel={close} />
        </form>
      </Modal>

      {/* Subject modal */}
      <Modal open={modal?.kind === "subject"} onClose={close} title="Add subject">
        <form onSubmit={submitSubject} className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <TextField
              id="subject-code"
              label="Code *"
              required
              value={subjForm.code}
              onChange={(e) => setSubjForm({ ...subjForm, code: e.target.value })}
              placeholder="HIST"
            />
            <SelectField
              label="Type"
              value={subjForm.type}
              onChange={(e) => setSubjForm({ ...subjForm, type: e.target.value })}
            >
              {SUBJECT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </SelectField>
          </div>
          <TextField
            id="subject-name"
            label="Name *"
            required
            value={subjForm.name}
            onChange={(e) => setSubjForm({ ...subjForm, name: e.target.value })}
            placeholder="History"
          />
          <TextAreaField
            label="Description"
            value={subjForm.description}
            onChange={(e) => setSubjForm({ ...subjForm, description: e.target.value })}
          />
          <FormActions error={formError} submitting={submitting} label="Create subject" onCancel={close} />
        </form>
      </Modal>

      {/* Class modal */}
      <Modal open={modal?.kind === "class"} onClose={close} title="Add class">
        <form onSubmit={submitClass} className="space-y-4">
          <SelectField
            label="Academic year *"
            required
            value={classForm.academic_year_id}
            onChange={(e) => setClassForm({ ...classForm, academic_year_id: e.target.value })}
          >
            {years.map((y) => (
              <option key={y.id} value={y.id}>
                {y.name}
              </option>
            ))}
          </SelectField>
          <div className="grid grid-cols-2 gap-3">
            <TextField
              id="class-name"
              label="Name *"
              required
              value={classForm.name}
              onChange={(e) => setClassForm({ ...classForm, name: e.target.value })}
              placeholder="Form 2A"
            />
            <TextField
              label="Level"
              value={classForm.level}
              onChange={(e) => setClassForm({ ...classForm, level: e.target.value })}
              placeholder="Form 2"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <SelectField
              label="Programme"
              value={classForm.programme_id}
              onChange={(e) => setClassForm({ ...classForm, programme_id: e.target.value })}
            >
              <option value="">—</option>
              {programmes.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </SelectField>
            <TextField
              label="Capacity"
              type="number"
              min={1}
              value={classForm.capacity}
              onChange={(e) => setClassForm({ ...classForm, capacity: e.target.value })}
            />
          </div>
          <FormActions error={formError} submitting={submitting} label="Create class" onCancel={close} />
        </form>
      </Modal>
    </>
  );
}

export default function AcademicsPage() {
  return (
    <ProtectedRoute requiredPermission="class.view">
      <AppShell title="Academic Structure">
        <AcademicStructure />
      </AppShell>
    </ProtectedRoute>
  );
}
