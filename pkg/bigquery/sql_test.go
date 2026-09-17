package bigquery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	assert.NoError(t, err)
	return parsed
}

func TestWhereClause(t *testing.T) {
	testCases := []struct {
		name       string
		conditions []string
		expected   string
	}{
		{name: "no conditions", conditions: nil, expected: ""},
		{name: "only empty conditions are dropped", conditions: []string{"", "  "}, expected: ""},
		{name: "single condition", conditions: []string{"pais = @pais"}, expected: "WHERE pais = @pais"},
		{
			name:       "conditions joined with AND, blanks skipped",
			conditions: []string{"a = @a", "", "b >= @b"},
			expected:   "WHERE a = @a AND b >= @b",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, WhereClause(tc.conditions...))
		})
	}
}

func TestCountSQL(t *testing.T) {
	table := "`proj.ds.facturas_*`"

	assert.Equal(t, "SELECT COUNT(*) FROM `proj.ds.facturas_*`", CountSQL(table, ""))
	assert.Equal(t, "SELECT COUNT(*) FROM `proj.ds.facturas_*` WHERE pais = @pais", CountSQL(table, "WHERE pais = @pais"))
}

func TestPaginate(t *testing.T) {
	base := "SELECT a FROM t ORDER BY a"

	testCases := []struct {
		name           string
		page           int
		pageSize       int
		expectedSQL    string
		expectedOffset int64
		expectedErr    error
	}{
		{name: "first page", page: 1, pageSize: 20, expectedSQL: base + " LIMIT @pageSize OFFSET @offset", expectedOffset: 0},
		{name: "third page", page: 3, pageSize: 25, expectedSQL: base + " LIMIT @pageSize OFFSET @offset", expectedOffset: 50},
		{name: "page zero is rejected", page: 0, pageSize: 20, expectedErr: ErrInvalidPage},
		{name: "negative page is rejected", page: -1, pageSize: 20, expectedErr: ErrInvalidPage},
		{name: "pageSize zero is rejected", page: 1, pageSize: 0, expectedErr: ErrInvalidPageSize},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql, params, err := Paginate(base, tc.page, tc.pageSize)

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Empty(t, sql)
				assert.Nil(t, params)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedSQL, sql)
			assert.Equal(t, []Param{
				{Name: PageSizeParam, Value: int64(tc.pageSize)},
				{Name: OffsetParam, Value: tc.expectedOffset},
			}, params)
		})
	}
}

func TestTableSuffixCondition(t *testing.T) {
	testCases := []struct {
		name     string
		from     time.Time
		to       time.Time
		expected string
		pruned   bool
	}{
		{name: "no bounds leaves the wildcard unpruned"},
		{
			name:     "closed range",
			from:     mustDate(t, "2026-02-10"),
			to:       mustDate(t, "2026-02-15"),
			expected: "_TABLE_SUFFIX BETWEEN '20260210' AND '20260215'",
			pruned:   true,
		},
		{
			name:     "single day",
			from:     mustDate(t, "2026-02-02"),
			to:       mustDate(t, "2026-02-02"),
			expected: "_TABLE_SUFFIX BETWEEN '20260202' AND '20260202'",
			pruned:   true,
		},
		{
			name:     "only from",
			from:     mustDate(t, "2026-01-01"),
			expected: "_TABLE_SUFFIX >= '20260101'",
			pruned:   true,
		},
		{
			name:     "only to",
			to:       mustDate(t, "2026-03-31"),
			expected: "_TABLE_SUFFIX <= '20260331'",
			pruned:   true,
		},
		{
			name:     "an invalid from drops that bound only",
			from:     time.Date(12345, 1, 1, 0, 0, 0, 0, time.UTC),
			to:       mustDate(t, "2026-03-31"),
			expected: "_TABLE_SUFFIX <= '20260331'",
			pruned:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			condition, pruned := TableSuffixCondition(tc.from, tc.to)
			assert.Equal(t, tc.pruned, pruned)
			assert.Equal(t, tc.expected, condition)
		})
	}
}

// El literal de _TABLE_SUFFIX es lo único que se interpola en el SQL, así que
// cualquier valor que no sea exactamente 8 dígitos tiene que quedar afuera.
func TestTableSuffixGuard(t *testing.T) {
	testCases := []struct {
		name     string
		date     time.Time
		expected string
		ok       bool
	}{
		{name: "zero date has no suffix", date: time.Time{}, ok: false},
		{name: "regular date", date: mustDate(t, "2026-02-10"), expected: "20260210", ok: true},
		{name: "five-digit year is rejected", date: time.Date(12345, 6, 1, 0, 0, 0, 0, time.UTC), ok: false},
		{name: "three-digit year is zero-padded to 8 digits", date: time.Date(999, 6, 1, 0, 0, 0, 0, time.UTC), expected: "09990601", ok: true},
		{name: "negative year is rejected", date: time.Date(-50, 6, 1, 0, 0, 0, 0, time.UTC), ok: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suffix, ok := TableSuffix(tc.date)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.expected, suffix)
			if ok {
				assert.Len(t, suffix, tableSuffixDigits)
			}
		})
	}
}

func TestIsTableSuffix(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"20260210", true},
		{"2026021", false},
		{"202602100", false},
		{"2026021a", false},
		{"2026-02-1", false},
		{"", false},
		{"'; DROP", false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			assert.Equal(t, tc.expected, isTableSuffix(tc.value))
		})
	}
}
