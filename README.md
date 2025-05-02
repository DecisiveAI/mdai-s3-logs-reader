# mdai-s3-logs-reader

A lightweight Go API for retrieving and transforming OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). Returns most recent JSON file for a given hourly timestamp.

---

# THIS README IS A WIP! Under construction and will be updated as the project progresses.

## Prerequisites

- Go 1.20+
- Existing Cluster (Recommended: [mdai](https://docs.mydecisive.ai/))
- [mdai-collector](https://github.com/DecisiveAI/watcher-collector) deployed on your cluster
    - Clone the repository
    - ```bash
      make docker-build-mdai-collector
      kind load docker-image mdai-collector:0.1.4 --name mdai
      kubectl apply -f ./deployment/mdai-collector
      ```
*(Note: steps above are temporary and will change once the mdai-collector is active in helm. There may also be changes that have to be made for the deployment to work in the collector)*
- S3-compatible storage set up for mdai-collector **or** [MinIO](https://min.io/)
    - Deploy MinIO server into local cluster using the [Minio walkthrough](/simulation/README)
    - [S3 setup](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)
- Log files must be valid JSON

## Getting Started
- Clone the repository
- ```ssh 
  go mod vendor
  ```
- [Set up AWS credentials via CLI using SSO](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html#cli-configure-sso-session) for S3 set up for mdai-collector OR use existing credentials
  - Be sure output format is `json`
- Login to AWS SSO
  - ```ssh 
    aws sso login --profile <your-profile-name>
    ```
- Replace 'awsSsoProfile' in `main.go` with your profile name

### Run it!
- `go run main.go`
- http://localhost:3000/log/YYYY-MM-DDTHH (replace with date and time for your bucket) Note: UTC time is used
  - Example: http://localhost:3000/log/2025-04-01T20
  - Pagination is supported, by default limit is 100 logs (ex. http://localhost:3000/log/2025-04-01T20?limit=100&offset=200)
