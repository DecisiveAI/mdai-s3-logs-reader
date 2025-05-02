package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/decisiveai/mdai-s3-logs-reader/internal"
)

// TODO: Make tests work, these are not working currently and are WIP

type mockS3Client struct {
	FakeListOutput *s3.ListObjectsV2Output
	FakeObjectMap  map[string][]byte
}

func TestMain(m *testing.M) {
	if os.Getenv("SHOW_LOGS") == "1" {
		log.SetOutput(os.Stderr)
	} else {
		log.SetOutput(io.Discard)
	}
	m.Run()
}

func (m *mockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.FakeListOutput, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	data := m.FakeObjectMap[*input.Key]
	return &s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader(data)),
	}, nil
}

func TestListAndParseLogs(t *testing.T) {
	fileData, err := os.ReadFile("sample-data/otel-runtime-sample.json")
	if err != nil {
		t.Fatalf("failed to read sample file: %v", err)
	}

	mock := &mockS3Client{
		FakeListOutput: &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("logs/2025/04/28/20/ooda-event-handler-sample.json")},
			},
		},
		FakeObjectMap: map[string][]byte{
			"logs/2025/04/28/20/ooda-event-handler-sample.json": fileData,
		},
	}

	logs, err := internal.LoadLogsFromS3(context.TODO(), mock, "bucket-name", "logs/2025/04/28/20")
	if err != nil {
		t.Fatal(err)
	}

	if len(logs) == 0 {
		t.Error("expected logs, got none")
	}
}
