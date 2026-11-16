package bigquery

import (
	"context"
	"errors"
	"fmt"
	"time"

	bq "cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	ErrEmptyProjectID   = errors.New("bigquery: projectID is required")
	ErrEmptyCredentials = errors.New("bigquery: credentialsJSON is required")
	// ErrNoRows se devuelve cuando una query escalar no produce ninguna fila.
	ErrNoRows = errors.New("bigquery: query returned no rows")
	// ErrNotScalar se devuelve cuando la primera fila de Count no es una
	// única columna INT64.
	ErrNotScalar = errors.New("bigquery: query did not return a single INT64 column")
	// ErrNotDML se devuelve cuando Exec ejecuta algo que no es un statement
	// DML (INSERT, UPDATE, DELETE, MERGE) o el job no informa estadísticas de
	// query, así que no hay cantidad de filas afectadas para devolver.
	ErrNotDML = errors.New("bigquery: statement is not DML")
)

// dmlStatementTypes son los StatementType que BigQuery informa para DML.
var dmlStatementTypes = map[string]bool{
	"INSERT": true,
	"UPDATE": true,
	"DELETE": true,
	"MERGE":  true,
}

// Provider ejecuta queries sobre BigQuery con un cliente construido una sola
// vez y compartido entre llamadas. Es seguro para uso concurrente.
type Provider interface {
	// Read ejecuta sql con params y devuelve las filas sin materializar.
	// Combinar con ScanAll para obtener []T con chequeo de schema.
	Read(ctx context.Context, sql string, params []Param) (Rows, error)
	// Count ejecuta una query escalar (COUNT(*), COUNT(DISTINCT x), ...) y
	// devuelve la única columna de la primera fila como int64. Falla con
	// ErrNoRows si no hay fila y con ErrNotScalar si la forma no es la
	// esperada; nunca devuelve 0 por un type assertion silencioso.
	Count(ctx context.Context, sql string, params []Param) (int64, error)
	// Exec ejecuta un statement DML (INSERT, UPDATE, DELETE, MERGE) con
	// params, espera a que el job termine y devuelve la cantidad de filas
	// afectadas. Un job fallido devuelve su error; un statement que no es DML
	// devuelve ErrNotDML. Con 0 filas afectadas el consumidor puede
	// distinguir "no existe" o "ya existe" sin una lectura previa.
	Exec(ctx context.Context, sql string, params []Param) (int64, error)
	// Insert inserta una fila en table (referencia completa, con backticks),
	// una columna por Param de values. Si values incluye created_at y/o
	// created_by, la lib pone su valor: la hora UTC y el usuario de
	// WithCreatedBy (NULL si no hay). Devuelve esos valores.
	Insert(ctx context.Context, table string, values []Param) (InsertResult, error)
	// Close libera el cliente. Llamar en el graceful shutdown del servicio.
	Close() error
}

type provider struct {
	client *bq.Client
	// now es el reloj de created_at.
	now func() time.Time
}

// NewProvider construye el cliente una única vez, con context.Background() y
// las credenciales del service account en memoria (JSON crudo, tal como sale
// de Secrets Manager — ver CredentialsFromSecret), y lo retiene. El key
// nunca toca disco.
func NewProvider(projectID string, credentialsJSON string) (Provider, error) {
	p, err := newProvider(projectID, credentialsJSON)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func newProvider(projectID string, credentialsJSON string) (*provider, error) {
	if projectID == "" {
		return nil, ErrEmptyProjectID
	}
	if credentialsJSON == "" {
		return nil, ErrEmptyCredentials
	}

	client, err := bq.NewClient(context.Background(), projectID, option.WithCredentialsJSON([]byte(credentialsJSON)))
	if err != nil {
		return nil, fmt.Errorf("bigquery: creating client: %w", err)
	}

	return &provider{client: client, now: time.Now}, nil
}

func (p *provider) Read(ctx context.Context, sql string, params []Param) (Rows, error) {
	query := p.client.Query(sql)
	query.Parameters = toQueryParameters(params)

	it, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}
	return rowIterator{it: it}, nil
}

func (p *provider) Count(ctx context.Context, sql string, params []Param) (int64, error) {
	rows, err := p.Read(ctx, sql, params)
	if err != nil {
		return 0, err
	}
	return scalarInt64(rows)
}

func (p *provider) Exec(ctx context.Context, sql string, params []Param) (int64, error) {
	return p.runDML(ctx, sql, params)
}

func (p *provider) Insert(ctx context.Context, table string, values []Param) (InsertResult, error) {
	result := InsertResult{
		CreatedAt: p.now().UTC().Truncate(time.Microsecond),
		CreatedBy: CreatedByFrom(ctx),
	}
	sql, params, err := buildInsert(table, values, result)
	if err != nil {
		return InsertResult{}, err
	}
	if _, err := p.runDML(ctx, sql, params); err != nil {
		return InsertResult{}, err
	}
	return result, nil
}

// runDML ejecuta un statement DML, espera el job y devuelve las filas
// afectadas.
func (p *provider) runDML(ctx context.Context, sql string, params []Param) (int64, error) {
	query := p.client.Query(sql)
	query.Parameters = toQueryParameters(params)

	job, err := query.Run(ctx)
	if err != nil {
		return 0, err
	}
	// Wait devuelve nil aunque el job haya fallado: el error del job viaja en
	// status.Err(), que affectedRows chequea.
	status, err := job.Wait(ctx)
	if err != nil {
		return 0, err
	}
	return affectedRows(status)
}

func (p *provider) Close() error {
	return p.client.Close()
}

// affectedRows interpreta el estado final de un job DML. Separado de Exec
// para poder testearlo sin red.
func affectedRows(status *bq.JobStatus) (int64, error) {
	if status == nil {
		return 0, ErrNotDML
	}
	if err := status.Err(); err != nil {
		return 0, err
	}
	if status.Statistics == nil {
		return 0, ErrNotDML
	}
	stats, ok := status.Statistics.Details.(*bq.QueryStatistics)
	if !ok || stats == nil {
		return 0, ErrNotDML
	}
	if !dmlStatementTypes[stats.StatementType] {
		return 0, fmt.Errorf("%w: got %q", ErrNotDML, stats.StatementType)
	}
	return stats.NumDMLAffectedRows, nil
}

// scalarInt64 lee la primera fila de rows y la interpreta como un único
// INT64. Separado de Count para poder testearlo con un Rows fake.
func scalarInt64(rows Rows) (int64, error) {
	var row []Value
	if err := rows.Next(&row); err != nil {
		if errors.Is(err, iterator.Done) {
			return 0, ErrNoRows
		}
		return 0, err
	}
	if len(row) != 1 {
		return 0, fmt.Errorf("%w: got %d columns", ErrNotScalar, len(row))
	}
	count, ok := row[0].(int64)
	if !ok {
		return 0, fmt.Errorf("%w: got %T", ErrNotScalar, row[0])
	}
	return count, nil
}

// rowIterator adapta *bq.RowIterator a Rows.
type rowIterator struct {
	it *bq.RowIterator
}

func (r rowIterator) Next(dst any) error { return r.it.Next(dst) }
func (r rowIterator) Schema() Schema     { return r.it.Schema }
