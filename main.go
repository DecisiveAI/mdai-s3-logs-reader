package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"

	"github.com/decisiveai/mdai-s3-logs-reader/internal"
)

var (
	s3Bucket = "mdai-collector-logs"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithClientLogMode(aws.LogRetries),
	)
	if err != nil {
		log.Fatal("unable to load SDK config, ", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	r := chi.NewRouter()
	r.Get("/logs/{timestamp}", func(w http.ResponseWriter, r *http.Request) {
		internal.ListObjectsHandler(w, r, s3Client, s3Bucket)
	})

	log.Println("Listening on :4400")
	log.Fatal(http.ListenAndServe(":4400", r)) //Grafana uses port 3000, so making port 4400
}
