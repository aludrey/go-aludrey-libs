package bigquery

import (
	"context"
	"sync"
	"testing"

	bq "cloud.google.com/go/bigquery"

	"github.com/stretchr/testify/assert"
)

func TestNewProviderValidation(t *testing.T) {
	testCases := []struct {
		name        string
		projectID   string
		credentials string
		expectedErr error
	}{
		{name: "empty project", projectID: "", credentials: `{"type":"service_account"}`, expectedErr: ErrEmptyProjectID},
		{name: "empty credentials", projectID: "proj", credentials: "", expectedErr: ErrEmptyCredentials},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewProvider(tc.projectID, tc.credentials)

			assert.Nil(t, p)
			assert.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestNewProviderRejectsMalformedCredentials(t *testing.T) {
	p, err := NewProvider("proj", "not-json")

	assert.Nil(t, p)
	assert.Error(t, err)
}

func TestScalarInt64(t *testing.T) {
	testCases := []struct {
		name        string
		rows        *fakeRows
		expected    int64
		expectedErr error
	}{
		{
			name:     "count row",
			rows:     newFakeRows([]string{"f0_"}, []Value{int64(42)}),
			expected: 42,
		},
		{
			name:     "zero count",
			rows:     newFakeRows([]string{"f0_"}, []Value{int64(0)}),
			expected: 0,
		},
		{
			name:        "no rows",
			rows:        newFakeRows([]string{"f0_"}),
			expectedErr: ErrNoRows,
		},
		{
			name:        "wrong type is an error, not a silent zero",
			rows:        newFakeRows([]string{"f0_"}, []Value{"42"}),
			expectedErr: ErrNotScalar,
		},
		{
			name:        "more than one column",
			rows:        newFakeRows([]string{"a", "b"}, []Value{int64(1), int64(2)}),
			expectedErr: ErrNotScalar,
		},
		{
			name:        "empty row",
			rows:        newFakeRows(nil, []Value{}),
			expectedErr: ErrNotScalar,
		},
		{
			name:        "iterator error is propagated",
			rows:        &fakeRows{nextErr: errBoom},
			expectedErr: errBoom,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := scalarInt64(tc.rows)

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Zero(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestToQueryParameters(t *testing.T) {
	assert.Nil(t, toQueryParameters(nil))

	got := toQueryParameters([]Param{P("pais", "AR"), P("numero", int64(3))})

	assert.Len(t, got, 2)
	assert.Equal(t, "pais", got[0].Name)
	assert.Equal(t, "AR", got[0].Value)
	assert.Equal(t, "numero", got[1].Name)
	assert.Equal(t, int64(3), got[1].Value)
}

// fakeProvider registra las queries ejecutadas y responde con Rows
// preparados. Sirve para testear QueryPage sin red.
type fakeProvider struct {
	countRows *fakeRows
	countErr  error
	pageRows  *fakeRows
	pageErr   error

	mu    sync.Mutex
	calls []call
}

type call struct {
	sql    string
	params []Param
}

func (f *fakeProvider) record(sql string, params []Param) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call{sql: sql, params: params})
}

func (f *fakeProvider) Read(_ context.Context, sql string, params []Param) (Rows, error) {
	f.record(sql, params)
	if f.pageErr != nil {
		return nil, f.pageErr
	}
	return f.pageRows, nil
}

func (f *fakeProvider) Count(_ context.Context, sql string, params []Param) (int64, error) {
	f.record(sql, params)
	if f.countErr != nil {
		return 0, f.countErr
	}
	return scalarInt64(f.countRows)
}

func (f *fakeProvider) Exec(_ context.Context, sql string, params []Param) (int64, error) {
	f.record(sql, params)
	return 0, nil
}

func (f *fakeProvider) Close() error { return nil }

func TestAffectedRows(t *testing.T) {
	dml := func(statementType string, affected int64) *bq.JobStatus {
		return &bq.JobStatus{
			State: bq.Done,
			Statistics: &bq.JobStatistics{
				Details: &bq.QueryStatistics{StatementType: statementType, NumDMLAffectedRows: affected},
			},
		}
	}

	testCases := []struct {
		name        string
		status      *bq.JobStatus
		expected    int64
		expectedErr error
	}{
		{name: "insert affected rows", status: dml("INSERT", 1), expected: 1},
		{name: "update affected several rows", status: dml("UPDATE", 3), expected: 3},
		{name: "delete affected rows", status: dml("DELETE", 2), expected: 2},
		{name: "merge affected rows", status: dml("MERGE", 4), expected: 4},
		{name: "zero affected rows is not an error", status: dml("UPDATE", 0), expected: 0},
		{name: "select is not DML", status: dml("SELECT", 0), expectedErr: ErrNotDML},
		{name: "nil status", status: nil, expectedErr: ErrNotDML},
		{name: "missing statistics", status: &bq.JobStatus{State: bq.Done}, expectedErr: ErrNotDML},
		{
			name: "non query statistics",
			status: &bq.JobStatus{
				State:      bq.Done,
				Statistics: &bq.JobStatistics{Details: &bq.LoadStatistics{}},
			},
			expectedErr: ErrNotDML,
		},
		{
			name: "nil query statistics",
			status: &bq.JobStatus{
				State:      bq.Done,
				Statistics: &bq.JobStatistics{Details: (*bq.QueryStatistics)(nil)},
			},
			expectedErr: ErrNotDML,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := affectedRows(tc.status)

			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.Zero(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}
}
