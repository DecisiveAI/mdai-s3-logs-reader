package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-chi/chi/v5"

	internalTypes "github.com/decisiveai/mdai-s3-logs-reader/types"
)

var (
	bucket = "mdai-collector-logs"
)

func ListObjectsHandler(w http.ResponseWriter, r *http.Request, s3Client *s3.Client) {
	timestamp := chi.URLParam(r, "timestamp")

	parsedTime, err := time.Parse("2006-01-02T15", timestamp)
	if err != nil {
		http.Error(w, "Invalid timestamp format. Use YYYY-MM-DDTHH", http.StatusBadRequest)
		return
	}
	prefix := fmt.Sprintf("log/%04d/%02d/%02d/%02d/", parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), parsedTime.Hour())

	listed, err := listObjects(r.Context(), s3Client, bucket, prefix)
	if err != nil {
		http.Error(w, "Error listing objects", http.StatusInternalServerError)
		return
	}

	mostRecentLogs, err := getObject(r.Context(), s3Client, bucket, listed)
	if err != nil {
		http.Error(w, "Error listing objects", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(mostRecentLogs)
	if err != nil {
		return
	}
}

func listObjects(ctx context.Context, client *s3.Client, bucket string, prefix string) ([]internalTypes.ListedObject, error) {
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

func getObject(ctx context.Context, client *s3.Client, bucketName string, listed []internalTypes.ListedObject) ([]byte, error) {
	if len(listed) == 0 {
		return nil, fmt.Errorf("no logs found for the given timestamp")
	}
	objKey := listed[0].Key

	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objKey),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		if errors.As(err, &noKey) {
			log.Printf("Can't get object %s from bucket %s. No such key exists.\n", objKey, bucketName)
			err = noKey
		} else {
			log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objKey, err)
		}
	}

	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objKey, err)
	}

	return body, err
}
