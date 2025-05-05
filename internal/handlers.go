package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-chi/chi/v5"

	internalTypes "github.com/decisiveai/mdai-s3-logs-reader/types"
)

// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/sdk-utilities-s3.html

type S3API interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func ListLogsHandler(w http.ResponseWriter, r *http.Request, s3Client *s3.Client, s3Bucket string) {
	auditPath := chi.URLParam(r, "auditPath")
	timestamp := chi.URLParam(r, "timestamp")

	if auditPath == "" {
		http.Error(w, "Invalid audit path: must be provided", http.StatusBadRequest)
		return
	}

	parsedTime, err := time.Parse("2006-01-02T15", timestamp)
	if err != nil {
		http.Error(w, "Invalid timestamp format. Use YYYY-MM-DDTHH", http.StatusBadRequest)
		return
	}
	prefix := fmt.Sprintf("%s/%04d/%02d/%02d/%02d/", auditPath, parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), parsedTime.Hour())

	returnedLogs, err := LoadLogsFromS3(r.Context(), s3Client, s3Bucket, prefix)
	if err != nil {
		return
	}

	paginatedLogs := paginateLogs(returnedLogs, r)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(paginatedLogs)
	if err != nil {
		log.Printf("failed to encode JSON: %v", err)
	}
}

func LoadLogsFromS3(ctx context.Context, client S3API, bucket string, prefix string) ([]internalTypes.LogRecord, error) {
	var returnedLogs []internalTypes.LogRecord

	listed, err := listObjects(ctx, client, bucket, prefix)
	if err != nil {
		return nil, err
	}

	for _, obj := range listed {
		data, err := retrieveObject(ctx, client, bucket, obj.Key)
		if err != nil {
			log.Printf("Error downloading %s: %v", obj.Key, err)
			continue
		}

		logs, err := parseLogRecords(data)
		if err != nil {
			log.Printf("Error parsing logs from %s: %v", obj.Key, err)
			continue
		}

		returnedLogs = append(returnedLogs, logs...)
	}

	return returnedLogs, nil
}

func listObjects(ctx context.Context, client S3API, bucket string, prefix string) ([]internalTypes.ListedObject, error) {
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
		} else {
			objects = append(objects, output.Contents...)
		}
	}

	var listed []internalTypes.ListedObject
	for _, obj := range objects {
		if obj.Key != nil && obj.LastModified != nil {
			listed = append(listed, internalTypes.ListedObject{
				Key:          *obj.Key,
				LastModified: *obj.LastModified,
			})
		}
	}

	return listed, err
}

func retrieveObject(ctx context.Context, client S3API, bucket string, key string) ([]byte, error) {
	buf := manager.NewWriteAtBuffer([]byte{})
	downloader := manager.NewDownloader(client)

	_, err := downloader.Download(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// paginates the logs based on the limit and offset query parameters (ex. ?limit=100&offset=200)
func paginateLogs(logs []internalTypes.LogRecord, r *http.Request) []internalTypes.LogRecord {
	limit := 100
	offset := 0

	query := r.URL.Query()

	if l := query.Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	if o := query.Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	start := offset
	end := offset + limit
	if start > len(logs) {
		start = len(logs)
	}
	if end > len(logs) {
		end = len(logs)
	}

	return logs[start:end]
}

// TODO: Find a better way to handle this
func parseLogRecords(data []byte) ([]internalTypes.LogRecord, error) {
	var raw map[string]any
	err := json.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	resourceLogs, ok := raw["resourceLogs"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid format: missing resourceLogs")
	}

	var records []internalTypes.LogRecord

	for _, res := range resourceLogs {
		resMap := res.(map[string]any)
		resourceAttrs := extractAttributes(resMap["resource"])

		objectName := resourceAttrs["k8s.object.name"]

		scopeLogs := resMap["scopeLogs"].([]any)
		for _, scope := range scopeLogs {
			scopeMap := scope.(map[string]any)
			logRecords := scopeMap["logRecords"].([]any)

			for _, rec := range logRecords {
				recMap := rec.(map[string]any)
				attrs := extractAttributes(recMap)

				records = append(records, internalTypes.LogRecord{
					Timestamp:         safeString(recMap["timeUnixNano"]),
					ObservedTimestamp: safeString(recMap["observedTimeUnixNano"]),
					Severity:          safeString(recMap["severityText"]),
					SeverityNumber:    safeString(recMap["severityNumber"]),
					Body:              safeString(recMap["body"].(map[string]any)["stringValue"]),
					Reason:            attrs["k8s.event.reason"],
					EventName:         attrs["k8s.event.name"],
					Pod:               objectName,
					ServiceName:       resourceAttrs["service.name"],
				})
			}
		}
	}

	return records, nil
}

func extractAttributes(obj any) map[string]string {
	result := make(map[string]string)
	if obj == nil {
		return result
	}
	objMap := obj.(map[string]any)
	attrs, ok := objMap["attributes"].([]any)
	if !ok {
		return result
	}
	for _, attr := range attrs {
		attrMap := attr.(map[string]any)
		key := safeString(attrMap["key"])
		valMap := attrMap["value"].(map[string]any)
		val := safeString(valMap["stringValue"])
		if key != "" {
			result[key] = val
		}
	}
	return result
}

func safeString(val any) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
