package bigquery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// taggedRow mapea todas las columnas de la proyección de facturas.
type taggedRow struct {
	Idunico      NullString `bigquery:"idunico"`
	Numero       NullInt64  `bigquery:"numero"`
	FechaFactura NullString `bigquery:"fecha_factura"`
	DocumentType NullString `bigquery:"document_type"`
}

// untaggedRow es el bug original: sin tags, las columnas snake_case no
// matchean contra CamelCase y el driver las deja en zero value.
type untaggedRow struct {
	Idunico      NullString
	Numero       NullInt64
	FechaFactura NullString
	DocumentType NullString
}

var invoiceColumns = []string{"idunico", "numero", "fecha_factura", "document_type"}

func invoiceRow(id string, numero int64, fecha string, docType string) []Value {
	return []Value{
		NullString{StringVal: id, Valid: true},
		NullInt64{Int64: numero, Valid: true},
		NullString{StringVal: fecha, Valid: true},
		NullString{StringVal: docType, Valid: true},
	}
}

func TestScanAllLoadsTaggedRows(t *testing.T) {
	rows := newFakeRows(invoiceColumns,
		invoiceRow("A-1", 10, "2026-02-01", "FC"),
		invoiceRow("A-2", 11, "2026-02-02", "NC"),
	)

	got, err := ScanAll[taggedRow](rows)

	assert.NoError(t, err)
	assert.Equal(t, []taggedRow{
		{
			Idunico:      NullString{StringVal: "A-1", Valid: true},
			Numero:       NullInt64{Int64: 10, Valid: true},
			FechaFactura: NullString{StringVal: "2026-02-01", Valid: true},
			DocumentType: NullString{StringVal: "FC", Valid: true},
		},
		{
			Idunico:      NullString{StringVal: "A-2", Valid: true},
			Numero:       NullInt64{Int64: 11, Valid: true},
			FechaFactura: NullString{StringVal: "2026-02-02", Valid: true},
			DocumentType: NullString{StringVal: "NC", Valid: true},
		},
	}, got)
}

// El caso que motiva el paquete: una columna del SELECT sin campo destino
// tiene que ser un error, no una fila medio vacía.
func TestScanAllFailsOnUnmappedColumn(t *testing.T) {
	testCases := []struct {
		name     string
		scan     func(Rows) (int, error)
		expected string
	}{
		{
			name: "struct without tags misses every snake_case column",
			scan: func(r Rows) (int, error) {
				got, err := ScanAll[untaggedRow](r)
				return len(got), err
			},
			expected: "document_type, fecha_factura",
		},
		{
			name: "struct missing a single tag",
			scan: func(r Rows) (int, error) {
				type partial struct {
					Idunico      NullString `bigquery:"idunico"`
					Numero       NullInt64  `bigquery:"numero"`
					FechaFactura NullString `bigquery:"fecha_factura"`
					DocumentType NullString // sin tag: no matchea document_type
				}
				got, err := ScanAll[partial](r)
				return len(got), err
			},
			expected: "document_type",
		},
		{
			name: "a field excluded with \"-\" does not count as destination",
			scan: func(r Rows) (int, error) {
				type excluded struct {
					Idunico      NullString `bigquery:"idunico"`
					Numero       NullInt64  `bigquery:"numero"`
					FechaFactura NullString `bigquery:"fecha_factura"`
					DocumentType NullString `bigquery:"-"`
				}
				got, err := ScanAll[excluded](r)
				return len(got), err
			},
			expected: "document_type",
		},
		{
			name: "an unexported field is not a destination",
			scan: func(r Rows) (int, error) {
				type unexported struct {
					Idunico      NullString `bigquery:"idunico"`
					Numero       NullInt64  `bigquery:"numero"`
					FechaFactura NullString `bigquery:"fecha_factura"`
					documentType NullString `bigquery:"document_type"` //nolint:unused
				}
				got, err := ScanAll[unexported](r)
				return len(got), err
			},
			expected: "document_type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rows := newFakeRows(invoiceColumns, invoiceRow("A-1", 10, "2026-02-01", "FC"))

			n, err := tc.scan(rows)

			assert.ErrorIs(t, err, ErrUnmappedColumn)
			assert.ErrorContains(t, err, tc.expected)
			assert.Equal(t, 0, n, "no rows must be returned on a schema mismatch")
		})
	}
}

func TestScanAllMatchesCaseInsensitiveWithoutTag(t *testing.T) {
	type row struct {
		IdUnico NullString // matchea "idunico" por case-fold, como el driver
		Numero  NullInt64
	}
	rows := newFakeRows([]string{"idunico", "numero"},
		[]Value{NullString{StringVal: "A-1", Valid: true}, NullInt64{Int64: 7, Valid: true}},
	)

	got, err := ScanAll[row](rows)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "A-1", got[0].IdUnico.StringVal)
	assert.Equal(t, int64(7), got[0].Numero.Int64)
}

// Struct más ancha que la query: legítimo, el campo sobrante queda en zero.
func TestScanAllAllowsExtraStructFields(t *testing.T) {
	rows := newFakeRows([]string{"idunico"},
		[]Value{NullString{StringVal: "A-1", Valid: true}},
	)

	got, err := ScanAll[taggedRow](rows)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "A-1", got[0].Idunico.StringVal)
	assert.False(t, got[0].Numero.Valid)
}

func TestScanAllFlattensEmbeddedStructs(t *testing.T) {
	type base struct {
		Idunico NullString `bigquery:"idunico"`
	}
	type row struct {
		base
		Numero NullInt64 `bigquery:"numero"`
	}
	rows := newFakeRows([]string{"idunico", "numero"},
		[]Value{NullString{StringVal: "A-1", Valid: true}, NullInt64{Int64: 3, Valid: true}},
	)

	_, err := ScanAll[row](rows)

	assert.NoError(t, err)
}

func TestScanAllEmptyResultIsEmptySlice(t *testing.T) {
	got, err := ScanAll[taggedRow](newFakeRows(invoiceColumns))

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

// El driver real llena Schema recién después de la primera Next; el chequeo
// tiene que hacerse después de esa llamada, no antes.
func TestScanAllChecksSchemaAfterFirstNext(t *testing.T) {
	rows := newFakeRows(invoiceColumns, invoiceRow("A-1", 10, "2026-02-01", "FC"))
	rows.lazySchema = true

	_, err := ScanAll[untaggedRow](rows)

	assert.ErrorIs(t, err, ErrUnmappedColumn)
}

func TestScanAllPropagatesIteratorErrors(t *testing.T) {
	rows := newFakeRows(invoiceColumns)
	rows.nextErr = errBoom

	_, err := ScanAll[taggedRow](rows)

	assert.ErrorIs(t, err, errBoom)
}

func TestScanAllRejectsNonStruct(t *testing.T) {
	_, err := ScanAll[string](newFakeRows([]string{"x"}))

	assert.ErrorContains(t, err, "must be a struct")
}
