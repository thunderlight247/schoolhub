"use client";

// DataTable (M23, upgraded in M25): the reusable table every list screen
// renders through. Columns are declarative — a typed key, a header label,
// and an optional render function — so screens describe WHAT to show, not
// how to draw a <table>.
//
// M25 polish (professional bar, patterns from the dashboard reference):
//  - zebra striping and row hover for scanability
//  - per-column responsive hiding (`hideBelow`) so wide tables stay usable
//    on mobile instead of overflowing
//  - rounded card frame with header row separated by weight, not color

export type HideBelow = "sm" | "md" | "lg" | "xl";

const hideClasses: Record<HideBelow, string> = {
  sm: "hidden sm:table-cell",
  md: "hidden md:table-cell",
  lg: "hidden lg:table-cell",
  xl: "hidden xl:table-cell",
};

export interface Column<T> {
  key: string;
  header: string;
  /** Render the cell for this row. Defaults to String(row[key]). */
  render?: (row: T) => React.ReactNode;
  /** Optional extra classes for the <td>. */
  className?: string;
  /** Hide this column below the given breakpoint (reference-repo pattern). */
  hideBelow?: HideBelow;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  rows: T[];
  /** Stable row key. */
  getRowKey: (row: T) => string;
  emptyMessage?: string;
}

export function DataTable<T>({
  columns,
  rows,
  getRowKey,
  emptyMessage = "No records found.",
}: DataTableProps<T>) {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-slate-200 text-sm">
          <thead className="bg-slate-50">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.key}
                  scope="col"
                  className={`px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-500 ${
                    col.hideBelow ? hideClasses[col.hideBelow] : ""
                  }`}
                >
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {rows.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-12 text-center text-sm text-slate-400">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              rows.map((row, i) => (
                <tr
                  key={getRowKey(row)}
                  className={i % 2 === 1 ? "bg-slate-50/60 hover:bg-slate-50" : "hover:bg-slate-50"}
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      className={`px-4 py-3 text-slate-700 ${col.hideBelow ? hideClasses[col.hideBelow] : ""} ${col.className ?? ""}`}
                    >
                      {col.render ? col.render(row) : String((row as Record<string, unknown>)[col.key] ?? "")}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
