package bigquery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testCountSQL = "SELECT COUNT(*) FROM t WHERE pais = @pais"
	testPageSQL  = "SELECT idunico, numero, fecha_factura, document_type FROM t WHERE pais = @pais LIMIT @pageSize OFFSET @offset"
)

var (
	testFilterParams = []Param{P("pais", "AR")}
	testPageParams   = []Param{P(PageSizeParam, int64(20)), P(OffsetParam, int64(0))}
	testAllParams    = []Param{P("pais", "AR"), P(PageSizeParam, int64(20)), P(OffsetParam, int64(0))}
)

func findCall(calls []call, sql string) (call, bool) {
	for _, c := range calls {
		if c.sql == sql {
			return c, true
		}
	}
	return call{}, false
}

func TestQueryPageSequentialRunsCountThenPage(t *testing.T) {
	p := &fakeProvider{
		countRows: newFakeRows([]string{"f0_"}, []Value{int64(2)}),
		pageRows: newFakeRows(invoiceColumns,
			invoiceRow("A-1", 10, "2026-02-01", "FC"),
			invoiceRow("A-2", 11, "2026-02-02", "NC"),
		),
	}

	page, err := QueryPage[taggedRow](context.Background(), p, testCountSQL, testPageSQL, testFilterParams, testPageParams)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), page.Total)
	assert.Len(t, page.Rows, 2)
	assert.Equal(t, "A-2", page.Rows[1].Idunico.StringVal)

	// Orden y params: el COUNT sólo lleva los de filtro, la page todos.
	assert.Equal(t, []call{
		{sql: testCountSQL, params: testFilterParams},
		{sql: testPageSQL, params: testAllParams},
	}, p.calls)
}

// Si el total es cero, la page query (la cara) no se ejecuta.
func TestQueryPageSkipsPageQueryWhenTotalIsZero(t *testing.T) {
	p := &fakeProvider{
		countRows: newFakeRows([]string{"f0_"}, []Value{int64(0)}),
		pageErr:   errBoom, // si se ejecutara, fallaría
	}

	page, err := QueryPage[taggedRow](context.Background(), p, testCountSQL, testPageSQL, testFilterParams, testPageParams)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), page.Total)
	assert.NotNil(t, page.Rows)
	assert.Empty(t, page.Rows)
	assert.Equal(t, []call{{sql: testCountSQL, params: testFilterParams}}, p.calls)
}

func TestQueryPageParallelRunsBothEvenWhenTotalIsZero(t *testing.T) {
	p := &fakeProvider{
		countRows: newFakeRows([]string{"f0_"}, []Value{int64(0)}),
		pageRows:  newFakeRows(invoiceColumns),
	}

	page, err := QueryPage[taggedRow](context.Background(), p, testCountSQL, testPageSQL, testFilterParams, testPageParams, Parallel())

	assert.NoError(t, err)
	assert.Equal(t, int64(0), page.Total)
	assert.Empty(t, page.Rows)
	assert.Len(t, p.calls, 2)

	countCall, ok := findCall(p.calls, testCountSQL)
	assert.True(t, ok)
	assert.Equal(t, testFilterParams, countCall.params)

	pageCall, ok := findCall(p.calls, testPageSQL)
	assert.True(t, ok)
	assert.Equal(t, testAllParams, pageCall.params)
}

func TestQueryPageParallelReturnsRows(t *testing.T) {
	p := &fakeProvider{
		countRows: newFakeRows([]string{"f0_"}, []Value{int64(1)}),
		pageRows:  newFakeRows(invoiceColumns, invoiceRow("A-1", 10, "2026-02-01", "FC")),
	}

	page, err := QueryPage[taggedRow](context.Background(), p, testCountSQL, testPageSQL, testFilterParams, testPageParams, Parallel())

	assert.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Len(t, page.Rows, 1)
}

func TestQueryPageErrors(t *testing.T) {
	testCases := []struct {
		name     string
		provider *fakeProvider
		opts     []PageOption
		expected error
	}{
		{
			name:     "count error sequential",
			provider: &fakeProvider{countErr: errBoom},
			expected: errBoom,
		},
		{
			name: "page error sequential",
			provider: &fakeProvider{
				countRows: newFakeRows([]string{"f0_"}, []Value{int64(1)}),
				pageErr:   errBoom,
			},
			expected: errBoom,
		},
		{
			name:     "count error parallel",
			provider: &fakeProvider{countErr: errBoom, pageRows: newFakeRows(invoiceColumns)},
			opts:     []PageOption{Parallel()},
			expected: errBoom,
		},
		{
			name: "page error parallel",
			provider: &fakeProvider{
				countRows: newFakeRows([]string{"f0_"}, []Value{int64(1)}),
				pageErr:   errBoom,
			},
			opts:     []PageOption{Parallel()},
			expected: errBoom,
		},
		{
			name: "unmapped column surfaces through QueryPage",
			provider: &fakeProvider{
				countRows: newFakeRows([]string{"f0_"}, []Value{int64(1)}),
				pageRows:  newFakeRows(invoiceColumns, invoiceRow("A-1", 10, "2026-02-01", "FC")),
			},
			expected: ErrUnmappedColumn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				page Page[untaggedRow]
				err  error
			)
			page, err = QueryPage[untaggedRow](context.Background(), tc.provider, testCountSQL, testPageSQL, testFilterParams, testPageParams, tc.opts...)

			assert.ErrorIs(t, err, tc.expected)
			assert.Zero(t, page)
		})
	}
}
