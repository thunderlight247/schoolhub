// Command demo-seed populates the database with a realistic demo dataset
// (milestone M22): staff with accounts and department/position structure,
// a full class roster per class, guardian links, teacher assignments,
// several weeks of attendance, assessments with results, and announcements.
//
// It is IDEMPOTENT — every insert uses fixed UUIDs or the table's natural
// unique key with ON CONFLICT DO NOTHING, so it is safe to re-run and it
// will never overwrite data created through the API. Existing hand-seeded
// rows (the 11111111/22222222/55555555 fixed-UUID skeleton) are recognized
// and extended, not duplicated.
//
// It writes directly via SQL (not through the service layer) on purpose:
// the goal is demo data, not replaying business flows, and skipping the
// service layer avoids flooding the audit log with synthetic entries.
// Run it as the schema-owning role (same DATABASE_URL the migrate tool
// uses), not the least-privilege schoolhub_app role — not because it needs
// DDL, but because a seeder is an operator tool like migrate, not the app.
//
// Usage (from backend/, with DATABASE_URL set, e.g. the local dev stack on
// port 5433 per remember.md):
//
//	DATABASE_URL='postgres://schoolhub:schoolhub@localhost:5433/schoolhub?sslmode=disable' go run ./cmd/demo-seed
//
// Former demo login accounts (admin@/ada@/teacher@/parent@/student@
// schoolhub.test, shared "ChangeMe123!") were REMOVED from seeding in M34 —
// the database no longer carries shared demo credentials. Structure data
// (people/staff/students/classes/enrollments) is still seeded here.
//
//	admin@schoolhub.test    ADMIN    (historical — account removed in M34)
//
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/thunderlight247/schoolhub/backend/config"
	"github.com/thunderlight247/schoolhub/backend/internal/database"
)

// Fixed UUID prefixes keep demo rows recognizable in psql and make re-runs
// land on ON CONFLICT DO NOTHING instead of duplicating.
const (
	personPrefix     = "aaaa0000-0000-0000-0000-%012d"
	studentPrefix    = "bbbb0000-0000-0000-0000-%012d"
	enrollmentPrefix = "dddd0000-0000-0000-0000-%012d"
	assignPrefix     = "ffff0000-0000-0000-0000-%012d"
	classSubjPrefix  = "66666666-0000-0000-0000-%012d"
	assessmentPrefix = "eeee0000-0000-0000-0000-%012d"
	announcePrefix   = "77777777-0000-0000-0000-%012d"
	guardianLink1    = "dddd0001-0000-0000-0000-000000000001" // existing: parent ↔ Ama
	guardianLink2    = "dddd0002-0000-0000-0000-000000000002" // parent ↔ Kofi
	form1BID         = "55555555-5555-5555-5555-555555555556"
)

// Announcements need an author with an account; the master admin is the
// author in every environment. Resolved at runtime as the OLDEST MASTER_ADMIN
// user (ORDER BY created_at — min(uuid) is not a valid SQL aggregate), so a
// fresh deployment seeds cleanly as long as the bootstrap admin was created
// first (cmd/seed with -role ADMIN grants MASTER_ADMIN since the bootstrap
// invariant fix). The old hardcoded personal email broke fresh databases.

var (
	academicYearID = uuid.MustParse("11111111-1111-1111-1111-111111111111") // 2026/2027, current
	term1ID        = uuid.MustParse("22222222-2222-2222-2222-222222222222") // Term 1, current
	programmeID    = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	form1AID       = uuid.MustParse("55555555-5555-5555-5555-555555555555")
	amaID          = uuid.MustParse("bbbb0001-0000-0000-0000-000000000001")
	kofiID         = uuid.MustParse("bbbb0002-0000-0000-0000-000000000002")
)

func main() {
	// Generous deadline: every INSERT is a network round-trip, and a remote
	// demo deployment (e.g. Neon) turns a ~1s local seed into several
	// minutes for the full dataset. An operator tool failing at 4 minutes
	// in was worse than just waiting.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	if err := seed(ctx, pool); err != nil {
		log.Fatalf("demo-seed: %v", err)
	}
	log.Println("demo-seed: done (idempotent — safe to re-run)")
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	rng := rand.New(rand.NewSource(42)) // deterministic demo data

	var err error

	// --- Reference data: departments & positions ---------------------------
	for _, name := range []string{"Mathematics", "Science", "Languages", "Humanities"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO departments (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, name); err != nil {
			return fmt.Errorf("department %s: %w", name, err)
		}
	}

	posIDs := map[string]uuid.UUID{}
	for _, p := range [][2]string{
		{"Mathematics Teacher", "TEACHING"},
		{"Science Teacher", "TEACHING"},
		{"English Teacher", "TEACHING"},
		{"Principal", "ADMINISTRATION"},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO staff_positions (title, category) VALUES ($1, $2) ON CONFLICT (title) DO NOTHING`,
			p[0], p[1]); err != nil {
			return fmt.Errorf("position %s: %w", p[0], err)
		}
		var id uuid.UUID
		if err := pool.QueryRow(ctx,
			`SELECT id FROM staff_positions WHERE title = $1`, p[0]).Scan(&id); err != nil {
			return fmt.Errorf("position %s lookup: %w", p[0], err)
		}
		posIDs[p[0]] = id
	}

	// --- Staff: give Sarah Mensah a staff row -------------------------------
	// Resolution is by person now — M34 removed the @schoolhub.test login
	// accounts, but the underlying people rows persist. A fresh environment
	// (e.g. the Neon demo deployment) self-provisions her person row first,
	// exactly like Ada Obi below does.
	var sarahPersonID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT id FROM people WHERE first_name = 'Sarah' AND last_name = 'Mensah'`).Scan(&sarahPersonID)
	if err == pgx.ErrNoRows {
		sarahPersonID = uuid.MustParse(fmt.Sprintf(personPrefix, 901))
		if _, err := pool.Exec(ctx,
			`INSERT INTO people (id, first_name, last_name, gender, phone)
			 VALUES ($1, 'Sarah', 'Mensah', 'FEMALE', '+233-24-555-0103') ON CONFLICT (id) DO NOTHING`,
			sarahPersonID); err != nil {
			return fmt.Errorf("person Sarah Mensah: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("Sarah Mensah person lookup: %w", err)
	}
	var deptID uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT id FROM departments WHERE name = $1`, "Languages").Scan(&deptID); err != nil {
		return fmt.Errorf("department Languages lookup: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO staff (id, person_id, staff_number, department_id, position_id, employment_status, hired_date)
		 VALUES ($1, $2, $3, $4, $5, 'ACTIVE', '2023-08-14')
		 ON CONFLICT (person_id) DO NOTHING`,
		uuid.MustParse(fmt.Sprintf(assignPrefix, 900001)), sarahPersonID, "STF-002",
		deptID, posIDs["English Teacher"]); err != nil {
		return fmt.Errorf("staff Sarah Mensah: %w", err)
	}

	// Ada Obi is the second demo teacher (Mathematics & Science). M34 removed
	// the @schoolhub.test login accounts, so resolution goes through the staff
	// row; a fresh environment is seeded with the person + staff row directly
	// (the account layer is intentionally not touched by this seeder).
	var adaPersonID, adaStaffID uuid.UUID
	err = pool.QueryRow(ctx,
		`SELECT person_id, id FROM staff WHERE staff_number = 'STF-001'`).Scan(&adaPersonID, &adaStaffID)
	if err == pgx.ErrNoRows {
		adaPersonID = uuid.MustParse(fmt.Sprintf(personPrefix, 900))
		if _, err := pool.Exec(ctx,
			`INSERT INTO people (id, first_name, last_name, gender, phone)
			 VALUES ($1, 'Ada', 'Obi', 'FEMALE', '+233-24-555-0102') ON CONFLICT (id) DO NOTHING`,
			adaPersonID); err != nil {
			return fmt.Errorf("person Ada Obi: %w", err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO staff (id, person_id, staff_number, department_id, position_id, employment_status, hired_date)
			 VALUES ($1, $2, 'STF-001',
			         (SELECT id FROM departments WHERE name = 'Mathematics'),
			         (SELECT id FROM staff_positions WHERE title = 'Mathematics Teacher'),
			         'ACTIVE', '2021-01-11')
			 ON CONFLICT (person_id) DO NOTHING`,
			uuid.MustParse(fmt.Sprintf(assignPrefix, 900000)), adaPersonID); err != nil {
			return fmt.Errorf("staff Ada Obi: %w", err)
		}
		if err := pool.QueryRow(ctx,
			`SELECT id FROM staff WHERE person_id = $1`, adaPersonID).Scan(&adaStaffID); err != nil {
			return fmt.Errorf("ada staff lookup: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("STF-001 lookup: %w", err)
	}
	// An actively-teaching demo teacher should not show as on leave
	// (earlier hand-seeded data had her ON_LEAVE).
	if _, err := pool.Exec(ctx,
		`UPDATE staff SET employment_status = 'ACTIVE' WHERE id = $1 AND employment_status = 'ON_LEAVE'`,
		adaStaffID); err != nil {
		return fmt.Errorf("staff status fixup: %w", err)
	}
	var sarahStaffID uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT id FROM staff WHERE person_id = $1`, sarahPersonID).Scan(&sarahStaffID); err != nil {
		return fmt.Errorf("Sarah staff lookup: %w", err)
	}

	// A few more staff members so the M24 staff directory has a realistic
	// list (departments, categories, and account-yes/no variety).
	extraStaff := []struct {
		num         int64
		first, last string
		gender      string
		staffNo     string
		dept        string
		position    string
		hired       string
	}{
		{901000, "Grace", "Amponsah", "FEMALE", "STF-003", "Mathematics", "Mathematics Teacher", "2024-09-02"},
		{901001, "Yaw", "Boateng", "MALE", "STF-004", "Humanities", "Principal", "2019-01-14"},
		{901002, "Efua", "Tetteh", "FEMALE", "STF-005", "Science", "Science Teacher", "2025-08-18"},
	}
	for _, es := range extraStaff {
		pid := uuid.MustParse(fmt.Sprintf(personPrefix, es.num))
		if _, err := pool.Exec(ctx,
			`INSERT INTO people (id, first_name, last_name, gender, phone)
			 VALUES ($1, $2, $3, $4, '+233-24-555-01' || lpad(($5 % 10)::text, 1, '0'))
			 ON CONFLICT (id) DO NOTHING`,
			pid, es.first, es.last, es.gender, es.num); err != nil {
			return fmt.Errorf("person %s %s: %w", es.first, es.last, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO staff (id, person_id, staff_number, department_id, position_id, employment_status, hired_date)
			 VALUES ($1, $2, $3,
			         (SELECT id FROM departments WHERE name = $4),
			         (SELECT id FROM staff_positions WHERE title = $5),
			         'ACTIVE', $6)
			 ON CONFLICT (person_id) DO NOTHING`,
			uuid.MustParse(fmt.Sprintf(assignPrefix, es.num)), pid, es.staffNo,
			es.dept, es.position, es.hired); err != nil {
			return fmt.Errorf("staff %s %s: %w", es.first, es.last, err)
		}
	}

	// --- Subjects & classes ------------------------------------------------
	type subject struct{ code, name string }
	subjects := []subject{
		{"MATH", "Mathematics"},
		{"ENG", "English Language"},
		{"SCI", "Integrated Science"},
		{"SOC", "Social Studies"},
		{"ICT", "Computing"},
		{"FRE", "French"},
	}
	subjIDs := map[string]uuid.UUID{}
	for _, s := range subjects {
		if _, err := pool.Exec(ctx,
			`INSERT INTO subjects (code, name, type) VALUES ($1, $2, 'CORE') ON CONFLICT (code) DO NOTHING`,
			s.code, s.name); err != nil {
			return fmt.Errorf("subject %s: %w", s.code, err)
		}
		var id uuid.UUID
		if err := pool.QueryRow(ctx,
			`SELECT id FROM subjects WHERE code = $1`, s.code).Scan(&id); err != nil {
			return fmt.Errorf("subject %s lookup: %w", s.code, err)
		}
		subjIDs[s.code] = id
	}

	// --- Academic skeleton (fresh-database bootstrap) -----------------------
	// The dev database long ago got its academic year / term / programme /
	// Form 1A class rows by hand; a fresh deployment (e.g. the Neon demo)
	// has none of them, and everything downstream (classes, enrollments,
	// attendance, assessments, the term window for announcements) keys off
	// these fixed UUIDs. Self-provision them here: every insert is
	// idempotent (ON CONFLICT DO NOTHING), so on the dev DB this is a
	// no-op and existing dates/values are preserved.
	if _, err := pool.Exec(ctx,
		`INSERT INTO academic_years (id, name, start_date, end_date, is_current)
		 VALUES ($1, '2026/2027', '2026-09-08', '2027-07-30', true)
		 ON CONFLICT (name) DO NOTHING`,
		academicYearID); err != nil {
		return fmt.Errorf("academic year 2026/2027: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO terms (id, academic_year_id, name, start_date, end_date, is_current)
		 VALUES ($1, $2, 'Term 1', '2026-09-08', '2026-12-18', true)
		 ON CONFLICT (academic_year_id, name) DO NOTHING`,
		term1ID, academicYearID); err != nil {
		return fmt.Errorf("term Term 1: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO programmes (id, code, name, description)
		 VALUES ($1, 'GEN', 'General Programme', 'Core curriculum for Form 1–4')
		 ON CONFLICT (code) DO NOTHING`,
		programmeID); err != nil {
		return fmt.Errorf("programme GEN: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO classes (id, academic_year_id, programme_id, name, level, capacity)
		 VALUES ($1, $2, $3, 'Form 1A', 'Form 1', 30)
		 ON CONFLICT (academic_year_id, name) DO NOTHING`,
		form1AID, academicYearID, programmeID); err != nil {
		return fmt.Errorf("class Form 1A: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO classes (id, academic_year_id, programme_id, name, level, capacity)
		 VALUES ($1, $2, $3, 'Form 1B', 'Form 1', 30)
		 ON CONFLICT (academic_year_id, name) DO NOTHING`,
		uuid.MustParse(form1BID), academicYearID, programmeID); err != nil {
		return fmt.Errorf("class Form 1B: %w", err)
	}

	// Class × subject curriculum: both classes take all six subjects.
	classes := []struct {
		id   uuid.UUID
		name string
	}{{form1AID, "Form 1A"}, {uuid.MustParse(form1BID), "Form 1B"}}
	i := 0
	for _, c := range classes {
		for _, s := range subjects {
			i++
			if _, err := pool.Exec(ctx,
				`INSERT INTO class_subjects (id, class_id, subject_id) VALUES ($1, $2, $3)
				 ON CONFLICT (class_id, subject_id) DO NOTHING`,
				uuid.MustParse(fmt.Sprintf(classSubjPrefix, i)), c.id, subjIDs[s.code]); err != nil {
				return fmt.Errorf("class_subject %s/%s: %w", c.name, s.code, err)
			}
		}
	}

	// --- Students: 10 more Form 1 students (2 already exist) ----------------
	type newStudent struct {
		id, personID uuid.UUID
		first, last  string
		number       string
		gender       string
		birthYear    int
		classID      uuid.UUID
		className    string
	}
	students := []newStudent{
		{first: "Abena", last: "Owusu", number: "STU-2026-003", gender: "FEMALE", birthYear: 2013, classID: form1AID, className: "Form 1A"},
		{first: "Yaw", last: "Darko", number: "STU-2026-004", gender: "MALE", birthYear: 2013, classID: form1AID, className: "Form 1A"},
		{first: "Efua", last: "Sarpong", number: "STU-2026-005", gender: "FEMALE", birthYear: 2014, classID: form1AID, className: "Form 1A"},
		{first: "Kojo", last: "Antwi", number: "STU-2026-006", gender: "MALE", birthYear: 2013, classID: form1AID, className: "Form 1A"},
		{first: "Akosua", last: "Boateng", number: "STU-2026-007", gender: "FEMALE", birthYear: 2014, classID: uuid.MustParse(form1BID), className: "Form 1B"},
		{first: "Kwabena", last: "Osei", number: "STU-2026-008", gender: "MALE", birthYear: 2013, classID: uuid.MustParse(form1BID), className: "Form 1B"},
		{first: "Adwoa", last: "Mensimah", number: "STU-2026-009", gender: "FEMALE", birthYear: 2013, classID: uuid.MustParse(form1BID), className: "Form 1B"},
		{first: "Nana", last: "Agyeman", number: "STU-2026-010", gender: "MALE", birthYear: 2014, classID: uuid.MustParse(form1BID), className: "Form 1B"},
		{first: "Esi", last: "Quartey", number: "STU-2026-011", gender: "FEMALE", birthYear: 2013, classID: uuid.MustParse(form1BID), className: "Form 1B"},
		{first: "Fiifi", last: "Ackon", number: "STU-2026-012", gender: "MALE", birthYear: 2014, classID: uuid.MustParse(form1BID), className: "Form 1B"},
	}
	for idx, st := range students {
		st.id = uuid.MustParse(fmt.Sprintf(studentPrefix, idx+3))
		st.personID = uuid.MustParse(fmt.Sprintf(personPrefix, idx+4))
		if _, err := pool.Exec(ctx,
			`INSERT INTO people (id, first_name, last_name, date_of_birth, gender, address)
			 VALUES ($1, $2, $3, $4, $5, 'Takoradi, Ghana') ON CONFLICT (id) DO NOTHING`,
			st.personID, st.first, st.last,
			fmt.Sprintf("%d-0%d-1%d", st.birthYear, (idx%9)+1, idx%9), st.gender); err != nil {
			return fmt.Errorf("person %s %s: %w", st.first, st.last, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO students (id, person_id, student_number, admission_date, status)
			 VALUES ($1, $2, $3, '2026-09-01', 'ACTIVE') ON CONFLICT (id) DO NOTHING`,
			st.id, st.personID, st.number); err != nil {
			return fmt.Errorf("student %s: %w", st.number, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO enrollments (id, student_id, academic_year_id, class_id, status, enrollment_date)
			 VALUES ($1, $2, $3, $4, 'ENROLLED', '2026-09-01')
			 ON CONFLICT (student_id, academic_year_id) WHERE status IN ('ENROLLED', 'COMPLETED', 'TRANSFERRED') DO NOTHING`,
			uuid.MustParse(fmt.Sprintf(enrollmentPrefix, idx+13)), st.id, academicYearID, st.classID); err != nil {
			return fmt.Errorf("enrollment %s: %w", st.number, err)
		}
	}

	// --- Demo family skeleton (fresh-database bootstrap) --------------------
	// Ama Boateng (STU-2026-001), Kofi Mensah (STU-2026-002), and their
	// guardian Abena Boateng were hand-seeded in the dev database long ago;
	// everything downstream (guardian links, report cards, the parent
	// portal story) assumes they exist. Self-provision them here with the
	// same fixed UUID numbering — every insert is idempotent, so on the dev
	// DB this is a no-op (all rows already exist and are kept as-is).
	amaPersonID := uuid.MustParse(fmt.Sprintf(personPrefix, 1))
	kofiPersonID := uuid.MustParse(fmt.Sprintf(personPrefix, 2))
	abenaPersonID := uuid.MustParse(fmt.Sprintf(personPrefix, 890))
	amaEnrollmentID := uuid.MustParse(fmt.Sprintf(enrollmentPrefix, 11))
	kofiEnrollmentID := uuid.MustParse(fmt.Sprintf(enrollmentPrefix, 12))
	abenaGuardianID := uuid.MustParse("cccc0000-0000-0000-0000-000000000890")

	for _, p := range []struct {
		id            uuid.UUID
		first, last   string
		gender        string
		dob           string
	}{
		{amaPersonID, "Ama", "Boateng", "FEMALE", "2013-03-12"},
		{kofiPersonID, "Kofi", "Mensah", "MALE", "2013-05-08"},
		{abenaPersonID, "Abena", "Boateng", "FEMALE", "1988-02-20"},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO people (id, first_name, last_name, date_of_birth, gender, address)
			 VALUES ($1, $2, $3, $4, $5, 'Takoradi, Ghana') ON CONFLICT (id) DO NOTHING`,
			p.id, p.first, p.last, p.dob, p.gender); err != nil {
			return fmt.Errorf("person %s %s: %w", p.first, p.last, err)
		}
	}
	for _, st := range []struct {
		id, personID uuid.UUID
		num          string
		classID      uuid.UUID
		enrollID     uuid.UUID
	}{
		{amaID, amaPersonID, "STU-2026-001", form1AID, amaEnrollmentID},
		{kofiID, kofiPersonID, "STU-2026-002", uuid.MustParse(form1BID), kofiEnrollmentID},
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO students (id, person_id, student_number, admission_date, status)
			 VALUES ($1, $2, $3, '2026-09-01', 'ACTIVE') ON CONFLICT (id) DO NOTHING`,
			st.id, st.personID, st.num); err != nil {
			return fmt.Errorf("student %s: %w", st.num, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO enrollments (id, student_id, academic_year_id, class_id, status, enrollment_date)
			 VALUES ($1, $2, $3, $4, 'ENROLLED', '2026-09-01')
			 ON CONFLICT (student_id, academic_year_id) WHERE status IN ('ENROLLED', 'COMPLETED', 'TRANSFERRED') DO NOTHING`,
			st.enrollID, st.id, academicYearID, st.classID); err != nil {
			return fmt.Errorf("enrollment %s: %w", st.num, err)
		}
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO guardians (id, person_id, occupation)
		 VALUES ($1, $2, 'Trader') ON CONFLICT (person_id) DO NOTHING`,
		abenaGuardianID, abenaPersonID); err != nil {
		return fmt.Errorf("guardian Abena Boateng: %w", err)
	}
	// First child link: Abena ↔ Ama (the dev skeleton's original demo pair).
	if _, err := pool.Exec(ctx,
		`INSERT INTO student_guardians (id, student_id, guardian_id, relationship)
		 VALUES ($1, $2, $3, 'MOTHER')
		 ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse(guardianLink1), amaID, abenaGuardianID); err != nil {
		return fmt.Errorf("guardian link parent↔Ama: %w", err)
	}

	// Second child for the demo parent: the same guardian also cares for
	// Kofi Mensah (Abena Boateng's fixed guardian row).
	if _, err := pool.Exec(ctx,
		`INSERT INTO student_guardians (id, student_id, guardian_id, relationship)
		 VALUES ($1, $2, $3, 'MOTHER')
		 ON CONFLICT (id) DO NOTHING`,
		uuid.MustParse(guardianLink2), kofiID, abenaGuardianID); err != nil {
		return fmt.Errorf("guardian link parent↔Kofi: %w", err)
	}

	// --- Teacher assignments ------------------------------------------------
	// Ada Obi: Mathematics & Integrated Science in both Form 1 classes
	// (F1A×MATH already exists as ffff0001 in the demo skeleton).
	// Sarah Mensah: English Language in both classes.
	type assignment struct {
		id        uuid.UUID
		teacherID uuid.UUID
		classID   uuid.UUID
		subjectID uuid.UUID
	}
	assignments := []assignment{
		{uuid.MustParse(fmt.Sprintf(assignPrefix, 2)), adaStaffID, form1AID, subjIDs["SCI"]},
		{uuid.MustParse(fmt.Sprintf(assignPrefix, 3)), adaStaffID, uuid.MustParse(form1BID), subjIDs["MATH"]},
		{uuid.MustParse(fmt.Sprintf(assignPrefix, 4)), adaStaffID, uuid.MustParse(form1BID), subjIDs["SCI"]},
		{uuid.MustParse(fmt.Sprintf(assignPrefix, 5)), sarahStaffID, form1AID, subjIDs["ENG"]},
		{uuid.MustParse(fmt.Sprintf(assignPrefix, 6)), sarahStaffID, uuid.MustParse(form1BID), subjIDs["ENG"]},
	}
	for _, a := range assignments {
		if _, err := pool.Exec(ctx,
			`INSERT INTO teacher_assignments (id, teacher_id, class_id, subject_id) VALUES ($1, $2, $3, $4)
			 ON CONFLICT (teacher_id, class_id, subject_id) DO NOTHING`,
			a.id, a.teacherID, a.classID, a.subjectID); err != nil {
			return fmt.Errorf("teacher assignment: %w", err)
		}
	}

	// --- Attendance: 20 most recent school days of Term 1 --------------------
	var termStart, termEnd time.Time
	if err := pool.QueryRow(ctx,
		`SELECT start_date, end_date FROM terms WHERE id = $1`, term1ID).Scan(&termStart, &termEnd); err != nil {
		return fmt.Errorf("term lookup: %w", err)
	}
	schoolDays := recentWeekdays(termStart, time.Now().UTC(), 20)

	type combo struct {
		classID, subjectID uuid.UUID
		subjectCode        string
		teacherID          uuid.UUID
	}
	combos := []combo{
		{form1AID, subjIDs["MATH"], "MATH", adaStaffID},
		{form1AID, subjIDs["SCI"], "SCI", adaStaffID},
		{form1AID, subjIDs["ENG"], "ENG", sarahStaffID},
		{uuid.MustParse(form1BID), subjIDs["MATH"], "MATH", adaStaffID},
		{uuid.MustParse(form1BID), subjIDs["SCI"], "SCI", adaStaffID},
		{uuid.MustParse(form1BID), subjIDs["ENG"], "ENG", sarahStaffID},
	}

	// Per-student attendance temperament so individual summaries look human.
	roster := map[uuid.UUID][]uuid.UUID{}
	for _, c := range classes {
		rows, err := pool.Query(ctx,
			`SELECT student_id FROM enrollments
			 WHERE class_id = $1 AND academic_year_id = $2 AND status = 'ENROLLED'
			 ORDER BY student_id`, c.id, academicYearID)
		if err != nil {
			return fmt.Errorf("roster %s: %w", c.name, err)
		}
		for rows.Next() {
			var sid uuid.UUID
			if err := rows.Scan(&sid); err != nil {
				rows.Close()
				return fmt.Errorf("roster scan: %w", err)
			}
			roster[c.id] = append(roster[c.id], sid)
		}
		rows.Close()
	}

	sessions, records := 0, 0
	for _, cmb := range combos {
		for d, day := range schoolDays {
			var sessionID uuid.UUID
			err := pool.QueryRow(ctx,
				`INSERT INTO attendance_sessions (class_id, subject_id, term_id, session_date)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (class_id, subject_id, term_id, session_date) DO NOTHING
				 RETURNING id`,
				cmb.classID, cmb.subjectID, term1ID, day).Scan(&sessionID)
			if err == pgx.ErrNoRows {
				// Session already existed — resolve its id.
				if err := pool.QueryRow(ctx,
					`SELECT id FROM attendance_sessions
					 WHERE class_id = $1 AND subject_id = $2 AND term_id = $3 AND session_date = $4`,
					cmb.classID, cmb.subjectID, term1ID, day).Scan(&sessionID); err != nil {
					return fmt.Errorf("existing session lookup: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("attendance session: %w", err)
			} else {
				sessions++
			}

			for si, studentID := range roster[cmb.classID] {
				draw := rng.Float64() + float64(si%3)*0.005 - float64(d%4)*0.004
				status := "PRESENT"
				switch {
				case draw < 0.06:
					status = "ABSENT"
				case draw < 0.12:
					status = "LATE"
				case draw < 0.145:
					status = "EXCUSED"
				}
				tag, err := pool.Exec(ctx,
					`INSERT INTO attendance_records (session_id, student_id, status)
					 VALUES ($1, $2, $3) ON CONFLICT (session_id, student_id) DO NOTHING`,
					sessionID, studentID, status)
				if err != nil {
					return fmt.Errorf("attendance record: %w", err)
				}
				records += int(tag.RowsAffected())
			}
		}
	}

	// --- Assessments & results ----------------------------------------------
	// Two assessments per class-subject pairing this term; per-student
	// "ability" keeps results stable and realistic across subjects.
	abilities := map[uuid.UUID]int{} // 45..92
	for cid, list := range roster {
		for j, sid := range list {
			abilities[sid] = 50 + ((j*7 + int(cid[0])) % 43)
		}
	}

	type assessSpec struct {
		title    string
		typ      string
		maxScore float64
		weight   float64
		dayOff   int // offset into schoolDays
	}
	specs := []assessSpec{
		{"Quiz 1", "QUIZ", 20, 10, 2},
		{"Midterm Test", "MIDTERM", 100, 30, 9},
	}

	assessments, results := 0, 0
	i = 0
	for _, cmb := range combos {
		for _, spec := range specs {
			i++
			assessDate := schoolDays[spec.dayOff].Format("2006-01-02")
			var assessmentID uuid.UUID
			err := pool.QueryRow(ctx,
				`INSERT INTO assessments (id, class_id, subject_id, term_id, title, type, max_score, weight, assessment_date)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 ON CONFLICT (class_id, subject_id, term_id, title) DO NOTHING
				 RETURNING id`,
				uuid.MustParse(fmt.Sprintf(assessmentPrefix, i)),
				cmb.classID, cmb.subjectID, term1ID, spec.title, spec.typ,
				spec.maxScore, spec.weight, assessDate).Scan(&assessmentID)
			if err == pgx.ErrNoRows {
				if err := pool.QueryRow(ctx,
					`SELECT id FROM assessments WHERE class_id = $1 AND subject_id = $2 AND term_id = $3 AND title = $4`,
					cmb.classID, cmb.subjectID, term1ID, spec.title).Scan(&assessmentID); err != nil {
					return fmt.Errorf("existing assessment lookup: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("assessment %s: %w", spec.title, err)
			} else {
				assessments++
			}

			for _, studentID := range roster[cmb.classID] {
				pct := float64(abilities[studentID]) + rng.Float64()*12 - 6
				if pct > 100 {
					pct = 100
				}
				if pct < 20 {
					pct = 20
				}
				score := float64(int((spec.maxScore*pct/100)*4+0.5)) / 4
				grade, remark := gradeFor(pct)
				tag, err := pool.Exec(ctx,
					`INSERT INTO results (assessment_id, student_id, score, grade, remarks)
					 VALUES ($1, $2, $3, $4, $5) ON CONFLICT (assessment_id, student_id) DO NOTHING`,
					assessmentID, studentID, score, grade, remark)
				if err != nil {
					return fmt.Errorf("result: %w", err)
				}
				results += int(tag.RowsAffected())
			}
		}
	}

	// --- Announcements --------------------------------------------------------
	// Author is a master admin (see announcementAuthorEmail comment above).
	var adminUserID uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT u.id FROM users u
		 JOIN user_roles ur ON ur.user_id = u.id
		 JOIN roles r ON r.id = ur.role_id
		 WHERE r.name = 'MASTER_ADMIN'
		 ORDER BY u.created_at
		 LIMIT 1`).Scan(&adminUserID); err != nil {
		return fmt.Errorf("announcement author lookup (no MASTER_ADMIN user found — create the master admin first): %w", err)
	}

	expires := termEnd.Format("2006-01-02") + " 18:00:00+00"
	announcements := []struct {
		id      uuid.UUID
		title   string
		content string
		status  string
		target  string
		role    *string
		class   *uuid.UUID
		prio    string
	}{
		{uuid.MustParse(fmt.Sprintf(announcePrefix, 1)), "PTA General Meeting",
			"The first Parent-Teacher Association meeting of the year takes place on Friday at 5:00 PM in the main hall. All parents and guardians are encouraged to attend.",
			"PUBLISHED", "ALL", nil, nil, "URGENT"},
		{uuid.MustParse(fmt.Sprintf(announcePrefix, 2)), "Staff Briefing — Monday Morning",
			"All teaching staff should report to the staff common room at 7:30 AM on Monday for the weekly briefing. Lesson notes will be reviewed.",
			"PUBLISHED", "ROLE", strPtr("TEACHER"), nil, "HIGH"},
		{uuid.MustParse(fmt.Sprintf(announcePrefix, 3)), "Mathematics Quiz on Friday",
			"Form 1A will write a mathematics quiz covering fractions and decimals this Friday during the first period. Come prepared with your exercise books.",
			"PUBLISHED", "CLASS", nil, &form1AID, "NORMAL"},
		{uuid.MustParse(fmt.Sprintf(announcePrefix, 4)), "Draft: Inter-Schools Sports Festival",
			"[Draft] Planning for the inter-schools sports festival is underway. Team selection trials will be announced once the schedule is confirmed.",
			"DRAFT", "ALL", nil, nil, "LOW"},
		{uuid.MustParse(fmt.Sprintf(announcePrefix, 5)), "Last Term's Newsletter",
			"Archive copy of the previous term's end-of-term newsletter, kept for reference.",
			"ARCHIVED", "ALL", nil, nil, "LOW"},
	}
	for _, a := range announcements {
		if _, err := pool.Exec(ctx,
			`INSERT INTO announcements (id, title, content, author_id, status, target_type, target_role, target_class_id, priority, published_at, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
			         CASE WHEN $5 = 'PUBLISHED' THEN now() ELSE NULL END,
			         CASE WHEN $5 = 'PUBLISHED' THEN $10::timestamptz ELSE NULL END)
			 ON CONFLICT (id) DO NOTHING`,
			a.id, a.title, a.content, adminUserID, a.status, a.target, a.role, a.class, a.prio, expires); err != nil {
			return fmt.Errorf("announcement %q: %w", a.title, err)
		}
	}

	log.Printf("demo-seed: +%d attendance sessions, +%d attendance records, +%d assessments, +%d results",
		sessions, records, assessments, results)
	// NOTE (M34): no demo login accounts are seeded. QA personas and their
	// rotated passwords live in the gitignored frontend/.check-credentials.json.
	return nil
}

// recentWeekdays returns up to `count` most recent weekdays (Mon–Fri) in
// [from, to], oldest first.
func recentWeekdays(from, to time.Time, count int) []time.Time {
	var days []time.Time
	for d := to; !d.Before(from); d = d.AddDate(0, 0, -1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			days = append(days, time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC))
		}
		if len(days) == count {
			break
		}
	}
	// reverse → oldest first
	for l, r := 0, len(days)-1; l < r; l, r = l+1, r-1 {
		days[l], days[r] = days[r], days[l]
	}
	return days
}

func gradeFor(pct float64) (string, string) {
	switch {
	case pct >= 80:
		return "A", "Excellent work"
	case pct >= 70:
		return "B", "Very good"
	case pct >= 60:
		return "C", "Good effort"
	case pct >= 50:
		return "D", "Needs improvement"
	default:
		return "F", "Must do better — see teacher"
	}
}

func strPtr(s string) *string { return &s }
