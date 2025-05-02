# mdai-s3-logs-reader

A lightweight Go API for retrieving and transforming OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). Returns most recent JSON file for a given hourly timestamp.

---

# THIS README IS A WIP! Under construction and will be updated as the project progresses.

## Prerequisites

- Go 1.20+
- Existing Cluster (Recommended: [mdai](https://docs.mydecisive.ai/))
- mdai-collector - TBD adding explanation for this
- S3-compatible storage set up for mdai-collector **or** [MinIO](https://min.io/)
    - Deploy MinIO server into local cluster using the [Minio walkthrough](/simulation/README)
    - [S3 setup](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)
- Log files must be valid JSON

## Getting Started
- Clone the repository
- ```ssh 
  go mod vendor
  ```

### TBD on AWS config step; moved to kube secret for this (was initially SSO via AWS CLI)

### Run it!
- `go run main.go`
- http://localhost:3000/log/YYYY-MM-DDTHH (replace with date and time for your bucket) Note: UTC time is used
  - Example: http://localhost:3000/log/2025-04-01T20
  - Pagination is supported, by default limit is 100 logs (ex. http://localhost:3000/log/2025-04-01T20?limit=100&offset=200)
