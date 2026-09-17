package bigquery

import bq "cloud.google.com/go/bigquery"

// Param es un parámetro nombrado de query. Todo valor de filtro debe viajar
// como Param (referenciado en el SQL como @name), nunca interpolado en el
// texto; la única excepción documentada es el literal de _TABLE_SUFFIX, que
// sale exclusivamente de TableSuffix.
type Param struct {
	Name  string
	Value any
}

// P construye un Param.
func P(name string, value any) Param {
	return Param{Name: name, Value: value}
}

func toQueryParameters(params []Param) []bq.QueryParameter {
	if len(params) == 0 {
		return nil
	}
	out := make([]bq.QueryParameter, len(params))
	for i, p := range params {
		out[i] = bq.QueryParameter{Name: p.Name, Value: p.Value}
	}
	return out
}
