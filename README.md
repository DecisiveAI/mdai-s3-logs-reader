# mdai-s3-logs-reader

A lightweight Go API for retrieving and transforming OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). Returns most recent JSON file for a given hourly timestamp.

---

## Prerequisites

- Go 1.20+
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- Existing Cluster (Recommended: [mdai](https://docs.mydecisive.ai/))
- [mdai-collector](https://github.com/DecisiveAI/watcher-collector/tree/rlaw/debug-collector) deployed on your cluster
    - Clone the repository
    - `make docker-build-mdai-collector`
    - `kind load docker-image mdai-collector:0.1.4 --name mdai`
    - `kubectl apply -f ./deployment/mdai-collector`
*(Note: steps above are temporary and will change once the mdai-collector is active in helm. There may also be changes that have to be made for the deployment to work in the collector)*
- [MinIO](https://min.io/) **or** S3-compatible storage
    - Deploy MinIO server into local cluster using the [Minio walkthrough](/simulation/simulation.md)
    - [S3 setup](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)
- Log files must be valid JSON

## Getting Started
- Clone the repository
- `go mod vendor`

### Run it!
- `go run main.go`
- http://localhost:3000/logs/2025-04-22T22 (replace with date and time for your bucket)
