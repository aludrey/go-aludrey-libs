package bigquery

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	bq "cloud.google.com/go/bigquery"
)

// Carga de archivos CSV desde Cloud Storage hacia tablas de BigQuery. Las
// funciones de este archivo son puras (sin red) para poder testearlas; la
// ejecución contra GCP vive en loader.go.

var (
	ErrInvalidSourceURI = errors.New("bigquery: source URI must be gs://<bucket>/<object>")
	ErrEmptyDatasetID   = errors.New("bigquery: datasetID is required")
	ErrEmptyTableID     = errors.New("bigquery: tableID is required")
	ErrInvalidDelimiter = errors.New("bigquery: CSV delimiter must be a single printable character")
	ErrInvalidEncoding  = errors.New("bigquery: CSV encoding must be UTF-8 or ISO-8859-1")
	// ErrEmptyHeader se devuelve cuando el CSV no tiene fila de encabezado.
	ErrEmptyHeader = errors.New("bigquery: CSV has no header row")
	// ErrInvalidHeader se devuelve cuando la fila de encabezado no es CSV
	// válido (comillas sin cerrar, encabezado de más de 1 MiB, ...). Junto con
	// ErrEmptyHeader distingue un archivo defectuoso de una falla de red.
	ErrInvalidHeader = errors.New("bigquery: CSV header row is not valid CSV")
	// ErrLoadAttemptsExhausted se devuelve cuando todos los IDs de job
	// derivados de una misma versión del archivo ya fueron usados por cargas
	// que fallaron.
	ErrLoadAttemptsExhausted = errors.New("bigquery: load attempts exhausted for this file version")
)

const (
	// maxLoadAttempts acota los reintentos de una misma versión del archivo:
	// cada intento usa un ID de job distinto (ver loadJobID).
	maxLoadAttempts = 5
	// maxHeaderBytes acota cuánto se lee del objeto para obtener el encabezado.
	maxHeaderBytes = 1 << 20
	// maxColumnNameLength es el largo máximo de un nombre de columna en BigQuery.
	maxColumnNameLength = 300
	// maxJobErrorDetails acota cuántos errores de detalle de un job se
	// devuelven: BigQuery puede informar uno por fila rechazada.
	maxJobErrorDetails = 20
	loadJobIDPrefix    = "csv_load_"
	defaultDelimiter   = ','
	utf8BOM            = "\uFEFF"
)

// reservedColumnPrefixes son prefijos que BigQuery no admite en nombres de
// columna (comparación sin distinguir mayúsculas).
var reservedColumnPrefixes = []string{"_table_", "_file_", "_partition", "_row_timestamp", "__root__", "_colidentifier"}

// LoadCSVRequest describe la carga de un CSV de Cloud Storage en una tabla.
type LoadCSVRequest struct {
	// SourceURI del objeto, con formato gs://<bucket>/<object>.
	SourceURI string
	// DatasetID y TableID de la tabla destino. La tabla se crea si no existe;
	// si existe con datos la carga falla (nunca sobreescribe ni agrega).
	DatasetID string
	TableID   string
	// Delimiter del CSV; cero significa coma.
	Delimiter rune
	// Labels que se adjuntan al job (trazabilidad desde la consola de GCP).
	Labels map[string]string
	// Location del job (ej. "US", "southamerica-east1"). Vacío deja que
	// BigQuery la resuelva por la ubicación del dataset.
	Location string
	// Encoding del archivo; vacío significa UTF-8. Con UTF-8, un archivo que
	// no lo es (p. ej. un CSV exportado por Excel en Windows-1252) suele
	// cargarse igual pero con los acentos y la ñ alterados, sin error: para
	// esos archivos usar EncodingISO88591.
	Encoding Encoding
}

// Encoding es la codificación de caracteres de un CSV.
type Encoding string

const (
	EncodingUTF8 Encoding = "UTF-8"
	// EncodingISO88591 (Latin-1) también cubre en la práctica los CSV en
	// Windows-1252 con texto en español: difieren sólo en símbolos poco
	// usados (€, comillas tipográficas).
	EncodingISO88591 Encoding = "ISO-8859-1"
)

// JobRef identifica un job de BigQuery.
type JobRef struct {
	ID       string `json:"id"`
	Location string `json:"location,omitempty"`
}

// JobState es el estado de un job de BigQuery.
type JobState string

const (
	JobStateUnknown JobState = "UNKNOWN"
	JobStatePending JobState = "PENDING"
	JobStateRunning JobState = "RUNNING"
	JobStateDone    JobState = "DONE"
)

// JobInfo es la foto del estado de un job de carga.
type JobInfo struct {
	Ref   JobRef
	State JobState
	Err   string // vacío si el job no terminó o terminó bien
	// ErrorDetails son los errores puntuales que informó BigQuery (por fila
	// o columna, con su ubicación), hasta 20. Err suele ser un resumen
	// genérico ("too many errors"); el detalle útil para corregir el archivo
	// está acá.
	ErrorDetails []string
	DatasetID    string
	TableID      string
	OutputRows   int64
	Labels       map[string]string
	Created      time.Time
	Started      time.Time
	Ended        time.Time
}

// Succeeded indica si el job terminó sin error.
func (j JobInfo) Succeeded() bool {
	return j.State == JobStateDone && j.Err == ""
}

// LoadResult es el resultado de LoadCSVFromGCS.
type LoadResult struct {
	// Job es el job que carga (o ya cargó) esta versión del archivo.
	Job JobInfo
	// Columns son los nombres de columna (todas STRING) derivados del
	// encabezado del CSV.
	Columns []string
	// Reused es true cuando esta versión del archivo ya tenía un job
	// exitoso o en curso: no se lanzó uno nuevo y Job describe el existente,
	// cuya tabla destino puede diferir de la pedida.
	Reused bool
}

// parseGCSURI separa gs://bucket/object en sus partes.
func parseGCSURI(uri string) (bucket string, object string, err error) {
	rest, ok := strings.CutPrefix(uri, "gs://")
	if !ok {
		return "", "", ErrInvalidSourceURI
	}
	bucket, object, ok = strings.Cut(rest, "/")
	if !ok || bucket == "" || object == "" || strings.ContainsAny(object, "*") {
		return "", "", ErrInvalidSourceURI
	}
	return bucket, object, nil
}

// resolveDelimiter valida el delimitador y aplica el default.
func resolveDelimiter(delimiter rune) (rune, error) {
	if delimiter == 0 {
		return defaultDelimiter, nil
	}
	if delimiter == '"' || delimiter == '\r' || delimiter == '\n' || delimiter == utf8.RuneError || (delimiter < ' ' && delimiter != '\t') {
		return 0, ErrInvalidDelimiter
	}
	return delimiter, nil
}

// resolveEncoding valida la codificación; vacío significa UTF-8.
func resolveEncoding(encoding Encoding) (Encoding, error) {
	switch encoding {
	case "":
		return EncodingUTF8, nil
	case EncodingUTF8, EncodingISO88591:
		return encoding, nil
	default:
		return "", ErrInvalidEncoding
	}
}

// validateLoadRequest valida el request y devuelve bucket, objeto y delimitador.
func validateLoadRequest(req LoadCSVRequest) (bucket string, object string, delimiter rune, err error) {
	bucket, object, err = parseGCSURI(req.SourceURI)
	if err != nil {
		return "", "", 0, err
	}
	if req.DatasetID == "" {
		return "", "", 0, ErrEmptyDatasetID
	}
	if req.TableID == "" {
		return "", "", 0, ErrEmptyTableID
	}
	delimiter, err = resolveDelimiter(req.Delimiter)
	if err != nil {
		return "", "", 0, err
	}
	if _, err := resolveEncoding(req.Encoding); err != nil {
		return "", "", 0, err
	}
	return bucket, object, delimiter, nil
}

// readHeaderRecord lee la primera fila del CSV (respetando comillas) y
// descarta un BOM UTF-8 inicial. Un archivo ISO-8859-1 se convierte a UTF-8
// antes de leerlo, para que los nombres de columna salgan igual que si el
// archivo fuera UTF-8. Falla con ErrEmptyHeader si no hay encabezado y con
// ErrInvalidHeader si no es CSV válido.
func readHeaderRecord(r io.Reader, delimiter rune, encoding Encoding) ([]string, error) {
	var source io.Reader = io.LimitReader(r, maxHeaderBytes)
	if encoding == EncodingISO88591 {
		source = &latin1Reader{source: source}
	}
	reader := csv.NewReader(source)
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	// Comillas estrictas, igual que BigQuery al cargar: un encabezado con
	// comillas mal formadas falla acá con ErrInvalidHeader en lugar de
	// producir nombres de columna basura.

	record, err := reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, ErrEmptyHeader
	}
	if err != nil {
		var parseErr *csv.ParseError
		if errors.As(err, &parseErr) {
			return nil, fmt.Errorf("%w: %w", ErrInvalidHeader, err)
		}
		return nil, fmt.Errorf("bigquery: reading CSV header: %w", err)
	}
	record[0] = strings.TrimPrefix(record[0], utf8BOM)
	if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
		return nil, ErrEmptyHeader
	}
	return record, nil
}

// ColumnNames convierte un encabezado CSV en nombres de columna válidos para
// BigQuery: conserva letras ASCII, dígitos y '_', reemplaza el resto por '_',
// antepone '_' si empieza con dígito, nombra column_<n> a los vacíos, evita
// prefijos reservados y desambigua duplicados (sin distinguir mayúsculas)
// con sufijos _2, _3, ...
func ColumnNames(header []string) []string {
	names := make([]string, len(header))
	seen := make(map[string]bool, len(header))

	for i, raw := range header {
		name := sanitizeColumnName(raw)
		if name == "" {
			name = "column_" + strconv.Itoa(i+1)
		}
		unique := name
		for n := 2; seen[strings.ToLower(unique)]; n++ {
			suffix := "_" + strconv.Itoa(n)
			unique = truncate(name, maxColumnNameLength-len(suffix)) + suffix
		}
		seen[strings.ToLower(unique)] = true
		names[i] = unique
	}
	return names
}

func sanitizeColumnName(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	name := b.String()
	if strings.Trim(name, "_") == "" {
		return ""
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	lower := strings.ToLower(name)
	for _, prefix := range reservedColumnPrefixes {
		if strings.HasPrefix(lower, prefix) {
			name = "col" + name
			break
		}
	}
	return truncate(name, maxColumnNameLength)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

// stringSchema arma un schema con todas las columnas STRING y nullable.
func stringSchema(columns []string) Schema {
	schema := make(Schema, len(columns))
	for i, name := range columns {
		schema[i] = &FieldSchema{Name: name, Type: bq.StringFieldType}
	}
	return schema
}

// loadJobID deriva un ID de job determinístico de la versión exacta del
// objeto (bucket, nombre y generation) y del número de intento. BigQuery
// rechaza un ID repetido, lo que vuelve idempotente la carga: reintentar la
// misma versión del archivo encuentra el job existente en lugar de cargar dos
// veces. Una versión nueva del objeto (re-subida) cambia la generation y por
// lo tanto el ID.
func loadJobID(bucket string, object string, generation int64, attempt int) string {
	sum := sha256.Sum256([]byte(bucket + "/" + object + "#" + strconv.FormatInt(generation, 10)))
	return loadJobIDPrefix + hex.EncodeToString(sum[:16]) + "_" + strconv.Itoa(attempt)
}

// toJobState traduce el estado del driver.
func toJobState(state bq.State) JobState {
	switch state {
	case bq.Pending:
		return JobStatePending
	case bq.Running:
		return JobStateRunning
	case bq.Done:
		return JobStateDone
	default:
		return JobStateUnknown
	}
}

// errorDetails convierte los errores del job en mensajes legibles
// ("<ubicación>: <mensaje>"), sin repetir el error principal y acotados a
// maxJobErrorDetails.
func errorDetails(errs []*bq.Error, mainErr string) []string {
	details := make([]string, 0, min(len(errs), maxJobErrorDetails))
	for _, e := range errs {
		if e == nil || e.Message == "" {
			continue
		}
		detail := e.Message
		if e.Location != "" {
			detail = e.Location + ": " + e.Message
		}
		if detail == mainErr || e.Message == mainErr {
			continue
		}
		details = append(details, detail)
		if len(details) == maxJobErrorDetails {
			break
		}
	}
	if len(details) == 0 {
		return nil
	}
	return details
}

// toJobInfo arma JobInfo a partir de lo que devuelve el driver. errMsg viene
// aparte porque el error de JobStatus sólo es accesible vía su método Err().
func toJobInfo(ref JobRef, state bq.State, errMsg string, details []string, stats *bq.JobStatistics, config bq.JobConfig) JobInfo {
	info := JobInfo{Ref: ref, State: toJobState(state), Err: errMsg, ErrorDetails: details}

	if stats != nil {
		info.Created = stats.CreationTime
		info.Started = stats.StartTime
		info.Ended = stats.EndTime
		if load, ok := stats.Details.(*bq.LoadStatistics); ok {
			info.OutputRows = load.OutputRows
		}
	}
	if load, ok := config.(*bq.LoadConfig); ok {
		if load.Dst != nil {
			info.DatasetID = load.Dst.DatasetID
			info.TableID = load.Dst.TableID
		}
		info.Labels = load.Labels
	}
	return info
}

// latin1Reader convierte ISO-8859-1 a UTF-8: cada byte es el code point del
// mismo valor, así que los bytes >= 0x80 pasan a ocupar dos bytes.
type latin1Reader struct {
	source  io.Reader
	pending []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	if len(l.pending) == 0 {
		raw := make([]byte, max(len(p)/2, 1))
		n, err := l.source.Read(raw)
		for _, b := range raw[:n] {
			l.pending = utf8.AppendRune(l.pending, rune(b))
		}
		if len(l.pending) == 0 {
			return 0, err
		}
	}
	n := copy(p, l.pending)
	l.pending = l.pending[n:]
	return n, nil
}
