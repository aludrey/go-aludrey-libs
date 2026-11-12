package bigquery

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	bq "cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestParseGCSURI(t *testing.T) {
	testCases := []struct {
		name           string
		uri            string
		expectedBucket string
		expectedObject string
		expectedErr    error
	}{
		{name: "object at root", uri: "gs://bucket/file.csv", expectedBucket: "bucket", expectedObject: "file.csv"},
		{name: "object with prefix", uri: "gs://bucket/pending/a b.csv", expectedBucket: "bucket", expectedObject: "pending/a b.csv"},
		{name: "missing scheme", uri: "bucket/file.csv", expectedErr: ErrInvalidSourceURI},
		{name: "s3 scheme", uri: "s3://bucket/file.csv", expectedErr: ErrInvalidSourceURI},
		{name: "missing object", uri: "gs://bucket/", expectedErr: ErrInvalidSourceURI},
		{name: "missing bucket", uri: "gs:///file.csv", expectedErr: ErrInvalidSourceURI},
		{name: "no slash", uri: "gs://bucket", expectedErr: ErrInvalidSourceURI},
		{name: "wildcard", uri: "gs://bucket/*.csv", expectedErr: ErrInvalidSourceURI},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bucket, object, err := parseGCSURI(tc.uri)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expectedBucket, bucket)
			assert.Equal(t, tc.expectedObject, object)
		})
	}
}

func TestResolveDelimiter(t *testing.T) {
	testCases := []struct {
		name        string
		in          rune
		expected    rune
		expectedErr error
	}{
		{name: "default", in: 0, expected: ','},
		{name: "semicolon", in: ';', expected: ';'},
		{name: "pipe", in: '|', expected: '|'},
		{name: "tab", in: '\t', expected: '\t'},
		{name: "quote", in: '"', expectedErr: ErrInvalidDelimiter},
		{name: "newline", in: '\n', expectedErr: ErrInvalidDelimiter},
		{name: "carriage return", in: '\r', expectedErr: ErrInvalidDelimiter},
		{name: "control char", in: '\x01', expectedErr: ErrInvalidDelimiter},
		{name: "rune error", in: '\uFFFD', expectedErr: ErrInvalidDelimiter},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveDelimiter(tc.in)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestValidateLoadRequest(t *testing.T) {
	valid := LoadCSVRequest{SourceURI: "gs://b/o.csv", DatasetID: "ds", TableID: "t"}

	testCases := []struct {
		name        string
		mutate      func(*LoadCSVRequest)
		expectedErr error
	}{
		{name: "valid", mutate: func(*LoadCSVRequest) {}},
		{name: "bad uri", mutate: func(r *LoadCSVRequest) { r.SourceURI = "x" }, expectedErr: ErrInvalidSourceURI},
		{name: "no dataset", mutate: func(r *LoadCSVRequest) { r.DatasetID = "" }, expectedErr: ErrEmptyDatasetID},
		{name: "no table", mutate: func(r *LoadCSVRequest) { r.TableID = "" }, expectedErr: ErrEmptyTableID},
		{name: "bad delimiter", mutate: func(r *LoadCSVRequest) { r.Delimiter = '"' }, expectedErr: ErrInvalidDelimiter},
		{name: "latin-1", mutate: func(r *LoadCSVRequest) { r.Encoding = EncodingISO88591 }},
		{name: "explicit utf-8", mutate: func(r *LoadCSVRequest) { r.Encoding = EncodingUTF8 }},
		{name: "bad encoding", mutate: func(r *LoadCSVRequest) { r.Encoding = "UTF-16" }, expectedErr: ErrInvalidEncoding},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := valid
			tc.mutate(&req)

			bucket, object, delimiter, err := validateLoadRequest(req)

			assert.ErrorIs(t, err, tc.expectedErr)
			if tc.expectedErr == nil {
				assert.Equal(t, "b", bucket)
				assert.Equal(t, "o.csv", object)
				assert.Equal(t, ',', delimiter)
			}
		})
	}
}

func TestReadHeaderRecord(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		delimiter   rune
		expected    []string
		expectedErr error
	}{
		{name: "simple", content: "a,b,c\n1,2,3\n", delimiter: ',', expected: []string{"a", "b", "c"}},
		{name: "semicolon", content: "a;b\n1;2\n", delimiter: ';', expected: []string{"a", "b"}},
		{name: "quoted with delimiter and newline", content: "\"a,1\",\"b\nx\"\n1,2\n", delimiter: ',', expected: []string{"a,1", "b\nx"}},
		{name: "bom removed", content: "\uFEFFid,name\n", delimiter: ',', expected: []string{"id", "name"}},
		{name: "header only no newline", content: "id", delimiter: ',', expected: []string{"id"}},
		{name: "crlf", content: "a,b\r\n1,2\r\n", delimiter: ',', expected: []string{"a", "b"}},
		{name: "empty file", content: "", delimiter: ',', expectedErr: ErrEmptyHeader},
		{name: "blank header", content: "   \n1\n", delimiter: ',', expectedErr: ErrEmptyHeader},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readHeaderRecord(strings.NewReader(tc.content), tc.delimiter, EncodingUTF8)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestReadHeaderRecordRejectsMalformedHeader(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{name: "unterminated quote", content: "\"id,name\n1,2\n"},
		{name: "bare quote", content: "i\"d,name\n"},
		{name: "header over the size limit", content: "\"" + strings.Repeat("x", maxHeaderBytes+10)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := readHeaderRecord(strings.NewReader(tc.content), ',', EncodingUTF8)

			assert.ErrorIs(t, err, ErrInvalidHeader)
		})
	}
}

func TestReadHeaderRecordWrapsReaderErrors(t *testing.T) {
	_, err := readHeaderRecord(failingReader{}, ',', EncodingUTF8)

	assert.ErrorContains(t, err, "reading CSV header")
}

func TestResolveEncoding(t *testing.T) {
	testCases := []struct {
		in          Encoding
		expected    Encoding
		expectedErr error
	}{
		{in: "", expected: EncodingUTF8},
		{in: EncodingUTF8, expected: EncodingUTF8},
		{in: EncodingISO88591, expected: EncodingISO88591},
		{in: "latin1", expectedErr: ErrInvalidEncoding},
		{in: "utf-8", expectedErr: ErrInvalidEncoding},
	}

	for _, tc := range testCases {
		t.Run(string(tc.in), func(t *testing.T) {
			got, err := resolveEncoding(tc.in)

			assert.ErrorIs(t, err, tc.expectedErr)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// A Latin-1 header is decoded before parsing: the names come out as they
// would from the same header saved as UTF-8.
func TestReadHeaderRecordLatin1(t *testing.T) {
	latin1 := "A\xf1o;Descripci\xf3n;Pa\xeds\n2026;x;AR\n" // "Año;Descripción;País"

	got, err := readHeaderRecord(strings.NewReader(latin1), ';', EncodingISO88591)
	require.NoError(t, err)
	assert.Equal(t, []string{"Año", "Descripción", "País"}, got)

	utf8Header, err := readHeaderRecord(strings.NewReader("Año;Descripción;País\n"), ';', EncodingUTF8)
	require.NoError(t, err)
	assert.Equal(t, ColumnNames(utf8Header), ColumnNames(got))
}

// The decoder works through reads of any size, including one byte at a time.
func TestLatin1ReaderSmallReads(t *testing.T) {
	reader := &latin1Reader{source: strings.NewReader("\xd1and\xfa")}
	var out []byte
	buf := make([]byte, 1)
	for {
		n, err := reader.Read(buf)
		out = append(out, buf[:n]...)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	assert.Equal(t, "Ñandú", string(out))
}

func TestLatin1ReaderPropagatesErrors(t *testing.T) {
	_, err := readHeaderRecord(failingReader{}, ',', EncodingISO88591)

	assert.ErrorContains(t, err, "boom")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestColumnNames(t *testing.T) {
	long := strings.Repeat("x", 310)

	testCases := []struct {
		name     string
		header   []string
		expected []string
	}{
		{name: "kept as is", header: []string{"Id", "Fecha_Op", "importe"}, expected: []string{"Id", "Fecha_Op", "importe"}},
		{name: "invalid chars", header: []string{"Fecha Op", "Importe ($)", "año"}, expected: []string{"Fecha_Op", "Importe____", "a_o"}},
		{name: "trimmed", header: []string{"  id  "}, expected: []string{"id"}},
		{name: "leading digit", header: []string{"2026"}, expected: []string{"_2026"}},
		{name: "empty and symbols only", header: []string{"", "---", "a"}, expected: []string{"column_1", "column_2", "a"}},
		{name: "duplicates case insensitive", header: []string{"id", "ID", "Id"}, expected: []string{"id", "ID_2", "Id_3"}},
		{name: "duplicate after sanitize", header: []string{"a b", "a-b"}, expected: []string{"a_b", "a_b_2"}},
		{name: "generated name collides", header: []string{"column_2", ""}, expected: []string{"column_2", "column_2_2"}},
		{name: "reserved prefix", header: []string{"_TABLE_x", "_partitiontime", "_FILE_name"}, expected: []string{"col_TABLE_x", "col_partitiontime", "col_FILE_name"}},
		{name: "too long", header: []string{long}, expected: []string{long[:300]}},
		{name: "too long duplicate", header: []string{long, long}, expected: []string{long[:300], long[:298] + "_2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, ColumnNames(tc.header))
		})
	}
}

func TestStringSchema(t *testing.T) {
	schema := stringSchema([]string{"a", "b"})

	assert.Len(t, schema, 2)
	for i, name := range []string{"a", "b"} {
		assert.Equal(t, name, schema[i].Name)
		assert.Equal(t, bq.StringFieldType, schema[i].Type)
		assert.False(t, schema[i].Required)
	}
}

func TestLoadJobID(t *testing.T) {
	base := loadJobID("bucket", "pending/a.csv", 100, 1)

	assert.True(t, strings.HasPrefix(base, loadJobIDPrefix))
	assert.True(t, strings.HasSuffix(base, "_1"))
	assert.Equal(t, base, loadJobID("bucket", "pending/a.csv", 100, 1), "deterministic")
	assert.NotEqual(t, base, loadJobID("bucket", "pending/a.csv", 101, 1), "new generation")
	assert.NotEqual(t, base, loadJobID("bucket", "pending/b.csv", 100, 1), "other object")
	assert.NotEqual(t, base, loadJobID("other", "pending/a.csv", 100, 1), "other bucket")
	assert.Equal(t, strings.TrimSuffix(base, "_1")+"_2", loadJobID("bucket", "pending/a.csv", 100, 2), "attempt suffix")

	for _, r := range base {
		assert.True(t, r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z', "invalid job id char %q", r)
	}
}

func TestToJobState(t *testing.T) {
	assert.Equal(t, JobStatePending, toJobState(bq.Pending))
	assert.Equal(t, JobStateRunning, toJobState(bq.Running))
	assert.Equal(t, JobStateDone, toJobState(bq.Done))
	assert.Equal(t, JobStateUnknown, toJobState(bq.StateUnspecified))
}

func TestToJobInfo(t *testing.T) {
	created := time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC)
	ref := JobRef{ID: "job", Location: "US"}
	stats := &bq.JobStatistics{
		CreationTime: created,
		StartTime:    created.Add(time.Second),
		EndTime:      created.Add(time.Minute),
		Details:      &bq.LoadStatistics{OutputRows: 42},
	}
	config := &bq.LoadConfig{
		Dst:    &bq.Table{ProjectID: "p", DatasetID: "ds", TableID: "Ventas"},
		Labels: map[string]string{"run_id": "r1"},
	}

	info := toJobInfo(ref, bq.Done, "", nil, stats, config)

	assert.Equal(t, JobInfo{
		Ref: ref, State: JobStateDone, DatasetID: "ds", TableID: "Ventas", OutputRows: 42,
		Labels: map[string]string{"run_id": "r1"}, Created: created, Started: created.Add(time.Second), Ended: created.Add(time.Minute),
	}, info)
	assert.True(t, info.Succeeded())
}

func TestToJobInfoPartialData(t *testing.T) {
	testCases := []struct {
		name      string
		state     bq.State
		errMsg    string
		stats     *bq.JobStatistics
		config    bq.JobConfig
		succeeded bool
	}{
		{name: "nothing known", state: bq.StateUnspecified},
		{name: "running", state: bq.Running, stats: &bq.JobStatistics{}},
		{name: "failed", state: bq.Done, errMsg: "bad row", config: &bq.LoadConfig{}},
		{name: "query job config ignored", state: bq.Done, config: &bq.QueryConfig{}, stats: &bq.JobStatistics{Details: &bq.QueryStatistics{}}, succeeded: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := toJobInfo(JobRef{ID: "j"}, tc.state, tc.errMsg, nil, tc.stats, tc.config)

			assert.Equal(t, tc.errMsg, info.Err)
			assert.Empty(t, info.TableID)
			assert.Zero(t, info.OutputRows)
			assert.Equal(t, tc.succeeded, info.Succeeded())
		})
	}
}

func TestIsHTTPStatus(t *testing.T) {
	conflict := &googleapi.Error{Code: http.StatusConflict}

	assert.True(t, isHTTPStatus(conflict, http.StatusConflict))
	assert.True(t, isHTTPStatus(fmt.Errorf("wrapped: %w", conflict), http.StatusConflict))
	assert.False(t, isHTTPStatus(conflict, http.StatusNotFound))
	assert.False(t, isHTTPStatus(errors.New("plain"), http.StatusConflict))
	assert.False(t, isHTTPStatus(nil, http.StatusConflict))
}

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", firstNonEmpty("", "a", "b"))
	assert.Equal(t, "", firstNonEmpty("", ""))
}

func TestNewClientValidation(t *testing.T) {
	c, err := NewClient("", `{"type":"service_account"}`)
	assert.Nil(t, c)
	assert.ErrorIs(t, err, ErrEmptyProjectID)

	c, err = NewClient("proj", "")
	assert.Nil(t, c)
	assert.ErrorIs(t, err, ErrEmptyCredentials)
}

func TestTableExistsValidation(t *testing.T) {
	c := &client{}

	_, err := c.TableExists(t.Context(), "", "t")
	assert.ErrorIs(t, err, ErrEmptyDatasetID)

	_, err = c.TableExists(t.Context(), "ds", "")
	assert.ErrorIs(t, err, ErrEmptyTableID)
}

func TestLoadCSVFromGCSValidation(t *testing.T) {
	c := &client{}

	_, err := c.LoadCSVFromGCS(t.Context(), LoadCSVRequest{SourceURI: "bad"})
	assert.ErrorIs(t, err, ErrInvalidSourceURI)
}

func TestErrorDetails(t *testing.T) {
	main := "Error while reading data, error message: CSV processing encountered too many errors"
	errs := []*bq.Error{
		{Message: main, Reason: "invalid"},
		nil,
		{Message: ""},
		{Location: "gs://b/a.csv", Message: "Error while reading data, error message: CSV table references column position 4, but line contains only 3 columns.; line_number: 7", Reason: "invalid"},
		{Message: "Could not parse 'x' as INT64"},
	}

	assert.Equal(t, []string{
		"gs://b/a.csv: Error while reading data, error message: CSV table references column position 4, but line contains only 3 columns.; line_number: 7",
		"Could not parse 'x' as INT64",
	}, errorDetails(errs, main))
	assert.Nil(t, errorDetails(nil, main))
}

func TestErrorDetailsIsCapped(t *testing.T) {
	errs := make([]*bq.Error, 50)
	for i := range errs {
		errs[i] = &bq.Error{Message: fmt.Sprintf("row %d", i)}
	}

	assert.Len(t, errorDetails(errs, ""), maxJobErrorDetails)
}

func TestToJobInfoKeepsErrorDetails(t *testing.T) {
	info := toJobInfo(JobRef{ID: "j"}, bq.Done, "main", []string{"row 3"}, nil, nil)

	assert.Equal(t, []string{"row 3"}, info.ErrorDetails)
}

func TestJobErrorMessage(t *testing.T) {
	assert.Equal(t, "too many errors", jobErrorMessage(&bq.Error{Message: "too many errors"}))
	assert.Equal(t, "gs://b/a.csv: bad row", jobErrorMessage(fmt.Errorf("w: %w", &bq.Error{Location: "gs://b/a.csv", Message: "bad row"})))
	assert.Equal(t, "plain", jobErrorMessage(errors.New("plain")))
}
