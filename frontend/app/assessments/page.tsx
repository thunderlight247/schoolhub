"use client";

// Assessments & Results Workspace (M29)
// Full teacher/admin assessment creation, assessment directory,
// batch score sheet entry, and single-result mark corrections.
// Gated under assessment.view / result.view (shared by ADMIN and TEACHER).

import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { StatCard } from "@/components/ui/stat-card";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  listAssessments,
  getAssessment,
  createAssessment,
  updateAssessment,
  deleteAssessment,
  listAssessmentResults,
  recordResults,
  updateResult,
} from "@/services/assessments";
import { listClasses, listSubjects, listTerms, listClassSubjects } from "@/services/academics";
import { listEnrollments } from "@/services/enrollments";
import type {
  Assessment,
  AssessmentType,
  ResultItem,
} from "@/types/assessments";
import { ASSESSMENT_TYPES } from "@/types/assessments";
import type { Class, Subject, Term, ClassSubject } from "@/types/academics";

const PAGE_SIZE = 10;

function assessmentTypeBadge(type: AssessmentType | string) {
  const styles: Record<string, string> = {
    QUIZ: "bg-sky-50 text-sky-700 ring-sky-600/20",
    TEST: "bg-indigo-50 text-indigo-700 ring-indigo-600/20",
    ASSIGNMENT: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
    PROJECT: "bg-amber-50 text-amber-700 ring-amber-600/20",
    MIDTERM: "bg-orange-50 text-orange-700 ring-orange-600/20",
    FINAL: "bg-purple-50 text-purple-700 ring-purple-600/20",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ring-1 ring-inset ${
        styles[type] || "bg-slate-100 text-slate-600 ring-slate-500/20"
      }`}
    >
      {type}
    </span>
  );
}

function gradeBadge(grade?: string | null) {
  if (!grade) return <span className="text-slate-400 text-xs">—</span>;
  const g = grade.toUpperCase();
  const styles: Record<string, string> = {
    A: "bg-emerald-100 text-emerald-800",
    B: "bg-sky-100 text-sky-800",
    C: "bg-amber-100 text-amber-800",
    D: "bg-orange-100 text-orange-800",
    F: "bg-rose-100 text-rose-800",
  };
  return (
    <span
      className={`inline-flex items-center rounded px-1.5 py-0.5 text-xs font-bold ${
        styles[g] || "bg-slate-100 text-slate-700"
      }`}
    >
      {grade}
    </span>
  );
}

interface NewScoreRow {
  student_id: string;
  student_name: string;
  student_number: string;
  score: string;
  grade: string;
  remarks: string;
}

function AssessmentsWorkspace() {
  const { hasPermission } = useAuth();
  const canCreate = hasPermission("assessment.create");
  const canUpdate = hasPermission("assessment.update");
  const canRecordResults = hasPermission("result.create");
  const canUpdateResult = hasPermission("result.update");

  // Assessment directory state
  const [assessments, setAssessments] = useState<Assessment[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  // Filters
  const [filterClassId, setFilterClassId] = useState("");
  const [filterSubjectId, setFilterSubjectId] = useState("");
  const [filterTermId, setFilterTermId] = useState("");
  const [filterType, setFilterType] = useState("");
  const [search, setSearch] = useState("");

  // Metadata dropdowns
  const [classes, setClasses] = useState<Class[]>([]);
  const [subjects, setSubjects] = useState<Subject[]>([]);
  const [terms, setTerms] = useState<Term[]>([]);

  // Create Assessment Modal state
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [createClassId, setCreateClassId] = useState("");
  const [createSubjectId, setCreateSubjectId] = useState("");
  const [createTermId, setCreateTermId] = useState("");
  const [createTitle, setCreateTitle] = useState("");
  const [createType, setCreateType] = useState<AssessmentType>("QUIZ");
  const [createMaxScore, setCreateMaxScore] = useState("100");
  const [createWeight, setCreateWeight] = useState("");
  const [createDate, setCreateDate] = useState(() => new Date().toISOString().split("T")[0]);
  const [classSubjects, setClassSubjects] = useState<ClassSubject[]>([]);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  // Edit Assessment Header Modal state
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingAssessment, setEditingAssessment] = useState<Assessment | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [editType, setEditType] = useState<AssessmentType>("QUIZ");
  const [editMaxScore, setEditMaxScore] = useState("100");
  const [editWeight, setEditWeight] = useState("");
  const [editDate, setEditDate] = useState("");
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  // Delete Assessment Confirmation
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [deletingAssessment, setDeletingAssessment] = useState<Assessment | null>(null);
  const [deleteSubmitting, setDeleteSubmitting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  // Score Sheet Modal state
  const [scoreSheetModalOpen, setScoreSheetModalOpen] = useState(false);
  const [activeAssessment, setActiveAssessment] = useState<Assessment | null>(null);
  const [results, setResults] = useState<ResultItem[]>([]);
  const [scoreSheetLoading, setScoreSheetLoading] = useState(false);
  const [scoreSheetError, setScoreSheetError] = useState<string | null>(null);

  // Fresh Score Entry Rows (when results.length === 0)
  const [newScoreRows, setNewScoreRows] = useState<NewScoreRow[]>([]);
  const [batchSubmitting, setBatchSubmitting] = useState(false);
  const [batchError, setBatchError] = useState<string | null>(null);

  // Inline result update state
  const [savingResultId, setSavingResultId] = useState<string | null>(null);
  const [resultFeedback, setResultFeedback] = useState<Record<string, string>>({});

  // Load metadata options
  useEffect(() => {
    listClasses().then(setClasses).catch(() => {});
    listSubjects().then(setSubjects).catch(() => {});
    listTerms().then(setTerms).catch(() => {});
  }, []);

  // Fetch assessment directory
  const loadAssessments = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await listAssessments({
        class_id: filterClassId || undefined,
        subject_id: filterSubjectId || undefined,
        term_id: filterTermId || undefined,
        type: filterType || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setAssessments(res.assessments || []);
      setTotal(res.pagination?.total || 0);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not load assessments.");
    } finally {
      setLoading(false);
    }
  }, [filterClassId, filterSubjectId, filterTermId, filterType, page]);

  useEffect(() => {
    loadAssessments();
  }, [loadAssessments]);

  // Client-side search filtering
  const filteredAssessments = useMemo(() => {
    if (!search.trim()) return assessments;
    const q = search.toLowerCase();
    return assessments.filter(
      (a) =>
        a.title.toLowerCase().includes(q) ||
        a.type.toLowerCase().includes(q) ||
        a.class_name?.toLowerCase().includes(q) ||
        a.subject_name?.toLowerCase().includes(q) ||
        a.subject_code?.toLowerCase().includes(q) ||
        a.term_name?.toLowerCase().includes(q) ||
        a.assessment_date.includes(q),
    );
  }, [assessments, search]);

  // Available terms for the selected class in Create Assessment
  const selectedClass = useMemo(
    () => classes.find((c) => c.id === createClassId),
    [classes, createClassId],
  );

  const availableClassTerms = useMemo(() => {
    if (!selectedClass) return terms;
    const matching = terms.filter((t) => t.academic_year_id === selectedClass.academic_year_id);
    return matching.length > 0 ? matching : terms;
  }, [selectedClass, terms]);

  // Handle class selection in Create Assessment modal
  const handleCreateClassChange = async (cid: string) => {
    setCreateClassId(cid);
    setCreateSubjectId("");
    setCreateError(null);
    if (!cid) return;

    const cls = classes.find((c) => c.id === cid);
    if (cls) {
      const matchedTerms = terms.filter((t) => t.academic_year_id === cls.academic_year_id);
      const cur = matchedTerms.find((t) => t.is_current) || matchedTerms[0] || terms[0];
      if (cur) setCreateTermId(cur.id);
    }

    try {
      const subs = await listClassSubjects(cid);
      setClassSubjects(subs || []);
      if (subs && subs.length > 0) {
        setCreateSubjectId(subs[0].id);
      }
    } catch {
      setClassSubjects([]);
    }
  };

  // Submit new assessment
  const handleCreateAssessmentSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!createClassId || !createSubjectId || !createTermId || !createTitle.trim() || !createDate) {
      setCreateError("Please fill in all required fields.");
      return;
    }
    const maxScoreNum = parseFloat(createMaxScore);
    if (isNaN(maxScoreNum) || maxScoreNum <= 0) {
      setCreateError("Max score must be greater than 0.");
      return;
    }
    const weightNum = createWeight.trim() ? parseFloat(createWeight) : null;
    if (weightNum !== null && (isNaN(weightNum) || weightNum < 0 || weightNum > 100)) {
      setCreateError("Weight must be between 0 and 100.");
      return;
    }

    setCreateSubmitting(true);
    setCreateError(null);
    try {
      await createAssessment({
        class_id: createClassId,
        subject_id: createSubjectId,
        term_id: createTermId,
        title: createTitle.trim(),
        type: createType,
        max_score: maxScoreNum,
        weight: weightNum,
        assessment_date: createDate,
      });

      setShowCreateModal(false);
      setNotice("Assessment successfully created.");
      setCreateTitle("");
      setCreateClassId("");
      setCreateSubjectId("");
      setCreateTermId("");
      loadAssessments();
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.code === "ALREADY_EXISTS" || err.status === 409) {
          setCreateError(
            "An assessment with this title already exists for this class, subject, and term.",
          );
        } else {
          setCreateError(err.message);
        }
      } else {
        setCreateError("Failed to create assessment.");
      }
    } finally {
      setCreateSubmitting(false);
    }
  };

  // Open Edit Assessment Header Modal
  const handleOpenEdit = (a: Assessment) => {
    setEditingAssessment(a);
    setEditTitle(a.title);
    setEditType(a.type);
    setEditMaxScore(String(a.max_score));
    setEditWeight(a.weight !== null && a.weight !== undefined ? String(a.weight) : "");
    setEditDate(a.assessment_date);
    setEditError(null);
    setEditModalOpen(true);
  };

  // Submit Edit Assessment
  const handleEditSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!editingAssessment) return;

    const maxScoreNum = parseFloat(editMaxScore);
    if (isNaN(maxScoreNum) || maxScoreNum <= 0) {
      setEditError("Max score must be greater than 0.");
      return;
    }
    const weightNum = editWeight.trim() ? parseFloat(editWeight) : null;
    if (weightNum !== null && (isNaN(weightNum) || weightNum < 0 || weightNum > 100)) {
      setEditError("Weight must be between 0 and 100.");
      return;
    }

    setEditSubmitting(true);
    setEditError(null);
    try {
      await updateAssessment(editingAssessment.id, {
        title: editTitle.trim(),
        type: editType,
        max_score: maxScoreNum,
        weight: weightNum,
        assessment_date: editDate,
      });
      setEditModalOpen(false);
      setNotice("Assessment header updated.");
      loadAssessments();
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : "Failed to update assessment.");
    } finally {
      setEditSubmitting(false);
    }
  };

  // Open Delete Confirmation
  const handleOpenDelete = (a: Assessment) => {
    setDeletingAssessment(a);
    setDeleteError(null);
    setDeleteModalOpen(true);
  };

  // Confirm Delete Assessment
  const handleDeleteConfirm = async () => {
    if (!deletingAssessment) return;
    setDeleteSubmitting(true);
    setDeleteError(null);
    try {
      await deleteAssessment(deletingAssessment.id);
      setDeleteModalOpen(false);
      setNotice("Assessment deleted.");
      loadAssessments();
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : "Failed to delete assessment.");
    } finally {
      setDeleteSubmitting(false);
    }
  };

  // Open Score Sheet Modal
  const handleOpenScoreSheet = async (a: Assessment) => {
    setActiveAssessment(a);
    setScoreSheetModalOpen(true);
    setScoreSheetLoading(true);
    setScoreSheetError(null);
    setBatchError(null);
    setResultFeedback({});
    try {
      const res = await listAssessmentResults(a.id);
      setResults(res || []);

      // If no results recorded yet, load enrolled students from this class
      if (!res || res.length === 0) {
        const enrRes = await listEnrollments({
          class_id: a.class_id,
          status: "ENROLLED",
          page_size: 100,
        });
        const rows: NewScoreRow[] = (enrRes.enrollments || []).map((e) => ({
          student_id: e.student_id,
          student_name: e.student_name || "Enrolled Student",
          student_number: e.student_number || "",
          score: "",
          grade: "",
          remarks: "",
        }));
        setNewScoreRows(rows);
      }
    } catch (err) {
      setScoreSheetError(err instanceof ApiError ? err.message : "Failed to load score sheet.");
    } finally {
      setScoreSheetLoading(false);
    }
  };

  // Submit fresh batch score sheet
  const handleBatchScoreSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!activeAssessment) return;
    if (newScoreRows.length === 0) {
      setBatchError("No enrolled students found to grade.");
      return;
    }

    setBatchSubmitting(true);
    setBatchError(null);
    try {
      const payload = newScoreRows.map((r) => {
        const parsed = r.score.trim() ? parseFloat(r.score) : null;
        return {
          student_id: r.student_id,
          score: parsed,
          grade: r.grade.trim() || null,
          remarks: r.remarks.trim() || null,
        };
      });

      const res = await recordResults(activeAssessment.id, payload);
      setResults(res || []);
      setNotice("Score sheet successfully recorded.");
    } catch (err) {
      setBatchError(err instanceof ApiError ? err.message : "Failed to save score sheet.");
    } finally {
      setBatchSubmitting(false);
    }
  };

  // Inline result update (single result correction)
  const handleUpdateResultRecord = async (
    resultId: string,
    scoreVal: string,
    gradeVal: string,
    remarksVal: string,
  ) => {
    setSavingResultId(resultId);
    setResultFeedback((prev) => ({ ...prev, [resultId]: "" }));
    try {
      const parsedScore = scoreVal.trim() ? parseFloat(scoreVal) : null;
      const updated = await updateResult(resultId, {
        score: parsedScore,
        grade: gradeVal.trim() || null,
        remarks: remarksVal.trim() || null,
      });

      setResults((prev) =>
        prev.map((r) =>
          r.id === resultId
            ? { ...r, score: updated.score, grade: updated.grade, remarks: updated.remarks }
            : r,
        ),
      );
      setResultFeedback((prev) => ({ ...prev, [resultId]: "Saved!" }));
      setTimeout(() => {
        setResultFeedback((prev) => {
          const next = { ...prev };
          delete next[resultId];
          return next;
        });
      }, 2500);
    } catch (err) {
      setScoreSheetError(err instanceof ApiError ? err.message : "Failed to update student result.");
    } finally {
      setSavingResultId(null);
    }
  };

  // Score sheet metrics
  const scoreStats = useMemo(() => {
    if (!activeAssessment || results.length === 0) return null;
    const scores = results
      .map((r) => r.score)
      .filter((s): s is number => s !== null && s !== undefined);

    if (scores.length === 0) {
      return { total: results.length, graded: 0, avg: 0, high: 0, low: 0, avgPct: 0 };
    }

    const sum = scores.reduce((a, b) => a + b, 0);
    const avg = sum / scores.length;
    const high = Math.max(...scores);
    const low = Math.min(...scores);
    const avgPct = activeAssessment.max_score > 0 ? (avg / activeAssessment.max_score) * 100 : 0;

    return {
      total: results.length,
      graded: scores.length,
      avg: Number(avg.toFixed(1)),
      high,
      low,
      avgPct: Math.round(avgPct),
    };
  }, [activeAssessment, results]);

  // Columns for Directory table
  const columns: Column<Assessment>[] = [
    {
      key: "assessment_date",
      header: "Date",
      render: (a) => (
        <span className="font-mono text-xs font-semibold text-slate-800">
          {a.assessment_date}
        </span>
      ),
    },
    {
      key: "title",
      header: "Title",
      render: (a) => (
        <div className="flex flex-col">
          <span className="font-bold text-slate-900">{a.title}</span>
          <span className="text-[11px] text-slate-400">
            Max: {a.max_score} pts {a.weight ? `• Weight: ${a.weight}%` : ""}
          </span>
        </div>
      ),
    },
    {
      key: "type",
      header: "Type",
      render: (a) => assessmentTypeBadge(a.type),
    },
    {
      key: "class_name",
      header: "Class",
      render: (a) => (
        <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">
          {a.class_name || "—"}
        </span>
      ),
    },
    {
      key: "subject",
      header: "Subject",
      render: (a) => (
        <div className="flex flex-col">
          <span className="font-medium text-slate-800">{a.subject_name || "—"}</span>
          {a.subject_code && (
            <span className="text-[11px] font-mono text-slate-400">{a.subject_code}</span>
          )}
        </div>
      ),
    },
    {
      key: "term_name",
      header: "Term",
      hideBelow: "sm",
      render: (a) => <span className="text-xs text-slate-600">{a.term_name || "—"}</span>,
    },
    {
      key: "actions",
      header: "Actions",
      className: "text-right",
      render: (a) => (
        <div className="flex items-center justify-end gap-1.5">
          <button
            onClick={() => handleOpenScoreSheet(a)}
            className="rounded-lg border border-brand-200 bg-brand-50/70 px-2.5 py-1 text-xs font-semibold text-brand-800 shadow-xs hover:bg-brand-100 transition-colors"
          >
            Score Sheet
          </button>
          {canUpdate && (
            <>
              <button
                onClick={() => handleOpenEdit(a)}
                className="rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50 transition-colors"
                title="Edit Assessment"
              >
                Edit
              </button>
              <button
                onClick={() => handleOpenDelete(a)}
                className="rounded-lg border border-rose-200 bg-rose-50/50 px-2 py-1 text-xs font-medium text-rose-700 hover:bg-rose-100 transition-colors"
                title="Delete Assessment"
              >
                Delete
              </button>
            </>
          )}
        </div>
      ),
    },
  ];

  const resetFilters = () => {
    setFilterClassId("");
    setFilterSubjectId("");
    setFilterTermId("");
    setFilterType("");
    setSearch("");
    setPage(1);
  };

  const hasActiveFilters = Boolean(
    filterClassId || filterSubjectId || filterTermId || filterType || search,
  );

  return (
    <AppShell
      title="Assessments & Results"
      actions={
        canCreate && (
          <button
            onClick={() => {
              setShowCreateModal(true);
              setCreateError(null);
            }}
            className="inline-flex items-center gap-1.5 rounded-lg bg-brand-700 px-3.5 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-brand-600 transition-colors"
          >
            <span>+</span> New Assessment
          </button>
        )
      }
    >
      <div className="space-y-6">
        {/* KPI Stat Cards */}
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <StatCard
            label="Total Assessments"
            value={total}
            tone="sky"
            chip="Academic Records"
          />
          <StatCard
            label="Class Cohorts"
            value={classes.length}
            tone="purple"
            chip="Assigned"
          />
          <StatCard
            label="Subjects Evaluated"
            value={subjects.length}
            tone="green"
            chip="Curriculum"
          />
          <StatCard
            label="Terms Active"
            value={terms.length}
            tone="yellow"
            chip="Calendar"
          />
        </div>

        {/* Notice alert */}
        {notice && (
          <div className="flex items-center justify-between rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
            <span>{notice}</span>
            <button
              onClick={() => setNotice(null)}
              className="text-xs font-semibold text-emerald-600 hover:text-emerald-800"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Directory Card */}
        <div className="space-y-4 rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 className="text-base font-bold text-slate-900">Assessments Directory</h2>
              <p className="text-xs text-slate-500">
                Manage graded quizzes, tests, assignments, and terminal examinations
              </p>
            </div>
            <div className="flex items-center gap-2">
              <SearchInput
                placeholder="Search title, class, subject..."
                onSearch={setSearch}
              />
              {hasActiveFilters && (
                <button
                  onClick={resetFilters}
                  className="rounded-lg border border-slate-200 bg-white px-2.5 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50"
                >
                  Reset
                </button>
              )}
            </div>
          </div>

          {/* Filter Bar */}
          <div className="grid grid-cols-1 gap-3 pt-2 sm:grid-cols-2 lg:grid-cols-4">
            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Class
              </label>
              <select
                value={filterClassId}
                onChange={(e) => {
                  setFilterClassId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Classes</option>
                {classes.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Subject
              </label>
              <select
                value={filterSubjectId}
                onChange={(e) => {
                  setFilterSubjectId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Subjects</option>
                {subjects.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.code} — {s.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Term
              </label>
              <select
                value={filterTermId}
                onChange={(e) => {
                  setFilterTermId(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Terms</option>
                {terms.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-1">
                Assessment Type
              </label>
              <select
                value={filterType}
                onChange={(e) => {
                  setFilterType(e.target.value);
                  setPage(1);
                }}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs text-slate-700 focus:border-brand-600 focus:outline-none focus:ring-1 focus:ring-brand-600"
              >
                <option value="">All Types</option>
                {ASSESSMENT_TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Table */}
          {error && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-4 text-xs text-rose-700">
              {error}
            </div>
          )}

          {loading ? (
            <div className="py-12 text-center text-sm text-slate-400">Loading assessments…</div>
          ) : (
            <>
              <DataTable
                columns={columns}
                rows={filteredAssessments}
                getRowKey={(a) => a.id}
                emptyMessage="No assessments found matching the current filters."
              />
              <Pagination
                page={page}
                totalItems={total}
                pageSize={PAGE_SIZE}
                onPageChange={(newPage) => setPage(newPage)}
              />
            </>
          )}
        </div>
      </div>

      {/* Create Assessment Modal */}
      <Modal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        title="Create New Assessment"
        maxWidth="max-w-2xl"
      >
        <form onSubmit={handleCreateAssessmentSubmit} className="space-y-4">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SelectField
              id="create-class"
              label="Class *"
              value={createClassId}
              onChange={(e) => handleCreateClassChange(e.target.value)}
              required
            >
              <option value="">Choose a class…</option>
              {classes.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </SelectField>

            <SelectField
              id="create-subject"
              label="Subject *"
              value={createSubjectId}
              onChange={(e) => setCreateSubjectId(e.target.value)}
              disabled={!createClassId || classSubjects.length === 0}
              required
            >
              <option value="">
                {!createClassId
                  ? "Select class first"
                  : classSubjects.length === 0
                  ? "No subjects assigned to class"
                  : "Choose subject…"}
              </option>
              {classSubjects.map((cs) => (
                <option key={cs.id} value={cs.id}>
                  {cs.code} — {cs.name}
                </option>
              ))}
            </SelectField>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SelectField
              id="create-term"
              label="Academic Term *"
              value={createTermId}
              onChange={(e) => setCreateTermId(e.target.value)}
              disabled={!createClassId}
              required
            >
              <option value="">Choose term…</option>
              {availableClassTerms.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} {t.is_current ? "(Current)" : ""}
                </option>
              ))}
            </SelectField>

            <SelectField
              id="create-type"
              label="Assessment Type *"
              value={createType}
              onChange={(e) => setCreateType(e.target.value as AssessmentType)}
              required
            >
              {ASSESSMENT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </SelectField>
          </div>

          <TextField
            id="create-title"
            label="Assessment Title *"
            placeholder="e.g. Midterm Test, Chapter 4 Quiz, Term Paper"
            value={createTitle}
            onChange={(e) => setCreateTitle(e.target.value)}
            required
          />

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <TextField
              id="create-max-score"
              label="Max Score *"
              type="number"
              step="any"
              min="0.1"
              value={createMaxScore}
              onChange={(e) => setCreateMaxScore(e.target.value)}
              required
            />

            <TextField
              id="create-weight"
              label="Weight % (Optional)"
              type="number"
              step="any"
              min="0"
              max="100"
              placeholder="e.g. 20"
              value={createWeight}
              onChange={(e) => setCreateWeight(e.target.value)}
            />

            <TextField
              id="create-date"
              label="Assessment Date *"
              type="date"
              value={createDate}
              onChange={(e) => setCreateDate(e.target.value)}
              required
            />
          </div>

          <FormActions
            error={createError}
            submitting={createSubmitting}
            label="Create Assessment"
            onCancel={() => setShowCreateModal(false)}
          />
        </form>
      </Modal>

      {/* Edit Assessment Header Modal */}
      <Modal
        open={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        title="Edit Assessment Header"
        maxWidth="max-w-lg"
      >
        <form onSubmit={handleEditSubmit} className="space-y-4">
          <TextField
            id="edit-title"
            label="Assessment Title *"
            value={editTitle}
            onChange={(e) => setEditTitle(e.target.value)}
            required
          />

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <SelectField
              id="edit-type"
              label="Type *"
              value={editType}
              onChange={(e) => setEditType(e.target.value as AssessmentType)}
              required
            >
              {ASSESSMENT_TYPES.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </SelectField>

            <TextField
              id="edit-date"
              label="Assessment Date *"
              type="date"
              value={editDate}
              onChange={(e) => setEditDate(e.target.value)}
              required
            />
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <TextField
              id="edit-max-score"
              label="Max Score *"
              type="number"
              step="any"
              min="0.1"
              value={editMaxScore}
              onChange={(e) => setEditMaxScore(e.target.value)}
              required
            />

            <TextField
              id="edit-weight"
              label="Weight % (Optional)"
              type="number"
              step="any"
              min="0"
              max="100"
              value={editWeight}
              onChange={(e) => setEditWeight(e.target.value)}
            />
          </div>

          <FormActions
            error={editError}
            submitting={editSubmitting}
            label="Save Changes"
            onCancel={() => setEditModalOpen(false)}
          />
        </form>
      </Modal>

      {/* Delete Assessment Confirmation Modal */}
      <Modal
        open={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        title="Delete Assessment"
        maxWidth="max-w-md"
      >
        <div className="space-y-4">
          {deleteError && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
              {deleteError}
            </div>
          )}
          <p className="text-sm text-slate-600">
            Are you sure you want to delete{" "}
            <strong className="text-slate-900">{deletingAssessment?.title}</strong>? This action
            will also permanently remove all recorded student results for this assessment.
          </p>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={() => setDeleteModalOpen(false)}
              className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={deleteSubmitting}
              onClick={handleDeleteConfirm}
              className="rounded-lg bg-rose-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-rose-700 disabled:opacity-50"
            >
              {deleteSubmitting ? "Deleting…" : "Delete Assessment"}
            </button>
          </div>
        </div>
      </Modal>

      {/* Score Sheet Entry & Correction Modal */}
      <Modal
        open={scoreSheetModalOpen}
        onClose={() => setScoreSheetModalOpen(false)}
        title={activeAssessment ? `Score Sheet: ${activeAssessment.title}` : "Score Sheet"}
        maxWidth="max-w-4xl"
      >
        <div className="space-y-4">
          {activeAssessment && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="font-bold text-slate-900 text-sm">
                    {activeAssessment.class_name || "Class"}
                  </span>
                  <span className="rounded bg-brand-50 px-2 py-0.5 text-xs font-semibold text-brand-700 border border-brand-100">
                    {activeAssessment.subject_code} — {activeAssessment.subject_name}
                  </span>
                  {assessmentTypeBadge(activeAssessment.type)}
                </div>
                <p className="text-xs text-slate-500">
                  {activeAssessment.term_name} &bull; Date:{" "}
                  <strong className="font-mono text-slate-700">
                    {activeAssessment.assessment_date}
                  </strong>{" "}
                  &bull; Max Score:{" "}
                  <strong className="text-slate-800">{activeAssessment.max_score} pts</strong>
                  {activeAssessment.weight ? ` • Weight: ${activeAssessment.weight}%` : ""}
                </p>
              </div>

              {/* Class Summary Pills */}
              {scoreStats && (
                <div className="flex flex-wrap items-center gap-2 text-xs">
                  <span className="rounded-lg bg-white border border-slate-200 px-2.5 py-1 font-medium text-slate-700">
                    Graded:{" "}
                    <strong>
                      {scoreStats.graded} / {scoreStats.total}
                    </strong>
                  </span>
                  <span className="rounded-lg bg-emerald-50 border border-emerald-200 px-2.5 py-1 font-bold text-emerald-800">
                    Class Avg: {scoreStats.avg} ({scoreStats.avgPct}%)
                  </span>
                  <span className="rounded-lg bg-sky-50 border border-sky-200 px-2.5 py-1 font-semibold text-sky-800">
                    High: {scoreStats.high}
                  </span>
                  <span className="rounded-lg bg-amber-50 border border-amber-200 px-2.5 py-1 font-semibold text-amber-800">
                    Low: {scoreStats.low}
                  </span>
                </div>
              )}
            </div>
          )}

          {scoreSheetError && (
            <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
              {scoreSheetError}
            </div>
          )}

          {scoreSheetLoading ? (
            <div className="py-12 text-center text-xs text-slate-400">Loading student scores…</div>
          ) : results.length > 0 ? (
            /* Mode 1: Results Already Recorded (Viewing & Individual Corrections) */
            <div className="space-y-2">
              <div className="flex items-center justify-between pb-1 border-b border-slate-200">
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                  Enrolled Students ({results.length})
                </span>
                <span className="text-xs text-slate-400">
                  Max score: {activeAssessment?.max_score} pts
                </span>
              </div>

              <div className="max-h-[50vh] overflow-y-auto space-y-2 pr-1">
                {results.map((rec) => {
                  const feedback = resultFeedback[rec.id];
                  const isSaving = savingResultId === rec.id;

                  return (
                    <div
                      key={rec.id}
                      className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-xs sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <p className="font-semibold text-xs text-slate-900">
                            {rec.student_name || "Student"}
                          </p>
                          {gradeBadge(rec.grade)}
                        </div>
                        <p className="text-[11px] font-mono text-slate-400">
                          {rec.student_number || "No student #"}
                        </p>
                      </div>

                      {canUpdateResult ? (
                        <div className="flex flex-wrap items-center gap-2">
                          <div className="flex items-center gap-1">
                            <label className="text-[11px] text-slate-400">Score:</label>
                            <input
                              type="number"
                              step="any"
                              min="0"
                              max={activeAssessment?.max_score || 100}
                              defaultValue={rec.score !== null ? String(rec.score) : ""}
                              id={`score-${rec.id}`}
                              className="w-20 rounded border border-slate-200 px-2 py-1 text-xs text-slate-800 font-semibold focus:border-brand-500 focus:outline-none"
                            />
                          </div>

                          <div className="flex items-center gap-1">
                            <label className="text-[11px] text-slate-400">Grade:</label>
                            <input
                              type="text"
                              maxLength={3}
                              placeholder="A/B/C"
                              defaultValue={rec.grade || ""}
                              id={`grade-${rec.id}`}
                              className="w-14 rounded border border-slate-200 px-2 py-1 text-xs text-slate-800 uppercase focus:border-brand-500 focus:outline-none"
                            />
                          </div>

                          <div className="flex items-center gap-1">
                            <input
                              type="text"
                              placeholder="Remarks…"
                              defaultValue={rec.remarks || ""}
                              id={`remarks-${rec.id}`}
                              className="w-36 rounded border border-slate-200 px-2 py-1 text-xs text-slate-700 focus:border-brand-500 focus:outline-none"
                            />
                          </div>

                          <button
                            type="button"
                            disabled={isSaving}
                            onClick={() => {
                              const sInput = document.getElementById(
                                `score-${rec.id}`,
                              ) as HTMLInputElement;
                              const gInput = document.getElementById(
                                `grade-${rec.id}`,
                              ) as HTMLInputElement;
                              const rInput = document.getElementById(
                                `remarks-${rec.id}`,
                              ) as HTMLInputElement;
                              handleUpdateResultRecord(
                                rec.id,
                                sInput?.value || "",
                                gInput?.value || "",
                                rInput?.value || "",
                              );
                            }}
                            className="rounded bg-brand-700 px-2.5 py-1 text-xs font-semibold text-white shadow-xs hover:bg-brand-600 disabled:opacity-50"
                          >
                            Save
                          </button>

                          {feedback && (
                            <span className="text-[11px] font-bold text-emerald-600">
                              {feedback}
                            </span>
                          )}
                          {isSaving && (
                            <span className="text-[11px] text-slate-400">Saving…</span>
                          )}
                        </div>
                      ) : (
                        <div className="text-xs font-bold text-slate-800">
                          {rec.score !== null ? `${rec.score} pts` : "Unmarked"}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          ) : (
            /* Mode 2: Fresh Score Sheet Entry (Batch POST /assessments/:id/results) */
            <form onSubmit={handleBatchScoreSubmit} className="space-y-4">
              <div className="flex items-center justify-between pb-1 border-b border-slate-200">
                <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                  Enter Scores for Active Class Enrollees ({newScoreRows.length})
                </span>
                <span className="text-xs font-mono text-slate-500">
                  Max: {activeAssessment?.max_score} pts
                </span>
              </div>

              {batchError && (
                <div className="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700">
                  {batchError}
                </div>
              )}

              {newScoreRows.length === 0 ? (
                <div className="py-8 text-center text-xs text-amber-600">
                  No active students enrolled in this class.
                </div>
              ) : (
                <div className="max-h-[50vh] overflow-y-auto space-y-2 pr-1">
                  {newScoreRows.map((student, idx) => (
                    <div
                      key={student.student_id}
                      className="flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 shadow-xs sm:flex-row sm:items-center sm:justify-between"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="font-semibold text-xs text-slate-900 truncate">
                          {student.student_name}
                        </p>
                        <p className="text-[11px] font-mono text-slate-400">
                          {student.student_number || "No student #"}
                        </p>
                      </div>

                      <div className="flex flex-wrap items-center gap-2">
                        <div className="flex items-center gap-1">
                          <label className="text-[11px] text-slate-400">Score:</label>
                          <input
                            type="number"
                            step="any"
                            min="0"
                            max={activeAssessment?.max_score || 100}
                            placeholder="0"
                            value={student.score}
                            onChange={(e) => {
                              const val = e.target.value;
                              setNewScoreRows((prev) =>
                                prev.map((s, i) => (i === idx ? { ...s, score: val } : s)),
                              );
                            }}
                            className="w-20 rounded border border-slate-200 px-2 py-1 text-xs text-slate-800 font-semibold focus:border-brand-500 focus:outline-none"
                          />
                        </div>

                        <div className="flex items-center gap-1">
                          <label className="text-[11px] text-slate-400">Grade:</label>
                          <input
                            type="text"
                            maxLength={3}
                            placeholder="A/B/C"
                            value={student.grade}
                            onChange={(e) => {
                              const val = e.target.value;
                              setNewScoreRows((prev) =>
                                prev.map((s, i) => (i === idx ? { ...s, grade: val } : s)),
                              );
                            }}
                            className="w-14 rounded border border-slate-200 px-2 py-1 text-xs text-slate-800 uppercase focus:border-brand-500 focus:outline-none"
                          />
                        </div>

                        <div className="flex items-center gap-1">
                          <input
                            type="text"
                            placeholder="Optional remarks…"
                            value={student.remarks}
                            onChange={(e) => {
                              const val = e.target.value;
                              setNewScoreRows((prev) =>
                                prev.map((s, i) => (i === idx ? { ...s, remarks: val } : s)),
                              );
                            }}
                            className="w-36 rounded border border-slate-200 px-2 py-1 text-xs text-slate-700 focus:border-brand-500 focus:outline-none"
                          />
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {canRecordResults && newScoreRows.length > 0 && (
                <FormActions
                  error={batchError}
                  submitting={batchSubmitting}
                  label="Save Score Sheet"
                  onCancel={() => setScoreSheetModalOpen(false)}
                />
              )}
            </form>
          )}
        </div>
      </Modal>
    </AppShell>
  );
}

export default function AssessmentsPage() {
  return (
    <ProtectedRoute requiredPermission="assessment.view">
      <AssessmentsWorkspace />
    </ProtectedRoute>
  );
}
