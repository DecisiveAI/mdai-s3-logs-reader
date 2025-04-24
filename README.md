# mdai-s3-logs-reader

A lightweight Go API for retrieving and transforming OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). Returns most recent JSON file for a given hourly timestamp.

---

## Prerequisites

- Go 1.20+
- [mdai-collector](https://github.com/DecisiveAI/watcher-collector/tree/rlaw/debug-collector) deployed on your cluster
    - Clone the repository
    - `make docker-build-mdai-collector`
    - `kind load docker-image mdai-collector:0.1.4 --name mdai`
    - `kubectl apply -f ./deployment/mdai-collector`
      
*(Note: steps above are temporary and will change once the mdai-collector is active in helm. There may also be changes that have to be made for the deployment to work in the collector)*
- MinIO  with log files stored in this format: log/YYYY/MM/DD/HH/hub-logs_{UID}.json

[//]: # (TODO: or  S3 bucket)

- Log files must be valid JSON in OpenTelemetry structure

## Getting Started
- Clone the repository
- `go mod vendor`

### Configure AWS credentials for MinIO 

[//]: # (TODO: S3 set up)

~/.aws/credentials
```ini
[local-minio]
aws_access_key_id = admin
aws_secret_access_key = admin123
```
~/.aws/config
```ini
[profile local-minio]
region = us-east-1
```
### Run it!
- `go run main.go`
- http://localhost:3000/logs/2025-04-22T22 (replace with date and time for your bucket)
