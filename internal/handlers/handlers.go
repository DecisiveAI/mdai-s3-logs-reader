package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	collectorlogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const s3LogsHandlerTimeout = 2 * time.Minute

// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/sdk-utilities-s3.html

type S3API interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func NewRouter(s3Client S3API, s3Bucket string) *http.ServeMux {
	r := http.NewServeMux()
	r.HandleFunc("GET /logs/{auditPath}/{timestamp}", func(w http.ResponseWriter, r *http.Request) {
		ListLogsHandler(r.Context(), w, r, s3Client, s3Bucket)
	})
	return r
}

func ListLogsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, s3Client S3API, s3Bucket string) {
	auditPath := r.PathValue("auditPath")

	if auditPath == "" {
		http.Error(w, "Invalid audit path: must be provided", http.StatusBadRequest)
		return
	}

	startParam := r.URL.Query().Get("start")
	endParam := r.URL.Query().Get("end")

	var prefixes []string

	if startParam != "" && endParam != "" {
		startTime, endTime, err := processTimeRange(startParam, endParam)
		if err != nil {
			writeJSONResponse(w, apiResponse{{"Error": upperFirst(err.Error())}})
			return
		}

		for t := startTime; !t.After(endTime); t = t.Add(time.Hour) {
			prefixes = append(prefixes, fmt.Sprintf("%s/%04d/%02d/%02d/%02d/", auditPath, t.Year(), t.Month(), t.Day(), t.Hour()))
		}
	}

	var returnedLogs []LogRecord
	for _, prefix := range prefixes {
		timeoutCtx, cancel := context.WithTimeout(ctx, s3LogsHandlerTimeout)
		logs, err := LoadLogsFromS3(timeoutCtx, s3Client, s3Bucket, prefix)
		cancel()
		if err != nil {
			log.Printf("Error loading logs for prefix %s: %v", prefix, err)
			continue
		}
		returnedLogs = append(returnedLogs, logs...)
	}

	if len(returnedLogs) == 0 {
		writeJSONResponse(w, apiResponse{{"Response": "No logs found for this range"}})
		return
	}

	// Leaving pagination logic here for now until we decide if we want to implement it
	// paginatedLogs := paginateLogs(returnedLogs, r)
	writeJSONResponse(w, returnedLogs)
}

func LoadLogsFromS3(ctx context.Context, client S3API, bucket string, prefix string) ([]LogRecord, error) {
	var returnedLogs []LogRecord

	if err := ctxCanceled(ctx); err != nil {
		return nil, err
	}

	listed, err := ListObjects(ctx, client, bucket, prefix)
	if err != nil {
		return nil, err
	}

	for _, obj := range listed {
		if err := ctxCanceled(ctx); err != nil {
			return nil, err
		}

		data, err := RetrieveObject(ctx, client, bucket, obj.Key)
		if err != nil {
			log.Printf("Error downloading %s: %v", obj.Key, err)
			continue
		}

		if err := ctxCanceled(ctx); err != nil {
			return nil, err
		}

		logs, err := ParseLogRecords(data)
		if err != nil {
			log.Printf("Error parsing logs from %s: %v", obj.Key, err)
			continue
		}

		returnedLogs = append(returnedLogs, logs...)
	}

	return returnedLogs, nil
}

func ListObjects(ctx context.Context, client S3API, bucket string, prefix string) ([]ListedObject, error) {
	var err error
	var output *s3.ListObjectsV2Output
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}
	var objects []types.Object
	objectPaginator := s3.NewListObjectsV2Paginator(client, input)
	for objectPaginator.HasMorePages() {
		output, err = objectPaginator.NextPage(ctx)
		if err != nil {
			var noBucket *types.NoSuchBucket
			if errors.As(err, &noBucket) {
				log.Printf("Bucket %s does not exist.\n", bucket)
				err = noBucket
			}
			break
		}
		objects = append(objects, output.Contents...)
	}

	var listed []ListedObject
	for _, obj := range objects {
		if obj.Key != nil && obj.LastModified != nil {
			listed = append(listed, ListedObject{
				Key:          *obj.Key,
				LastModified: *obj.LastModified,
			})
		}
	}

	return listed, err
}

func RetrieveObject(ctx context.Context, client S3API, bucket, key string) ([]byte, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v", err)
		}
	}()
	return io.ReadAll(resp.Body)
}

// Leaving pagination logic here for now until we decide if we want to implement it
// paginates the logs based on the limit and offset query parameters (ex. ?limit=100&offset=200)
// func paginateLogs(logs []LogRecord, r *http.Request) []LogRecord {
//	limit := 1000
//	offset := 0
//
//	query := r.URL.Query()
//
//	if l := query.Get("limit"); l != "" {
//		if val, err := strconv.Atoi(l); err == nil {
//			limit = val
//		}
//	}
//	if o := query.Get("offset"); o != "" {
//		if val, err := strconv.Atoi(o); err == nil {
//			offset = val
//		}
//	}
//
//	start := offset
//	end := offset + limit
//	if start > len(logs) {
//		start = len(logs)
//	}
//	if end > len(logs) {
//		end = len(logs)
//	}
//
//	return logs[start:end]
// }

func ParseLogRecords(data []byte) ([]LogRecord, error) {
	var req collectorlogspb.ExportLogsServiceRequest
	var records []LogRecord

	if err := protojson.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OTEL logs: %w", err)
	}

	for _, rlog := range req.GetResourceLogs() {
		for _, slog := range rlog.GetScopeLogs() {
			for _, lrec := range slog.GetLogRecords() {
				if lrec.GetTimeUnixNano() == 0 {
					continue
				}
				records = append(records, newLogRecord(lrec, rlog))
			}
		}
	}

	logMap := make(map[string]LogRecord)
	for _, parsed := range records {
		key := parsed.key()
		if existingLog, found := logMap[key]; found {
			existingLog.Count++
			logMap[key] = existingLog
		} else {
			parsed.Count = 1
			logMap[key] = parsed
		}
	}

	return slices.Collect(maps.Values(logMap)), nil
}

func normalizeSeverity(severity string) string {
	severity = strings.ToLower(severity)
	switch severity {
	case "info", "normal":
		return "INFO"
	case "warn", "warning":
		return "WARN"
	case "error":
		return "ERROR"
	case "fatal":
		return "FATAL"
	default:
		return strings.ToUpper(severity)
	}
}
