"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { ProtectedRoute } from "@/components/protected-route";
import { DataTable, type Column } from "@/components/ui/table";
import { Pagination } from "@/components/ui/pagination";
import { StatCard } from "@/components/ui/stat-card";
import { Modal } from "@/components/ui/modal";
import { TextField } from "@/components/ui/form";
import { listAuditLogs, listAuditActions } from "@/services/audit";
import type { AuditEntry, AuditListParams } from "@/types/audit";

const PAGE_SIZE = 20;

// Resource types actually stored in audit_logs (lowercase, queried live).
// The dynamic /actions endpoint covers action codes, whose set grows every
// milestone, but there is no resource-types endpoint — this static list
// mirrors the values Record writes today.
const RESOURCE_TYPES = [
  "academic_year",
  "announcement",
  "assessment",
  "attendance_record",
  "attendance_session",
  "class",
  "department",
  "guardian",
  "programme",
  "staff",
  "staff_position",
  "student",
  "subject",
  "teacher_assignment",
  "term",
  "user",
] as const;

function fmtTime(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function actorName(e: AuditEntry): string {
  const name = [e.actor_first_name, e.actor_last_name].filter(Boolean).join(" ");
  return name || e.actor_email || "System";
}

// Action chip tone: colour by broad category so the log is scannable.
function toneForAction(action: string): string {
  if (action.includes("DELETED")) return "bg-rose-50 text-rose-700 border border-rose-200";
  if (action.includes("CREATED")) return "bg-emerald-50 text-emerald-700 border border-emerald-200";
  if (action.includes("ARCHIVED") || action.includes("SUSPENDED") || action.includes("BLOCK"))
    return "bg-amber-50 text-amber-700 border border-amber-200";
  if (action.includes("PUBLISHED") || action.includes("SET_CURRENT") || action.includes("ACTIVATED"))
    return "bg-sky-50 text-sky-700 border border-sky-200";
  return "bg-slate-100 text-slate-600 border border-slate-200";
}

export default function AuditPage() {
  // Loading/empty/error state shown in the table area (M33 keeps these).
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters come from the dropdowns/date inputs; a refless "filterKey"
  // string triggers the fetch effect so load() stays a plain function.
  const [actions, setActions] = useState<string[]>([]);
  const [actionFilter, setActionFilter] = useState("");
  const [resourceFilter, setResourceFilter] = useState("");
  const [dateFrom, setDateFrom] = useState("");
  const [dateTo, setDateTo] = useState("");
  const [requestSeq, setRequestSeq] = useState(0);

  const [detail, setDetail] = useState<{ open: boolean; entry: AuditEntry | null }>({
    open: false,
    entry: null,
  });

  const fetchPage = useCallback(
    async (pageToLoad: number) => {
      setLoading(true);
      setError(null);
      try {
        const params: AuditListParams = {
          page: pageToLoad,
          page_size: PAGE_SIZE,
        };
        if (actionFilter) params.action = actionFilter;
        if (resourceFilter) params.resource_type = resourceFilter;
        // Date inputs give YYYY-MM-DD; the backend needs RFC3339 timestamps.
        if (dateFrom) params.date_from = `${dateFrom}T00:00:00Z`;
        if (dateTo) params.date_to = `${dateTo}T23:59:59Z`;
        const res = await listAuditLogs(params);
        setEntries(res.audit_logs ?? []);
        setTotal(res.pagination?.total ?? 0);
        setPage(pageToLoad);
      } catch (e) {
        if (e instanceof Error && e.message.includes("404")) {
          setError("Audit log entry not found.");
        } else {
          setError(e instanceof Error ? e.message : "Failed to load audit logs.");
        }
      } finally {
        setLoading(false);
      }
    },
    [actionFilter, resourceFilter, dateFrom, dateTo],
  );

  // Re-fetch from page 1 whenever a filter changes (requestSeq tracks the
  // filter tuple so the effect forwards the *changed* filters' fetch).
  const filterKey = `${actionFilter}|${resourceFilter}|${dateFrom}|${dateTo}`;
  const filterKeyRef = useRef(filterKey);
  useEffect(() => {
    if (filterKeyRef.current !== filterKey) {
      filterKeyRef.current = filterKey;
    }
    fetchPage(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterKey, requestSeq]);

  useEffect(() => {
    listAuditActions()
      .then(setActions)
      .catch(() => setActions([]));
  }, []);

  // Initial load happens via the filterKey effect above (first mount runs it).

  const clearFilters = () => {
    setActionFilter("");
    setResourceFilter("");
    setDateFrom("");
    setDateTo("");
    setRequestSeq((s) => s + 1); // force a refetch even if key was already empty
  };

  const hasFilters = !!(actionFilter || resourceFilter || dateFrom || dateTo);

  const columns: Column<AuditEntry>[] = [
    {
      key: "created_at",
      header: "When",
      render: (e) => (
        <span className="whitespace-nowrap text-slate-600">{fmtTime(e.created_at)}</span>
      ),
    },
    {
      key: "actor",
      header: "By",
      render: (e) => (
        <div>
          <div className="font-medium text-slate-800">{actorName(e)}</div>
          {e.actor_email && <div className="text-xs text-slate-400">{e.actor_email}</div>}
        </div>
      ),
    },
    {
      key: "action",
      header: "Action",
      render: (e) => (
        <span
          className={`whitespace-nowrap rounded px-2 py-0.5 text-xs font-semibold ${toneForAction(e.action)}`}
        >
          {e.action}
        </span>
      ),
    },
    {
      key: "resource",
      header: "Resource",
      render: (e) => (
        <div>
          <div className="font-semibold text-slate-700">{e.resource_type}</div>
          {e.resource_id && (
            <div className="font-mono text-[11px] text-slate-400">
              {e.resource_id.slice(0, 8)}…
            </div>
          )}
        </div>
      ),
      hideBelow: "sm",
    },
    {
      key: "detail",
      header: "",
      render: (e) => (
        <button
          onClick={() => setDetail({ open: true, entry: e })}
          className="rounded border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-600 hover:bg-slate-50"
        >
          View
        </button>
      ),
    },
  ];

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <ProtectedRoute requiredPermission="audit.view">
      <AppShell
        title="Audit Log"
        actions={
          <button
            onClick={() => setRequestSeq((s) => s + 1)}
            disabled={loading}
            className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm hover:bg-slate-50 disabled:opacity-50"
          >
            {loading ? "Refreshing…" : "Refresh"}
          </button>
        }
      >
        <div className="space-y-6">
          {/* Banner */}
          <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="mb-2 inline-flex items-center gap-2">
              <span className="rounded border border-brand-100 bg-brand-50 px-2 py-0.5 text-xs font-bold uppercase tracking-wider text-brand-700">
                Accountability
              </span>
            </div>
            <h1 className="text-2xl font-bold tracking-tight text-slate-900">Audit Log</h1>
            <p className="mt-1 text-sm text-slate-500">
              Every mutation since launch — who did what, and when. Append-only; entries here
              cannot be edited or deleted.
            </p>
          </div>

          {/* StatCards */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatCard label="Total entries" value={total} tone="purple" />
            <StatCard label="Distinct actions" value={actions.length} tone="sky" />
            <StatCard label="Page" value={`${page} / ${totalPages}`} tone="green" />
          </div>

          {/* Filters */}
          <div className="space-y-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Action
                </label>
                <select
                  value={actionFilter}
                  onChange={(e) => setActionFilter(e.target.value)}
                  className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                >
                  <option value="">All actions ({actions.length})</option>
                  {actions.map((a) => (
                    <option key={a} value={a}>
                      {a}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wide text-slate-500">
                  Resource type
                </label>
                <select
                  value={resourceFilter}
                  onChange={(e) => setResourceFilter(e.target.value)}
                  className="mt-1 block w-full rounded-md border border-slate-200 px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-brand-500"
                >
                  <option value="">All resource types</option>
                  {RESOURCE_TYPES.map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-4">
              <TextField
                label="From date"
                type="date"
                value={dateFrom}
                onChange={(e) => setDateFrom(e.target.value)}
              />
              <TextField
                label="To date"
                type="date"
                value={dateTo}
                onChange={(e) => setDateTo(e.target.value)}
              />
              {hasFilters && (
                <div className="flex items-end">
                  <button
                    onClick={clearFilters}
                    className="rounded-md border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-600 hover:bg-slate-50"
                  >
                    Clear filters
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Error / table */}
          {error && (
            <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-800">
              {error}
            </div>
          )}

          {loading ? (
            <div className="rounded-xl border border-slate-200 bg-white p-10 text-center text-sm text-slate-400 shadow-sm">
              Loading audit log…
            </div>
          ) : (
            <>
              <DataTable<AuditEntry>
                columns={columns}
                rows={entries}
                getRowKey={(e) => e.id}
                emptyMessage="No audit log entries match the current filters."
              />
              <Pagination
                page={page}
                pageSize={PAGE_SIZE}
                totalItems={total}
                onPageChange={(p) => void fetchPage(p)}
              />
            </>
          )}

          {/* Detail modal */}
          <Modal
            open={detail.open}
            onClose={() => setDetail({ open: false, entry: null })}
            title="Audit entry detail"
          >
            {detail.entry && (
              <div className="space-y-4 text-sm">
                <dl className="space-y-2">
                  {(
                    [
                      ["Action", detail.entry.action],
                      ["When", fmtTime(detail.entry.created_at)],
                      [
                        "By",
                        `${actorName(detail.entry)}${detail.entry.actor_email ? ` (${detail.entry.actor_email})` : ""}`,
                      ],
                      ["Resource type", detail.entry.resource_type],
                      ["Resource ID", detail.entry.resource_id ?? "—"],
                    ] as const
                  ).map(([label, value]) => (
                    <div key={label} className="grid grid-cols-3 gap-2">
                      <dt className="text-xs font-medium uppercase text-slate-400">{label}</dt>
                      <dd className="col-span-2 break-words text-slate-800">{value}</dd>
                    </div>
                  ))}
                </dl>
                <div>
                  <p className="mb-1 text-xs font-medium uppercase text-slate-400">Metadata</p>
                  <pre className="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-slate-50 p-3 font-mono text-[11px] text-slate-700">
                    {JSON.stringify(detail.entry.metadata, null, 2)}
                  </pre>
                </div>
                <div className="flex justify-end pt-2">
                  <button
                    type="button"
                    onClick={() => setDetail({ open: false, entry: null })}
                    className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 shadow-sm hover:bg-slate-50"
                  >
                    Close
                  </button>
                </div>
              </div>
            )}
          </Modal>
        </div>
      </AppShell>
    </ProtectedRoute>
  );
}
