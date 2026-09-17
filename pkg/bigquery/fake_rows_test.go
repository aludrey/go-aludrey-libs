package bigquery

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/api/iterator"
)

// fakeRows es un Rows en memoria que imita al loader del driver: matchea
// columna→campo por tag exacto o case-insensitive y SALTEA en silencio las
// columnas sin destino. Ese es justamente el comportamiento que ScanAll
// tiene que convertir en error.
type fakeRows struct {
	schema     Schema
	rows       [][]Value
	pos        int
	nextErr    error // si está seteado, Next lo devuelve en la primera llamada
	lazySchema bool  // schema vacío hasta la primera Next, como el driver real
}

func newFakeRows(columns []string, rows ...[]Value) *fakeRows {
	schema := make(Schema, len(columns))
	for i, c := range columns {
		schema[i] = &FieldSchema{Name: c}
	}
	return &fakeRows{schema: schema, rows: rows}
}

func (f *fakeRows) Schema() Schema {
	if f.lazySchema && f.pos == 0 {
		return nil
	}
	return f.schema
}

func (f *fakeRows) Next(dst any) error {
	if f.nextErr != nil {
		return f.nextErr
	}
	if f.pos >= len(f.rows) {
		return iterator.Done
	}
	row := f.rows[f.pos]
	f.pos++

	switch d := dst.(type) {
	case *[]Value:
		*d = append([]Value(nil), row...)
		return nil
	}

	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fakeRows: unsupported dst %T", dst)
	}
	target := v.Elem()
	for i, col := range f.schema {
		field, ok := findField(target, col.Name)
		if !ok {
			continue // el driver real hace exactamente esto
		}
		if i >= len(row) || row[i] == nil {
			continue
		}
		val := reflect.ValueOf(row[i])
		if !val.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("fakeRows: cannot assign %T to field of type %s", row[i], field.Type())
		}
		field.Set(val)
	}
	return nil
}

func findField(target reflect.Value, column string) (reflect.Value, bool) {
	t := target.Type()
	var folded reflect.Value
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(sf.Tag.Get("bigquery"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = sf.Name
		}
		if name == column {
			return target.Field(i), true
		}
		if !folded.IsValid() && strings.EqualFold(name, column) {
			folded = target.Field(i)
		}
	}
	return folded, folded.IsValid()
}

var errBoom = errors.New("boom")
