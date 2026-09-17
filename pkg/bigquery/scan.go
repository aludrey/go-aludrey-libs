package bigquery

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"google.golang.org/api/iterator"
)

// ErrUnmappedColumn se devuelve cuando el SELECT trae una columna que no
// tiene campo destino en la struct. Usar errors.Is; el mensaje lista las
// columnas huérfanas.
var ErrUnmappedColumn = errors.New("bigquery: select column has no destination field")

// Rows abstrae el iterador de filas del driver para poder fakearlo en unit
// tests sin red. El *bq.RowIterator real la implementa vía rowIterator.
type Rows interface {
	// Next carga la siguiente fila en dst (puntero a struct). Devuelve
	// iterator.Done cuando no quedan filas.
	Next(dst any) error
	// Schema devuelve el schema del resultado. En el driver real puede estar
	// vacío hasta después de la primera llamada a Next.
	Schema() Schema
}

// ScanAll materializa todas las filas de rows en []T.
//
// Es el reemplazo estricto de `for it.Next(&row)`. El loader del driver
// matchea columna→campo por nombre (exacto o ignorando mayúsculas) y una
// columna que no matchea la saltea en silencio dejando el campo en zero
// value. Acá, después de la primera fila (momento en que el schema está
// garantizado), se compara el schema contra los campos mapeables de T y si
// alguna columna del SELECT no encontró destino se devuelve
// ErrUnmappedColumn sin filas. Un campo de T sin columna en el SELECT es
// legítimo (la misma struct puede servir a proyecciones distintas) y queda
// en zero value.
//
// Los campos de T se mapean con las mismas reglas que el driver: campo
// exportado, tag `bigquery:"nombre"` si existe (`"-"` lo excluye), nombre
// del campo si no; structs anónimas embebidas se aplanan.
func ScanAll[T any](rows Rows) ([]T, error) {
	var zero T
	if reflect.TypeOf(zero).Kind() != reflect.Struct {
		return nil, fmt.Errorf("bigquery: ScanAll type parameter must be a struct, got %T", zero)
	}

	out := make([]T, 0)
	checked := false
	for {
		var row T
		err := rows.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if !checked {
			if err := checkSchema(rows.Schema(), reflect.TypeOf(zero)); err != nil {
				return nil, err
			}
			checked = true
		}
		out = append(out, row)
	}
	return out, nil
}

// checkSchema devuelve ErrUnmappedColumn si alguna columna del schema no
// tiene campo destino en t.
func checkSchema(schema Schema, t reflect.Type) error {
	fields := mappedFields(t)

	var missing []string
	for _, col := range schema {
		if col == nil {
			continue
		}
		if !fields.has(col.Name) {
			missing = append(missing, col.Name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("%w: struct %s has no field for column(s) %s",
		ErrUnmappedColumn, t.String(), strings.Join(missing, ", "))
}

// fieldNames es el conjunto de nombres de columna que una struct puede
// recibir, con los dos niveles de match del driver: exacto y case-fold.
type fieldNames struct {
	exact  map[string]struct{}
	folded map[string]struct{}
}

func (f fieldNames) has(column string) bool {
	if _, ok := f.exact[column]; ok {
		return true
	}
	_, ok := f.folded[strings.ToLower(column)]
	return ok
}

// mappedFields recorre t (y sus structs anónimas embebidas) recogiendo el
// nombre de columna de cada campo exportado y no excluido.
func mappedFields(t reflect.Type) fieldNames {
	names := fieldNames{
		exact:  map[string]struct{}{},
		folded: map[string]struct{}{},
	}
	collectFields(t, names)
	return names
}

func collectFields(t reflect.Type, names fieldNames) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for field := range t.Fields() {
		tag, hasTag := field.Tag.Lookup("bigquery")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}

		if field.Anonymous && !hasTag {
			ft := field.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				collectFields(ft, names)
				continue
			}
		}
		if !field.IsExported() {
			continue
		}
		if name == "" {
			name = field.Name
		}
		names.exact[name] = struct{}{}
		names.folded[strings.ToLower(name)] = struct{}{}
	}
}
