package bigquery

import (
	"errors"
	"strings"
	"time"
)

// Builders puros de SQL. Ninguna función de este archivo ejecuta nada ni toca
// el cliente: devuelven strings y params para que el consumidor pueda
// asertar el SQL final en unit tests sin red.

// Nombres de los parámetros que agrega Paginate.
const (
	PageSizeParam = "pageSize"
	OffsetParam   = "offset"
)

var (
	ErrInvalidPage     = errors.New("bigquery: page must be >= 1")
	ErrInvalidPageSize = errors.New("bigquery: pageSize must be >= 1")
)

// WhereClause une las condiciones con AND y antepone "WHERE ". Las
// condiciones vacías se ignoran; sin condiciones devuelve "" para que el
// SQL no quede con un WHERE colgante.
func WhereClause(conditions ...string) string {
	kept := make([]string, 0, len(conditions))
	for _, c := range conditions {
		if c = strings.TrimSpace(c); c != "" {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return "WHERE " + strings.Join(kept, " AND ")
}

// CountSQL arma la query de total: "SELECT COUNT(*) FROM <table> <where>".
// where es la cláusula completa (con "WHERE ") tal como la devuelve
// WhereClause, o "" para contar toda la tabla.
func CountSQL(table string, where string) string {
	return "SELECT COUNT(*) FROM " + table + spaced(where)
}

// Paginate agrega LIMIT/OFFSET parametrizados al SQL y devuelve los params
// correspondientes (PageSizeParam y OffsetParam). page es 1-based.
func Paginate(sql string, page int, pageSize int) (string, []Param, error) {
	if page < 1 {
		return "", nil, ErrInvalidPage
	}
	if pageSize < 1 {
		return "", nil, ErrInvalidPageSize
	}

	paged := sql + " LIMIT @" + PageSizeParam + " OFFSET @" + OffsetParam
	params := []Param{
		P(PageSizeParam, int64(pageSize)),
		P(OffsetParam, int64(page-1)*int64(pageSize)),
	}
	return paged, params, nil
}

// spaced antepone el espacio separador a una cláusula no vacía.
func spaced(clause string) string {
	if clause == "" {
		return ""
	}
	return " " + clause
}

// tableSuffixLayout es el formato del sufijo de una tabla diaria (YYYYMMDD).
const tableSuffixLayout = "20060102"

// tableSuffixDigits es el ancho exacto del sufijo. Es el guard del único
// valor que se interpola en el texto SQL.
const tableSuffixDigits = 8

// TableSuffixCondition arma el predicado de poda de una tabla wildcard a
// partir de bounds de fecha. Un time.Time zero significa "sin bound".
// Devuelve pruned=false cuando no hay ningún bound válido.
//
// Los valores van interpolados como literales a propósito: BigQuery sólo
// poda las tablas del wildcard cuando el predicado sobre _TABLE_SUFFIX es
// una expresión constante. Con @param compila, pero escanea todas las tablas
// diarias. Los literales salen únicamente de TableSuffix, que garantiza
// exactamente 8 dígitos, así que no hay superficie de inyección.
func TableSuffixCondition(from time.Time, to time.Time) (condition string, pruned bool) {
	fromSuffix, hasFrom := TableSuffix(from)
	toSuffix, hasTo := TableSuffix(to)

	switch {
	case hasFrom && hasTo:
		return "_TABLE_SUFFIX BETWEEN '" + fromSuffix + "' AND '" + toSuffix + "'", true
	case hasFrom:
		return "_TABLE_SUFFIX >= '" + fromSuffix + "'", true
	case hasTo:
		return "_TABLE_SUFFIX <= '" + toSuffix + "'", true
	}
	return "", false
}

// TableSuffix formatea una fecha como sufijo YYYYMMDD de su tabla diaria.
// Devuelve ok=false para una fecha zero o para cualquier resultado que no
// sea exactamente 8 dígitos; descartar el bound sólo ensancha la búsqueda,
// nunca corrompe la query.
func TableSuffix(date time.Time) (suffix string, ok bool) {
	if date.IsZero() {
		return "", false
	}
	suffix = date.Format(tableSuffixLayout)
	if !isTableSuffix(suffix) {
		return "", false
	}
	return suffix, true
}

// isTableSuffix reporta si value son exactamente 8 dígitos ASCII.
func isTableSuffix(value string) bool {
	if len(value) != tableSuffixDigits {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}
