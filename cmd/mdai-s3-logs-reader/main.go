package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/decisiveai/mdai-s3-logs-reader/internal/handlers"
)

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 10 * time.Second
	defaultWriteTimeout      = 10 * time.Second
	defaultIdleTimeout       = 120 * time.Second
	defaultHTTPPort          = "4400"
)

func main() {
	s3Bucket := os.Getenv("S3_BUCKET")
	s3Region := os.Getenv("AWS_REGION")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv(s3Region)),
		config.WithClientLogMode(aws.LogRetries),
	)
	if err != nil {
		log.Fatal("unable to load SDK config, ", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	r := handlers.NewRouter(s3Client, s3Bucket)

	srv := &http.Server{
		Addr:              ":" + defaultHTTPPort, // Grafana uses port 3000, so making port 4400
		Handler:           r,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	log.Println("Listening on :4400")
	log.Fatal(srv.ListenAndServe())
}
