// Package bigquery encapsula el acceso a Google BigQuery: un provider con
// cliente cacheado, binding de parámetros, scan estricto de filas y builders
// puros de SQL (paginación, poda de tablas wildcard). El SQL de dominio queda
// siempre en el consumidor; acá vive sólo la mecánica.
package bigquery

import bq "cloud.google.com/go/bigquery"

// Aliases del driver para que el consumidor pueda declarar sus row structs
// sin importar cloud.google.com/go/bigquery directamente.
type (
	Schema        = bq.Schema
	FieldSchema   = bq.FieldSchema
	Value         = bq.Value
	NullString    = bq.NullString
	NullInt64     = bq.NullInt64
	NullFloat64   = bq.NullFloat64
	NullBool      = bq.NullBool
	NullTimestamp = bq.NullTimestamp
	NullDate      = bq.NullDate
	NullDateTime  = bq.NullDateTime
	NullTime      = bq.NullTime
)
