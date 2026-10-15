// Read-side of the audit package (milestone M19). Record (audit.go) is the
// write side every other domain calls from inside its own transaction;
// everything below is new for M19 and only ever reads the table.
package audit

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// ErrAuditLogNotFound is returned by Get when no entry matches the id.
var ErrAuditLogNotFound = errors.New("audit: entry not found")

// Service provides read access to the audit log for section 29's
// "authorized admin audit review". Unlike Record, which any domain calls
// through the tx-bound db.Querier interface, Service owns its own pool —
// reads here never need to participate in another package's transaction.
type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, queries: db.New(pool)}
}

// ListFilter is the backend-side filter/pagination request (section 25).
// Every field is optional; an absent filter is not applied.
type ListFilter struct {
	ActorUserID  *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	PageSize     int
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// NormalizePagination clamps client-provided pagination to sane bounds —
// duplicated from the other domain packages' identical helper rather than
// shared; see internal/staff's comment on that per-package convention.
func NormalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}

func toPgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func filterText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toPgTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// List returns one page of the filtered audit log plus the total matching
// count. Both queries apply the identical WHERE clause — see the sync
// note in audit_logs.sql.
func (s *Service) List(ctx context.Context, f ListFilter) ([]db.ListAuditLogsRow, int64, error) {
	page, pageSize := NormalizePagination(f.Page, f.PageSize)

	listParams := db.ListAuditLogsParams{
		ActorUserID:  toPgUUIDPtr(f.ActorUserID),
		Action:       filterText(f.Action),
		ResourceType: filterText(f.ResourceType),
		ResourceID:   toPgUUIDPtr(f.ResourceID),
		DateFrom:     toPgTimestamptzPtr(f.DateFrom),
		DateTo:       toPgTimestamptzPtr(f.DateTo),
		PageOffset:   int32((page - 1) * pageSize),
		PageLimit:    int32(pageSize),
	}
	countParams := db.CountAuditLogsParams{
		ActorUserID:  listParams.ActorUserID,
		Action:       listParams.Action,
		ResourceType: listParams.ResourceType,
		ResourceID:   listParams.ResourceID,
		DateFrom:     listParams.DateFrom,
		DateTo:       listParams.DateTo,
	}

	total, err := s.queries.CountAuditLogs(ctx, countParams)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.queries.ListAuditLogs(ctx, listParams)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// Get returns a single audit log entry.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.GetAuditLogByIDRow, error) {
	row, err := s.queries.GetAuditLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetAuditLogByIDRow{}, ErrAuditLogNotFound
		}
		return db.GetAuditLogByIDRow{}, err
	}
	return row, nil
}

// ListActions returns the distinct action codes actually present in the
// log, for a frontend filter dropdown.
func (s *Service) ListActions(ctx context.Context) ([]string, error) {
	return s.queries.ListDistinctAuditActions(ctx)
}
