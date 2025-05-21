package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

func ListLogsHandler(w http.ResponseWriter, r *http.Request, s3Client S3API, s3Bucket string) {
	auditPath := chi.URLParam(r, "auditPath")

	if auditPath == "" {
		http.Error(w, "Invalid audit path: must be provided", http.StatusBadRequest)
		return
	}

	startParam := r.URL.Query().Get("start")
	endParam := r.URL.Query().Get("end")

	var prefixes []string

	if startParam != "" && endParam != "" {
		var (
			startTime time.Time
			endTime   time.Time
		)

		if startInt, err := strconv.ParseInt(startParam, 10, 64); err == nil {
			startTime = time.UnixMilli(startInt).UTC()
		}

		if endInt, err := strconv.ParseInt(endParam, 10, 64); err == nil {
			endTime = time.UnixMilli(endInt).UTC()
		}

		if endTime.Before(startTime) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"Error": "End time must be after start time"},
			})
			return
		}
		if endTime.Sub(startTime) > 4*time.Hour {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]string{
				{"Error": "Time range must be 4 hours or less"},
			})
			return
		}

		if endTime.Sub(startTime) < time.Hour {
			startTime = startTime.Truncate(time.Hour)
			endTime = startTime.Add(time.Hour - time.Nanosecond)
		}

		for t := startTime; !t.After(endTime); t = t.Add(time.Hour) {
			prefixes = append(prefixes, fmt.Sprintf("%s/%04d/%02d/%02d/%02d/", auditPath, t.Year(), t.Month(), t.Day(), t.Hour()))
		}
	}

	var returnedLogs []internalTypes.LogRecord
	for _, prefix := range prefixes {
		ctx := context.Background()
		logs, err := LoadLogsFromS3(ctx, s3Client, s3Bucket, prefix)
		if err != nil {
			log.Printf("Error loading logs for prefix %s: %v", prefix, err)
			continue
		}
		returnedLogs = append(returnedLogs, logs...)
	}

	if len(returnedLogs) > 0 {
		// Leaving pagination logic here for now until we decide if we want to implement it
		//paginatedLogs := paginateLogs(returnedLogs, r)
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(returnedLogs)
		if err != nil {
			log.Printf("failed to encode JSON: %v", err)
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]map[string]string{
			{"Response": "No logs found for this range"},
		})
	}
}

func LoadLogsFromS3(ctx context.Context, client S3API, bucket string, prefix string) ([]internalTypes.LogRecord, error) {
	var returnedLogs []internalTypes.LogRecord

	listed, err := ListObjects(ctx, client, bucket, prefix)
	if err != nil {
		return nil, err
	}

	for _, obj := range listed {
		data, err := RetrieveObject(ctx, client, bucket, obj.Key)
		if err != nil {
			log.Printf("Error downloading %s: %v", obj.Key, err)
			continue
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

func ListObjects(ctx context.Context, client S3API, bucket string, prefix string) ([]internalTypes.ListedObject, error) {
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
//func paginateLogs(logs []internalTypes.LogRecord, r *http.Request) []internalTypes.LogRecord {
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
//}

func ParseLogRecords(data []byte) ([]internalTypes.LogRecord, error) {
	var rawLog map[string]any
	if err := json.Unmarshal(data, &rawLog); err != nil {
		return nil, err
	}

	var (
		resourceLogs = rawLog["resourceLogs"].([]any)
		records      []internalTypes.LogRecord
	)

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
					Timestamp:      parseTimestampNano(safeString(recMap["timeUnixNano"])),
					Severity:       normalizeSeverity(safeString(recMap["severityText"])),
					SeverityNumber: safeString(recMap["severityNumber"]),
					Body:           safeString(recMap["body"].(map[string]any)["stringValue"]),
					Reason:         attrs["k8s.event.reason"],
					EventName:      attrs["k8s.event.name"],
					Pod:            objectName,
					ServiceName:    resourceAttrs["service.name"],
				})
			}
		}
	}

	logMap := make(map[string]internalTypes.LogRecord)
	for _, parsed := range records {
		key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
			parsed.Timestamp,
			parsed.Severity,
			parsed.Reason,
			parsed.EventName,
			parsed.Pod,
			parsed.ServiceName,
			parsed.Body,
		)
		existingLog, found := logMap[key]
		if found {
			existingLog.Count++
			logMap[key] = existingLog
		} else {
			parsed.Count = 1
			logMap[key] = parsed
		}
	}

	countedLogs := make([]internalTypes.LogRecord, 0, len(logMap))
	for _, filtered := range logMap {
		countedLogs = append(countedLogs, filtered)
	}

	return countedLogs, nil
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

func parseTimestampNano(ts string) string {
	if nano, err := strconv.ParseInt(ts, 10, 64); err == nil {
		return time.Unix(0, nano).UTC().Format(time.RFC3339)
	}
	return ts
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
