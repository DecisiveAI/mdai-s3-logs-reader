package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/decisiveai/mdai-s3-logs-reader/internal"
)

// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/unit-testing.html

const (
	bucket    = "mdai-collector-logs"
	prefix    = "/hub-monitor-hub-logs/2025/01/17/02/"
	auditPath = "hub-monitor-hub-logs"
	key       = "some-logfile.json"
	logFile1  = "sample-data/hub-logs-sample.json"
	// logFile2, the first log timestamp is empty to test parsing filter for timestamp
	logFile2 = "sample-data/collector-logs-sample.json"
	logFile3 = "sample-data/audit-logs-sample.json"
)

type mockS3Client struct {
	GetObjectFunc     func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2Func func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

func (m *mockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.GetObjectFunc(ctx, params, optFns...)
}
func (m *mockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.ListObjectsV2Func(ctx, params, optFns...)
}

func TestListLogsHandler(t *testing.T) {
	cases := []struct {
		name           string
		filePath       string
		requestURL     string
		expectPrefixes map[string]bool
	}{
		{
			name:       "test1",
			filePath:   logFile1,
			requestURL: "/logs/hub-monitor-hub-logs/files?start=1737080400000&end=1737084000000",
			expectPrefixes: map[string]bool{
				"hub-monitor-hub-logs/2025/01/17/02/": true,
				"hub-monitor-hub-logs/2025/01/17/03/": true,
			},
		},
		{
			name:       "test2",
			filePath:   logFile2,
			requestURL: "/logs/hub-monitor-hub-logs/files?start=1737084000000&end=1737087600000",
			expectPrefixes: map[string]bool{
				"hub-monitor-hub-logs/2025/01/17/03/": true,
				"hub-monitor-hub-logs/2025/01/17/04/": true,
			},
		},
		{
			name:       "test3",
			filePath:   logFile3,
			requestURL: "/logs/hub-monitor-hub-logs/files?start=1737087600000&end=1737091200000",
			expectPrefixes: map[string]bool{
				"hub-monitor-hub-logs/2025/01/17/04/": true,
				"hub-monitor-hub-logs/2025/01/17/05/": true,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			testFile, err := os.ReadFile(tt.filePath)
			require.NoError(t, err)

			expectTime := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)

			mockClient := &mockS3Client{
				ListObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
					require.NotNil(t, params.Prefix, "Prefix should not be nil")
					actual := *params.Prefix
					t.Logf("ListObjectsV2Func called with prefix: %q", actual)
					require.Contains(t, tt.expectPrefixes, actual,
						"expected prefix to be one of %v, got %q", keys(tt.expectPrefixes), actual)
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{
								Key:          aws.String(tt.filePath),
								LastModified: aws.Time(expectTime),
							},
						},
					}, nil
				},
				GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(testFile)),
					}, nil
				},
			}

			req := httptest.NewRequest("GET", tt.requestURL, nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, &chi.Context{
				URLParams: chi.RouteParams{
					Keys:   []string{"auditPath"},
					Values: []string{auditPath},
				},
			}))

			w := httptest.NewRecorder()
			internal.ListLogsHandler(w, req, mockClient, bucket)
			resp := w.Result()
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Logf("error closing response body: %v", err)
				}
			}()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200, got %d", resp.StatusCode)
			var logs []struct {
				Timestamp string `json:"timestamp"`
				Body      string `json:"body"`
			}
			err = json.Unmarshal(body, &logs)
			require.NoError(t, err, "Failed to parse response JSON")
			assert.NotZero(t, len(logs), "Expected at least one log record in HTTP response")

			for i, rec := range logs {
				assert.NotEmpty(t, rec.Timestamp, "Missing timestamp in record %d: %+v", i, rec)
				assert.NotEmpty(t, rec.Body, "Missing body in record %d: %+v", i, rec)
			}
			t.Logf("[%s] HTTP status: %d", tt.name, resp.StatusCode)
			t.Logf("[%s] First log record: %+v", tt.name, logs[0])
		})
	}
}

func TestLoadLogsFromS3(t *testing.T) {
	testFile1, _ := os.ReadFile(logFile1)
	testFile2, _ := os.ReadFile(logFile2)
	testFile3, _ := os.ReadFile(logFile3)
	expectTime := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)

	cases := []struct {
		name     string
		bucket   string
		testFile []byte
		key      string
		prefix   string
	}{
		{"case1", bucket, testFile1, logFile1, prefix},
		{"case2", bucket, testFile2, logFile2, prefix},
		{"case3", bucket, testFile3, logFile3, prefix},
		{"case4", bucket, testFile1, logFile1, prefix},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				ListObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
					if params.Bucket == nil || *params.Bucket != bucket {
						t.Fatalf("expect bucket %s, got %v", bucket, params.Bucket)
					}
					if params.Prefix == nil || *params.Prefix != tt.prefix {
						t.Fatalf("expect prefix %s, got %v", tt.prefix, params.Prefix)
					}
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{
								Key:          aws.String(tt.key),
								LastModified: aws.Time(expectTime),
							},
						},
						Prefix: aws.String(tt.prefix),
					}, nil
				},
				GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					if params.Bucket == nil || *params.Bucket != bucket {
						t.Fatalf("expect bucket %s, got %v", bucket, params.Bucket)
					}
					if params.Key == nil || *params.Key != tt.key {
						t.Fatalf("expect key %s, got %v", tt.key, params.Key)
					}
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(tt.testFile)),
					}, nil
				},
			}

			ctx := context.TODO()
			logs, err := internal.LoadLogsFromS3(ctx, mockClient, bucket, tt.prefix)
			assert.NoError(t, err)
			assert.NotZero(t, len(logs), "Expected at least one log record")
			t.Logf("Success for %s: parsed %d records. First: %+v", tt.name, len(logs), logs[0])
		})
	}
}

func TestGetObjectFromS3(t *testing.T) {
	testFile1, _ := os.ReadFile(logFile1)
	cases := []struct {
		name     string
		bucket   string
		key      string
		prefix   string
		testFile []byte
	}{
		{"case1", bucket, key, prefix, testFile1},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				GetObjectFunc: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					t.Helper()
					if params.Bucket == nil {
						t.Fatal("expect bucket to not be nil")
					}
					if e, a := bucket, *params.Bucket; e != a {
						t.Errorf("expect %v, got %v", e, a)
					}
					if params.Key == nil {
						t.Fatal("expect key to not be nil")
					}
					if e, a := key, *params.Key; e != a {
						t.Errorf("expect %v, got %v", e, a)
					}
					return &s3.GetObjectOutput{
						Body: io.NopCloser(bytes.NewReader(testFile1)),
					}, nil
				},
			}
			ctx := context.TODO()
			_, err := internal.RetrieveObject(ctx, mockClient, tt.bucket, tt.key)
			assert.NoError(t, err)
			t.Logf("Success! Test %s", tt.name)
		})
	}
}

func TestListObjectsFromS3(t *testing.T) {
	expectTime := time.Date(2023, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name   string
		bucket string
		prefix string
	}{
		{"case1", bucket, prefix},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockS3Client{
				ListObjectsV2Func: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
					if params.Bucket == nil {
						t.Fatal("expect bucket to not be nil")
					}
					if e, a := bucket, *params.Bucket; e != a {
						t.Errorf("expect %v, got %v", e, a)
					}
					if e, a := prefix, *params.Prefix; e != a {
						t.Errorf("expect %v, got %v", e, a)
					}
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{
								Key:          aws.String(logFile1),
								LastModified: aws.Time(expectTime),
							},
							{
								Key:          aws.String(logFile2),
								LastModified: aws.Time(expectTime),
							},
						},
						Prefix: aws.String(prefix),
					}, nil
				},
			}
			ctx := context.TODO()
			content, err := internal.ListObjects(ctx, mockClient, tt.bucket, tt.prefix)
			assert.NoError(t, err, "Expected no error from ListObjects")
			t.Logf("Success! Test %s output: %+v", tt.name, content)
		})
	}
}

func TestParseLogRecords(t *testing.T) {
	testFile1, err := os.ReadFile(logFile1)
	require.NoError(t, err)
	testFile2, err := os.ReadFile(logFile2)
	require.NoError(t, err)
	testFile3, err := os.ReadFile(logFile3)
	require.NoError(t, err)

	cases := []struct {
		name string
		body []byte
	}{
		{"parseLogs1", testFile1},
		{"parseLogs2", testFile2},
		{"parseLogs3", testFile3},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			content, err := internal.ParseLogRecords(tt.body)
			assert.NoError(t, err)
			assert.NotZero(t, len(content), "Expected at least one log record")
			rec := content[0]
			if rec.Timestamp == "" || rec.Body == "" {
				t.Errorf("missing expected fields in record: %+v", rec)
			} else {
				t.Logf("First record: %+v", rec)
			}
		})
	}
}

func keys(m map[string]bool) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
