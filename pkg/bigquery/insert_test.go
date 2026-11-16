package bigquery

import (
	"context"
	"testing"
	"time"

	bq "cloud.google.com/go/bigquery"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const insertTestTable = "`proj.ds.table`"

var insertTestNow = time.Date(2026, 11, 16, 12, 30, 0, 0, time.UTC)

func TestBuildInsert(t *testing.T) {
	user := InsertResult{CreatedAt: insertTestNow, CreatedBy: "user@example.com"}
	userNull := bq.NullString{StringVal: "user@example.com", Valid: true}

	testCases := []struct {
		name           string
		values         []Param
		audit          InsertResult
		expectedSQL    string
		expectedParams []Param
	}{
		{
			name:           "no audit columns",
			values:         []Param{P("id", "1"), P("name", "a")},
			audit:          user,
			expectedSQL:    "INSERT INTO `proj.ds.table` (id, name) VALUES (@id, @name)",
			expectedParams: []Param{P("id", "1"), P("name", "a")},
		},
		{
			name:           "both audit columns get the lib values",
			values:         []Param{P("id", "1"), P("created_at", nil), P("created_by", nil)},
			audit:          user,
			expectedSQL:    "INSERT INTO `proj.ds.table` (id, created_at, created_by) VALUES (@id, @created_at, @created_by)",
			expectedParams: []Param{P("id", "1"), P("created_at", insertTestNow), P("created_by", userNull)},
		},
		{
			name:           "only created_at",
			values:         []Param{P("created_at", nil), P("id", "1")},
			audit:          user,
			expectedSQL:    "INSERT INTO `proj.ds.table` (created_at, id) VALUES (@created_at, @id)",
			expectedParams: []Param{P("created_at", insertTestNow), P("id", "1")},
		},
		{
			name:           "only created_by",
			values:         []Param{P("id", "1"), P("created_by", nil)},
			audit:          user,
			expectedSQL:    "INSERT INTO `proj.ds.table` (id, created_by) VALUES (@id, @created_by)",
			expectedParams: []Param{P("id", "1"), P("created_by", userNull)},
		},
		{
			name:           "a value sent for an audit column is replaced",
			values:         []Param{P("created_at", time.Time{}), P("created_by", "spoofed")},
			audit:          user,
			expectedSQL:    "INSERT INTO `proj.ds.table` (created_at, created_by) VALUES (@created_at, @created_by)",
			expectedParams: []Param{P("created_at", insertTestNow), P("created_by", userNull)},
		},
		{
			name:           "without user created_by is NULL",
			values:         []Param{P("id", "1"), P("created_by", nil)},
			audit:          InsertResult{CreatedAt: insertTestNow},
			expectedSQL:    "INSERT INTO `proj.ds.table` (id, created_by) VALUES (@id, @created_by)",
			expectedParams: []Param{P("id", "1"), P("created_by", bq.NullString{})},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql, params, err := buildInsert(insertTestTable, tc.values, tc.audit)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedSQL, sql)
			assert.Equal(t, tc.expectedParams, params)
		})
	}
}

func TestBuildInsertDoesNotMutateValues(t *testing.T) {
	values := []Param{P("id", "1"), P("created_at", nil)}

	_, _, err := buildInsert(insertTestTable, values, InsertResult{CreatedAt: insertTestNow})

	require.NoError(t, err)
	assert.Nil(t, values[1].Value)
}

func TestBuildInsertErrors(t *testing.T) {
	testCases := []struct {
		name        string
		values      []Param
		expectedErr error
	}{
		{name: "no values", values: nil, expectedErr: ErrNoInsertValues},
		{name: "empty values", values: []Param{}, expectedErr: ErrNoInsertValues},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql, params, err := buildInsert(insertTestTable, tc.values, InsertResult{CreatedAt: insertTestNow})

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Empty(t, sql)
			assert.Nil(t, params)
		})
	}
}

func TestWithCreatedBy(t *testing.T) {
	testCases := []struct {
		name     string
		ctx      context.Context
		expected string
	}{
		{name: "no user", ctx: context.Background(), expected: ""},
		{name: "user", ctx: WithCreatedBy(context.Background(), "user@example.com"), expected: "user@example.com"},
		{name: "user is trimmed", ctx: WithCreatedBy(context.Background(), "  user  "), expected: "user"},
		{name: "blank user", ctx: WithCreatedBy(context.Background(), "   "), expected: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, CreatedByFrom(tc.ctx))
		})
	}
}

// Insert valida antes de tocar el cliente: un error de armado nunca llega a
// BigQuery.
func TestInsertValidatesBeforeRunning(t *testing.T) {
	p := &provider{now: time.Now}

	result, err := p.Insert(context.Background(), insertTestTable, nil)

	assert.ErrorIs(t, err, ErrNoInsertValues)
	assert.Equal(t, InsertResult{}, result)
}
