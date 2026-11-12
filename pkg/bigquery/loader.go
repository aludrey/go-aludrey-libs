package bigquery

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// Loader carga archivos CSV de Cloud Storage en tablas de BigQuery y consulta
// el estado de esos jobs. Es seguro para uso concurrente.
type Loader interface {
	// TableExists indica si la tabla existe.
	TableExists(ctx context.Context, datasetID string, tableID string) (bool, error)
	// LoadCSVFromGCS lanza (sin esperar) un job que carga el CSV en una
	// tabla con todas las columnas STRING, nombradas a partir del encabezado
	// (ver ColumnNames). La tabla se crea si no existe y la carga falla si
	// ya tiene datos. Es idempotente por versión del objeto: si esa versión
	// ya tiene un job exitoso o en curso devuelve ese job con Reused=true.
	LoadCSVFromGCS(ctx context.Context, req LoadCSVRequest) (LoadResult, error)
	// JobStatus devuelve el estado actual de un job.
	JobStatus(ctx context.Context, ref JobRef) (JobInfo, error)
	// WaitJob bloquea hasta que el job termina (o ctx se cancela) y devuelve
	// su estado final. Un job que terminó con error no es un error de
	// WaitJob: se informa en JobInfo.Err.
	WaitJob(ctx context.Context, ref JobRef) (JobInfo, error)
}

// Client reúne las queries (Provider) y las cargas (Loader) sobre un único
// juego de credenciales. Close libera ambos clientes.
type Client interface {
	Provider
	Loader
}

type client struct {
	*provider
	gcs *storage.Client
}

// NewClient construye, una única vez y con context.Background(), los
// clientes de BigQuery y de Cloud Storage con las credenciales del service
// account en memoria. Usar en lugar de NewProvider cuando además de leer se
// necesita cargar archivos.
func NewClient(projectID string, credentialsJSON string) (Client, error) {
	p, err := newProvider(projectID, credentialsJSON)
	if err != nil {
		return nil, err
	}

	gcs, err := storage.NewClient(context.Background(), option.WithCredentialsJSON([]byte(credentialsJSON)))
	if err != nil {
		_ = p.Close()
		return nil, fmt.Errorf("bigquery: creating storage client: %w", err)
	}

	return &client{provider: p, gcs: gcs}, nil
}

func (c *client) Close() error {
	return errors.Join(c.provider.Close(), c.gcs.Close())
}

func (c *client) TableExists(ctx context.Context, datasetID string, tableID string) (bool, error) {
	if datasetID == "" {
		return false, ErrEmptyDatasetID
	}
	if tableID == "" {
		return false, ErrEmptyTableID
	}

	_, err := c.client.Dataset(datasetID).Table(tableID).Metadata(ctx)
	if isHTTPStatus(err, http.StatusNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *client) LoadCSVFromGCS(ctx context.Context, req LoadCSVRequest) (LoadResult, error) {
	bucket, object, delimiter, err := validateLoadRequest(req)
	if err != nil {
		return LoadResult{}, err
	}

	attrs, err := c.gcs.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return LoadResult{}, fmt.Errorf("bigquery: reading object attributes: %w", err)
	}

	encoding, _ := resolveEncoding(req.Encoding) // ya validada
	columns, err := c.headerColumns(ctx, bucket, object, attrs.Generation, delimiter, encoding)
	if err != nil {
		return LoadResult{}, err
	}

	for attempt := 1; attempt <= maxLoadAttempts; attempt++ {
		ref := JobRef{ID: loadJobID(bucket, object, attrs.Generation, attempt), Location: req.Location}

		job, err := c.startLoad(ctx, req, ref, columns, delimiter, encoding)
		if err == nil {
			return LoadResult{Job: jobInfoFromJob(ref, job), Columns: columns}, nil
		}
		if !isHTTPStatus(err, http.StatusConflict) {
			return LoadResult{}, fmt.Errorf("bigquery: starting load job: %w", err)
		}

		// El ID ya existe: esta versión del archivo ya tuvo un job.
		existing, err := c.JobStatus(ctx, ref)
		if err != nil {
			return LoadResult{}, err
		}
		if !existing.Succeeded() && existing.State == JobStateDone {
			continue // falló: el próximo intento usa otro ID
		}
		return LoadResult{Job: existing, Columns: columns, Reused: true}, nil
	}

	return LoadResult{}, ErrLoadAttemptsExhausted
}

func (c *client) JobStatus(ctx context.Context, ref JobRef) (JobInfo, error) {
	job, err := c.client.JobFromIDLocation(ctx, ref.ID, ref.Location)
	if err != nil {
		return JobInfo{}, fmt.Errorf("bigquery: getting job %s: %w", ref.ID, err)
	}
	return jobInfoFromJob(ref, job), nil
}

func (c *client) WaitJob(ctx context.Context, ref JobRef) (JobInfo, error) {
	job, err := c.client.JobFromIDLocation(ctx, ref.ID, ref.Location)
	if err != nil {
		return JobInfo{}, fmt.Errorf("bigquery: getting job %s: %w", ref.ID, err)
	}
	if _, err := job.Wait(ctx); err != nil {
		return JobInfo{}, fmt.Errorf("bigquery: waiting for job %s: %w", ref.ID, err)
	}
	return jobInfoFromJob(ref, job), nil
}

// headerColumns lee el encabezado de la versión exacta del objeto y lo
// convierte en nombres de columna.
func (c *client) headerColumns(ctx context.Context, bucket string, object string, generation int64, delimiter rune, encoding Encoding) ([]string, error) {
	reader, err := c.gcs.Bucket(bucket).Object(object).Generation(generation).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery: opening object: %w", err)
	}
	defer reader.Close()

	header, err := readHeaderRecord(reader, delimiter, encoding)
	if err != nil {
		return nil, err
	}
	return ColumnNames(header), nil
}

func (c *client) startLoad(ctx context.Context, req LoadCSVRequest, ref JobRef, columns []string, delimiter rune, encoding Encoding) (*bq.Job, error) {
	gcsRef := bq.NewGCSReference(req.SourceURI)
	gcsRef.SourceFormat = bq.CSV
	gcsRef.FieldDelimiter = string(delimiter)
	gcsRef.SkipLeadingRows = 1
	gcsRef.AllowQuotedNewlines = true
	gcsRef.Encoding = bq.Encoding(encoding)
	gcsRef.Schema = stringSchema(columns)

	loader := c.client.Dataset(req.DatasetID).Table(req.TableID).LoaderFrom(gcsRef)
	loader.JobID = ref.ID
	loader.Location = ref.Location
	loader.Labels = req.Labels
	loader.CreateDisposition = bq.CreateIfNeeded
	loader.WriteDisposition = bq.WriteEmpty

	return loader.Run(ctx)
}

// jobInfoFromJob usa el último estado conocido del job (sin otra llamada).
func jobInfoFromJob(ref JobRef, job *bq.Job) JobInfo {
	status := job.LastStatus()
	config, _ := job.Config() // sólo falla para tipos de job desconocidos: la info de tabla queda vacía

	var state bq.State
	var stats *bq.JobStatistics
	var details []string
	errMsg := ""
	if status != nil {
		state = status.State
		stats = status.Statistics
		if err := status.Err(); err != nil {
			errMsg = jobErrorMessage(err)
		}
		details = errorDetails(status.Errors, errMsg)
	}
	return toJobInfo(JobRef{ID: job.ID(), Location: firstNonEmpty(job.Location(), ref.Location)}, state, errMsg, details, stats, config)
}

// jobErrorMessage devuelve el mensaje del error principal del job sin el
// envoltorio de depuración del driver ({Location: ...; Message: ...}).
func jobErrorMessage(err error) string {
	var bqErr *bq.Error
	if errors.As(err, &bqErr) && bqErr.Message != "" {
		if bqErr.Location != "" {
			return bqErr.Location + ": " + bqErr.Message
		}
		return bqErr.Message
	}
	return err.Error()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func isHTTPStatus(err error, code int) bool {
	var apiErr *googleapi.Error
	return errors.As(err, &apiErr) && apiErr.Code == code
}
