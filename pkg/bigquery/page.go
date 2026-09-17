package bigquery

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Page es el resultado de una consulta paginada: el total del conjunto
// filtrado y las filas de la página pedida.
type Page[T any] struct {
	Total int64
	Rows  []T
}

// PageOption configura QueryPage.
type PageOption func(*pageOptions)

type pageOptions struct {
	parallel bool
}

// Parallel hace que QueryPage corra el COUNT y la page query concurrentes.
// Baja la latencia, pero pierde el skip de la page query cuando el total es
// cero — y BigQuery cobra los bytes leídos aunque el resultado esté vacío.
// Usarlo sólo donde el total es casi siempre > 0 (p. ej. dashboards).
func Parallel() PageOption {
	return func(o *pageOptions) { o.parallel = true }
}

// QueryPage ejecuta countSQL y pageSQL contra p y devuelve total + filas ya
// escaneadas con ScanAll (o sea, con chequeo estricto de schema).
//
// countSQL recibe filterParams; pageSQL recibe filterParams + pageParams
// (típicamente los que devuelve Paginate). Ambos SQL llegan ya armados: el
// armado es responsabilidad del consumidor, así puede asertarlo en tests sin
// red.
//
// Por default corre secuencial y, si el total es 0, NO ejecuta la page query:
// esa es la cara (lee columnas sobre todo el rango de tablas) y devolvería
// cero filas. Con Parallel() corre ambas a la vez.
func QueryPage[T any](ctx context.Context, p Provider, countSQL string, pageSQL string, filterParams []Param, pageParams []Param, opts ...PageOption) (Page[T], error) {
	var o pageOptions
	for _, opt := range opts {
		opt(&o)
	}

	allParams := make([]Param, 0, len(filterParams)+len(pageParams))
	allParams = append(allParams, filterParams...)
	allParams = append(allParams, pageParams...)

	if o.parallel {
		return queryPageParallel[T](ctx, p, countSQL, pageSQL, filterParams, allParams)
	}

	total, err := p.Count(ctx, countSQL, filterParams)
	if err != nil {
		return Page[T]{}, err
	}
	if total == 0 {
		return Page[T]{Total: 0, Rows: []T{}}, nil
	}

	rows, err := readAll[T](ctx, p, pageSQL, allParams)
	if err != nil {
		return Page[T]{}, err
	}
	return Page[T]{Total: total, Rows: rows}, nil
}

func queryPageParallel[T any](ctx context.Context, p Provider, countSQL string, pageSQL string, filterParams []Param, pageParams []Param) (Page[T], error) {
	var page Page[T]
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		total, err := p.Count(gctx, countSQL, filterParams)
		if err != nil {
			return err
		}
		page.Total = total
		return nil
	})
	g.Go(func() error {
		rows, err := readAll[T](gctx, p, pageSQL, pageParams)
		if err != nil {
			return err
		}
		page.Rows = rows
		return nil
	})

	if err := g.Wait(); err != nil {
		return Page[T]{}, err
	}
	return page, nil
}

// readAll es Read + ScanAll.
func readAll[T any](ctx context.Context, p Provider, sql string, params []Param) ([]T, error) {
	rows, err := p.Read(ctx, sql, params)
	if err != nil {
		return nil, err
	}
	return ScanAll[T](rows)
}
