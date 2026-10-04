package staff

import "testing"

// NormalizePagination clamps client input; these bounds are what keep one
// request from pulling the whole staff table (section 25 pagination).
func TestNormalizePagination(t *testing.T) {
	cases := []struct {
		name             string
		page, pageSize   int
		wantPage, wantSz int
	}{
		{"defaults on zero input", 0, 0, 1, DefaultPageSize},
		{"negative page clamps to 1", -3, 10, 1, 10},
		{"page 1 passes through", 1, 10, 1, 10},
		{"large page passes through", 42, 10, 42, 10},
		{"zero page size gets default", 2, 0, 2, DefaultPageSize},
		{"negative page size gets default", 2, -5, 2, DefaultPageSize},
		{"oversized page size clamps to max", 1, 100000, 1, MaxPageSize},
		{"max page size passes through", 1, MaxPageSize, 1, MaxPageSize},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, size := NormalizePagination(tc.page, tc.pageSize)
			if page != tc.wantPage || size != tc.wantSz {
				t.Fatalf("NormalizePagination(%d, %d) = (%d, %d), want (%d, %d)",
					tc.page, tc.pageSize, page, size, tc.wantPage, tc.wantSz)
			}
		})
	}
}

// The status/category sets must match the CHECK constraints in migrations
// 000005 (staff.employment_status, staff_positions.category) — a value
// accepted here but rejected by the database would turn a validation error
// into a 500.
func TestStatusAndCategorySetsMatchMigrations(t *testing.T) {
	wantStatuses := map[string]bool{
		"ACTIVE": true, "INACTIVE": true, "SUSPENDED": true, "ON_LEAVE": true,
	}
	if len(EmploymentStatuses) != len(wantStatuses) {
		t.Fatalf("EmploymentStatuses = %v, want exactly %v", EmploymentStatuses, wantStatuses)
	}
	for _, s := range EmploymentStatuses {
		if !wantStatuses[s] {
			t.Errorf("unexpected employment status %q", s)
		}
	}

	wantCategories := map[string]bool{
		"TEACHING": true, "ADMINISTRATION": true, "ICT": true,
		"ACCOUNTS": true, "MANAGEMENT": true, "SUPPORT": true,
	}
	if len(PositionCategories) != len(wantCategories) {
		t.Fatalf("PositionCategories = %v, want exactly %v", PositionCategories, wantCategories)
	}
	for _, c := range PositionCategories {
		if !wantCategories[c] {
			t.Errorf("unexpected position category %q", c)
		}
	}
}
