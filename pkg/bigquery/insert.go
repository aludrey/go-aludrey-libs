package bigquery

import (
	"context"
	"errors"
	"strings"
	"time"

	bq "cloud.google.com/go/bigquery"
)

// Columnas de auditoría. Si un Insert las incluye, el valor lo pone la lib.
const (
	CreatedAtColumn = "created_at"
	CreatedByColumn = "created_by"
)

// ErrNoInsertValues se devuelve cuando Insert no recibe ninguna columna.
var ErrNoInsertValues = errors.New("bigquery: insert needs at least one value")

// InsertResult son los valores de auditoría de Insert.
type InsertResult struct {
	// CreatedAt en UTC, con la precisión de microsegundos de un TIMESTAMP.
	CreatedAt time.Time
	// CreatedBy; "" significa NULL.
	CreatedBy string
}

type createdByKey struct{}

// WithCreatedBy devuelve un context que lleva el usuario que Insert guarda en
// created_by. Un usuario vacío se ignora y created_by queda NULL.
func WithCreatedBy(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, createdByKey{}, strings.TrimSpace(user))
}

// CreatedByFrom devuelve el usuario cargado con WithCreatedBy, o "" si no
// hay. Sirve también a los fakes de Provider de los consumidores.
func CreatedByFrom(ctx context.Context) string {
	user, _ := ctx.Value(createdByKey{}).(string)
	return user
}

// buildInsert arma "INSERT INTO table (cols...) VALUES (@cols...)" con una
// columna por Param. Los nombres de columna se interpolan tal cual: salen del
// código del consumidor, nunca de input externo. Si values incluye created_at
// o created_by, su valor se reemplaza por el de audit. values no se modifica.
// Función pura, separada de Insert para poder testearla sin red.
func buildInsert(table string, values []Param, audit InsertResult) (string, []Param, error) {
	if len(values) == 0 {
		return "", nil, ErrNoInsertValues
	}

	columns := make([]string, len(values))
	params := make([]Param, len(values))
	for i, v := range values {
		switch v.Name {
		case CreatedAtColumn:
			v.Value = audit.CreatedAt
		case CreatedByColumn:
			v.Value = bq.NullString{StringVal: audit.CreatedBy, Valid: audit.CreatedBy != ""}
		}
		columns[i] = v.Name
		params[i] = v
	}

	sql := "INSERT INTO " + table + " (" + strings.Join(columns, ", ") + ") " +
		"VALUES (@" + strings.Join(columns, ", @") + ")"
	return sql, params, nil
}
