// Package audit records administrative/security actions (section 29).
// M6 ships the write path only: domain services insert an audit row inside
// the same transaction as the mutation they describe, so an action either
// fully happens with its audit entry or doesn't happen at all ("we did it
// but can't prove who did it" is the worse failure for school records).
// The authorized read/report endpoints (audit.view) are milestone M19.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/thunderlight247/schoolhub/backend/db/generated"
)

// Action codes from section 29. Each milestone adds the codes its own
// mutations introduce; this set grows as domains land.
const (
	ActionDepartmentCreated  = "DEPARTMENT_CREATED"
	ActionPositionCreated    = "POSITION_CREATED"
	ActionStaffCreated       = "STAFF_CREATED"
	ActionStaffUpdated       = "STAFF_UPDATED"
	ActionStaffStatusChanged = "STAFF_STATUS_CHANGED"
	ActionUserCreated        = "USER_CREATED"
	ActionUserRolesUpdated   = "USER_ROLES_UPDATED"

	// M7 — students and guardians.
	ActionStudentCreated       = "STUDENT_CREATED"
	ActionStudentUpdated       = "STUDENT_UPDATED"
	ActionStudentStatusChanged = "STUDENT_STATUS_CHANGED"
	ActionGuardianCreated      = "GUARDIAN_CREATED"
	ActionGuardianUpdated      = "GUARDIAN_UPDATED"
	ActionGuardianLinked       = "STUDENT_GUARDIAN_LINKED"
	ActionGuardianUnlinked     = "STUDENT_GUARDIAN_UNLINKED"

	// M8 — academic structure.
	ActionAcademicYearCreated    = "ACADEMIC_YEAR_CREATED"
	ActionAcademicYearUpdated    = "ACADEMIC_YEAR_UPDATED"
	ActionAcademicYearSetCurrent = "ACADEMIC_YEAR_SET_CURRENT"
	ActionTermCreated            = "TERM_CREATED"
	ActionTermUpdated            = "TERM_UPDATED"
	ActionTermSetCurrent         = "TERM_SET_CURRENT"
	ActionProgrammeCreated       = "PROGRAMME_CREATED"
	ActionProgrammeUpdated       = "PROGRAMME_UPDATED"
	ActionSubjectCreated         = "SUBJECT_CREATED"
	ActionSubjectUpdated         = "SUBJECT_UPDATED"
	ActionClassCreated           = "CLASS_CREATED"
	ActionClassUpdated           = "CLASS_UPDATED"
	ActionClassSubjectAssigned   = "CLASS_SUBJECT_ASSIGNED"
	ActionClassSubjectRemoved    = "CLASS_SUBJECT_REMOVED"
)

// Resource types referenced by audit entries.
const (
	ResourceStaff         = "staff"
	ResourceUser          = "user"
	ResourceDepartment    = "department"
	ResourceStaffPosition = "staff_position"

	// M7.
	ResourceStudent  = "student"
	ResourceGuardian = "guardian"

	// M8.
	ResourceAcademicYear = "academic_year"
	ResourceTerm         = "term"
	ResourceProgramme    = "programme"
	ResourceSubject      = "subject"
	ResourceClass        = "class"
)

// Entry describes one auditable action. ActorUserID/ResourceID may be the
// zero uuid.UUID, which Record maps to SQL NULL (system actions have no
// actor; some actions target no single row).
type Entry struct {
	ActorUserID  uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	Metadata     map[string]any
}

// Record writes one audit row through q. Callers pass the tx-bound Querier
// so the entry commits atomically with the mutation it describes. A nil
// Metadata map is stored as {} rather than JSON null.
func Record(ctx context.Context, q db.Querier, e Entry) error {
	metadata := []byte("{}")
	if e.Metadata != nil {
		var err error
		metadata, err = json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("audit: marshaling metadata: %w", err)
		}
	}

	return q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		ActorUserID:  toPgUUID(e.ActorUserID),
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   toPgUUID(e.ResourceID),
		Metadata:     metadata,
	})
}

// toPgUUID maps the zero UUID to an invalid (NULL) pgtype.UUID and any
// real UUID to a valid one.
func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: id != uuid.Nil}
}
