"use client";

// Announcements workspace (M30): composer + directory for the M13 backend
// domain. Admins/staff with announcement.create can draft school-wide
// broadcasts or messages targeted at a role or a class cohort, publish
// (school-wide publishes trigger the backend's staff email notification),
// archive, and delete. Gated under announcement.view/create/update.

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { useAuth } from "@/components/auth-provider";
import { DataTable, type Column } from "@/components/ui/table";
import { StatCard } from "@/components/ui/stat-card";
import { SearchInput } from "@/components/ui/search-input";
import { Pagination } from "@/components/ui/pagination";
import { Modal } from "@/components/ui/modal";
import { TextField, SelectField, TextAreaField, FormActions } from "@/components/ui/form";
import { ApiError } from "@/lib/api";
import {
  listAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  publishAnnouncement,
  archiveAnnouncement,
  deleteAnnouncement,
} from "@/services/announcements";
import { listClasses } from "@/services/academics";
import type {
  Announcement,
  AnnouncementStatus,
  AnnouncementTargetType,
  AnnouncementPriority,
} from "@/types/announcements";
import {
  ANNOUNCEMENT_PRIORITIES,
  ANNOUNCEMENT_STATUSES,
  ANNOUNCEMENT_TARGET_ROLES,
} from "@/types/announcements";
import type { Class } from "@/types/academics";

const PAGE_SIZE = 10;

const emptyForm = {
  title: "",
  content: "",
  target_type: "ALL" as AnnouncementTargetType,
  target_role: "PARENT",
  target_class_id: "",
  priority: "NORMAL" as AnnouncementPriority,
  expires_at: "",
};

function statusBadge(status: string) {
  const styles: Record<string, string> = {
    DRAFT: "bg-slate-100 text-slate-600 ring-slate-500/20",
    PUBLISHED: "bg-emerald-50 text-emerald-700 ring-emerald-600/20",
    ARCHIVED: "bg-amber-50 text-amber-700 ring-amber-600/20",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ring-1 ring-inset ${
        styles[status] || "bg-slate-100 text-slate-600 ring-slate-500/20"
      }`}
    >
      {status}
    </span>
  );
}

function priorityBadge(priority: string) {
  const styles: Record<string, string> = {
    LOW: "bg-slate-50 text-slate-500 ring-slate-400/20",
    NORMAL: "bg-sky-50 text-sky-700 ring-sky-600/20",
    HIGH: "bg-orange-50 text-orange-700 ring-orange-600/20",
    URGENT: "bg-rose-50 text-rose-700 ring-rose-600/20",
  };
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-semibold ring-1 ring-inset ${
        styles[priority] || "bg-slate-100 text-slate-600"
      }`}
    >
      {priority}
    </span>
  );
}

function targetBadge(a: Announcement) {
  if (a.target_type === "ALL") return <span className="text-xs font-medium text-slate-600">Everyone</span>;
  if (a.target_type === "ROLE")
    return (
      <span className="inline-flex items-center rounded-full bg-indigo-50 px-2.5 py-0.5 text-xs font-semibold text-indigo-700 ring-1 ring-inset ring-indigo-600/20">
        {a.target_role}
      </span>
    );
  return (
    <span className="inline-flex items-center rounded-full bg-purple-50 px-2.5 py-0.5 text-xs font-semibold text-purple-700 ring-1 ring-inset ring-purple-600/20">
      {a.target_class_name || "Class"}
    </span>
  );
}

function fmtDate(iso?: string | null) {
  if (!iso) return "—";
  try {
    return new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
  } catch {
    return iso;
  }
}

type ComposerMode = { open: false } | { open: true; edit?: Announcement };

export default function AnnouncementsPage() {
  const { user, hasPermission } = useAuth();

  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState("");
  const [targetTypeFilter, setTargetTypeFilter] = useState("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [listError, setListError] = useState<string | null>(null);

  const [composer, setComposer] = useState<ComposerMode>({ open: false });
  const [form, setForm] = useState(emptyForm);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [classes, setClasses] = useState<Class[]>([]);
  const [confirmDelete, setConfirmDelete] = useState<Announcement | null>(null);
  const [actionBusy, setActionBusy] = useState<string | null>(null);
  const [banner, setBanner] = useState<string | null>(null);

  const canCreate = hasPermission?.("announcement.create") ?? true;
  const canUpdate = hasPermission?.("announcement.update") ?? true;

  const load = useCallback(async () => {
    setLoading(true);
    setListError(null);
    try {
      const res = await listAnnouncements({
        status: statusFilter || undefined,
        target_type: targetTypeFilter || undefined,
        search: search || undefined,
        page,
        page_size: PAGE_SIZE,
      });
      setAnnouncements(res.data);
      setTotal(res.meta.total);
    } catch (err) {
      setListError(
        err instanceof ApiError
          ? `${err.code}: ${err.message}`
          : "Failed to load announcements. Is the backend running?",
      );
    } finally {
      setLoading(false);
    }
  }, [statusFilter, targetTypeFilter, search, page]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (!composer.open || form.target_type !== "CLASS" || classes.length > 0) return;
    listClasses()
      .then(setClasses)
      .catch(() => {
        /* class dropdown just stays empty */
      });
  }, [composer.open, form.target_type, classes.length]);

  const openCreate = () => {
    setForm(emptyForm);
    setFormError(null);
    setComposer({ open: true });
  };

  const openEdit = (a: Announcement) => {
    setForm({
      title: a.title,
      content: a.content,
      target_type: a.target_type as AnnouncementTargetType,
      target_role: a.target_role || "PARENT",
      target_class_id: a.target_class_id || "",
      priority: a.priority as AnnouncementPriority,
      expires_at: a.expires_at ? a.expires_at.slice(0, 10) : "",
    });
    setFormError(null);
    setComposer({ open: true, edit: a });
  };

  const closeComposer = () => setComposer({ open: false });

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setFormError(null);
    setSubmitting(true);
    const payload = {
      title: form.title,
      content: form.content,
      target_type: form.target_type,
      target_role: form.target_type === "ROLE" ? form.target_role : undefined,
      target_class_id: form.target_type === "CLASS" ? form.target_class_id : undefined,
      priority: form.priority,
      ...(form.expires_at ? { expires_at: form.expires_at } : {}),
    };
    try {
      if (composer.open && composer.edit) {
        await updateAnnouncement(composer.edit.id, payload);
        setBanner("Announcement updated.");
      } else {
        await createAnnouncement({ ...payload, status: "DRAFT" });
        setBanner("Draft created. Publish it when it's ready.");
        setPage(1);
      }
      setComposer({ open: false });
      await load();
    } catch (err) {
      setFormError(
        err instanceof ApiError ? `${err.code}: ${err.message}` : "Something went wrong. Try again.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  const runAction = async (a: Announcement, action: "publish" | "archive" | "delete") => {
    setActionBusy(`${action}:${a.id}`);
    setBanner(null);
    try {
      if (action === "publish") {
        await publishAnnouncement(a.id);
        setBanner(
          a.target_type === "ALL"
            ? "Published — staff were notified by email (best effort)."
            : "Published to the target audience.",
        );
      } else if (action === "archive") {
        await archiveAnnouncement(a.id);
        setBanner("Announcement archived.");
      } else {
        await deleteAnnouncement(a.id);
        setBanner("Announcement deleted.");
      }
      await load();
    } catch (err) {
      setBanner(
        err instanceof ApiError ? `${err.code}: ${err.message}` : "Action failed. Try again.",
      );
    } finally {
      setActionBusy(null);
      setConfirmDelete(null);
    }
  };

  const stats = {
    published: announcements.filter((a) => a.status === "PUBLISHED").length +
      (statusFilter === "PUBLISHED" ? Math.max(0, total - announcements.length) : 0),
    drafts: announcements.filter((a) => a.status === "DRAFT").length +
      (statusFilter === "DRAFT" ? Math.max(0, total - announcements.length) : 0),
  };

  const columns: Column<Announcement>[] = [
    {
      key: "title",
      header: "Announcement",
      render: (a) => (
        <div className="max-w-md">
          <div className="flex items-center gap-2 font-semibold text-slate-900">
            {a.title}
            {a.priority !== "NORMAL" && priorityBadge(a.priority)}
          </div>
          <p className="mt-0.5 truncate text-xs text-slate-500">{a.content}</p>
        </div>
      ),
    },
    { key: "status", header: "Status", render: (a) => statusBadge(a.status) },
    { key: "target", header: "Audience", render: (a) => targetBadge(a) },
    {
      key: "author_name",
      header: "Author",
      hideBelow: "md",
      render: (a) => <span className="text-xs">{a.author_name || a.author_email}</span>,
    },
    {
      key: "published_at",
      header: "Published",
      hideBelow: "lg",
      render: (a) => <span className="text-xs">{fmtDate(a.published_at)}</span>,
    },
    {
      key: "expires_at",
      header: "Expires",
      hideBelow: "xl",
      render: (a) => <span className="text-xs">{fmtDate(a.expires_at)}</span>,
    },
    {
      key: "actions",
      header: "Actions",
      render: (a) =>
        canUpdate ? (
          <div className="flex flex-wrap items-center gap-1.5">
            <button
              onClick={() => openEdit(a)}
              className="rounded-md border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50"
            >
              Edit
            </button>
            {a.status === "DRAFT" && (
              <button
                onClick={() => runAction(a, "publish")}
                disabled={actionBusy !== null}
                className="rounded-md bg-brand-700 px-2 py-1 text-xs font-semibold text-white hover:bg-brand-600 disabled:opacity-50"
              >
                Publish
              </button>
            )}
            {a.status === "PUBLISHED" && (
              <button
                onClick={() => runAction(a, "archive")}
                disabled={actionBusy !== null}
                className="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 hover:bg-amber-100 disabled:opacity-50"
              >
                Archive
              </button>
            )}
            <button
              onClick={() => setConfirmDelete(a)}
              disabled={actionBusy !== null}
              className="rounded-md border border-red-200 bg-red-50 px-2 py-1 text-xs font-medium text-red-700 hover:bg-red-100 disabled:opacity-50"
            >
              Delete
            </button>
          </div>
        ) : (
          <span className="text-xs text-slate-400">—</span>
        ),
    },
  ];

  return (
    <ProtectedRoute requiredRole="ADMIN">
      <AppShell
        title="Announcements"
        actions={
          canCreate && (
            <button
              onClick={openCreate}
              className="rounded-lg bg-brand-700 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-brand-600"
            >
              + New Announcement
            </button>
          )
        }
      >
        <div className="space-y-6">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">Announcements</h1>
            <p className="mt-1 text-sm text-slate-500">
              Compose school-wide broadcasts and targeted messages. Publishing a school-wide
              announcement also emails staff (best effort).
            </p>
          </div>

          {banner && (
            <div className="flex items-center justify-between rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
              <span>{banner}</span>
              <button onClick={() => setBanner(null)} className="text-xs font-semibold underline">
                Dismiss
              </button>
            </div>
          )}

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
            <StatCard label="In view" value={total} />
            <StatCard label="Published on page" value={stats.published} />
            <StatCard label="Drafts on page" value={stats.drafts} />
          </div>

          <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <SearchInput
              placeholder="Search title or content…"
              onSearch={(q) => {
                setSearch(q);
                setPage(1);
              }}
              className="sm:max-w-xs"
            />
            <div className="w-40">
              <SelectField
                id="status-filter"
                label="Status"
                value={statusFilter}
                onChange={(e) => {
                  setStatusFilter(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">All statuses</option>
                {ANNOUNCEMENT_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </SelectField>
            </div>
            <div className="w-44">
              <SelectField
                id="target-type-filter"
                label="Audience"
                value={targetTypeFilter}
                onChange={(e) => {
                  setTargetTypeFilter(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">All audiences</option>
                <option value="ALL">Everyone</option>
                <option value="ROLE">Role-targeted</option>
                <option value="CLASS">Class-targeted</option>
              </SelectField>
            </div>
          </div>

          {listError && (
            <div className="flex items-center justify-between rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
              <span>{listError}</span>
              <button onClick={load} className="text-xs font-semibold underline">
                Try Again
              </button>
            </div>
          )}

          <DataTable
            columns={columns}
            rows={announcements}
            getRowKey={(a) => a.id}
            emptyMessage={loading ? "Loading…" : "No announcements yet. Create the first one!"}
          />

          <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />

          {/* Composer modal */}
          <Modal
            open={composer.open}
            onClose={closeComposer}
            title={composer.open && composer.edit ? "Edit Announcement" : "New Announcement"}
            maxWidth="max-w-2xl"
            footer={
              <FormActions
                error={formError}
                submitting={submitting}
                label={composer.open && composer.edit ? "Save Changes" : "Create Draft"}
                onCancel={closeComposer}
              />
            }
          >
            <form onSubmit={handleSubmit} className="space-y-4" id="announcement-form">
              <TextField
                id="ann-title"
                label="Title"
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                required
                placeholder="e.g. Midterm examination timetable released"
              />
              <TextAreaField
                id="ann-content"
                label="Content"
                value={form.content}
                onChange={(e) => setForm({ ...form, content: e.target.value })}
                required
                rows={6}
                placeholder="Write the announcement body…"
              />
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <SelectField
                  id="ann-target-type"
                  label="Audience"
                  value={form.target_type}
                  onChange={(e) =>
                    setForm({ ...form, target_type: e.target.value as AnnouncementTargetType })
                  }
                >
                  <option value="ALL">Everyone</option>
                  <option value="ROLE">A single role</option>
                  <option value="CLASS">A class cohort</option>
                </SelectField>
                {form.target_type === "ROLE" && (
                  <SelectField
                    id="ann-target-role"
                    label="Role"
                    value={form.target_role}
                    onChange={(e) => setForm({ ...form, target_role: e.target.value })}
                  >
                    {ANNOUNCEMENT_TARGET_ROLES.map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </SelectField>
                )}
                {form.target_type === "CLASS" && (
                  <SelectField
                    id="ann-target-class"
                    label="Class"
                    value={form.target_class_id}
                    onChange={(e) => setForm({ ...form, target_class_id: e.target.value })}
                    required
                  >
                    <option value="">Select class…</option>
                    {classes.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </SelectField>
                )}
                <SelectField
                  id="ann-priority"
                  label="Priority"
                  value={form.priority}
                  onChange={(e) => setForm({ ...form, priority: e.target.value as AnnouncementPriority })}
                >
                  {ANNOUNCEMENT_PRIORITIES.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </SelectField>
              </div>
              <TextField
                id="ann-expires"
                label="Expires on (optional)"
                type="date"
                value={form.expires_at}
                onChange={(e) => setForm({ ...form, expires_at: e.target.value })}
              />
              <p className="text-xs text-slate-400">
                {composer.open && composer.edit
                  ? "Changes apply to the existing announcement."
                  : "Created as a DRAFT — use Publish in the table when it's ready. Publishing school-wide sends staff an email."}
              </p>
            </form>
          </Modal>

          {/* Delete confirmation */}
          <Modal
            open={confirmDelete !== null}
            onClose={() => setConfirmDelete(null)}
            title="Delete announcement?"
            footer={
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setConfirmDelete(null)}
                  className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => confirmDelete && runAction(confirmDelete, "delete")}
                  disabled={actionBusy !== null}
                  className="rounded-lg bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-500 disabled:opacity-50"
                >
                  {actionBusy ? "Deleting…" : "Delete"}
                </button>
              </div>
            }
          >
            {confirmDelete && (
              <p className="text-sm text-slate-600">
                Permanently delete <span className="font-semibold">{confirmDelete.title}</span>? This
                cannot be undone.
              </p>
            )}
          </Modal>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
