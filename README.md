# mdai-s3-logs-reader

[![codecov](https://codecov.io/gh/mydecisive/mdai-s3-logs-reader/graph/badge.svg?token=U0LFMJSVNR)](https://codecov.io/gh/mydecisive/mdai-s3-logs-reader)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/mdai-s3-logs-reader)](https://artifacthub.io/packages/search?repo=mdai-s3-logs-reader)

A lightweight Go API for reading OpenTelemetry log export files from S3 and returning normalized log records for a given hourly timestamp.

The service is intended for MDAI environments where collector, hub, or audit logs are written to S3 using hourly prefixes.

## How it works

`mdai-s3-logs-reader` exposes one HTTP endpoint:

```text
GET /logs/{logPath}/files?start={unixMs}&end={unixMs}
```

For each hour in the requested range, the service lists objects under:

```text
{logPath}/YYYY/MM/DD/HH/
```

It downloads matching objects from the configured S3 bucket, parses OpenTelemetry `ExportLogsServiceRequest` JSON, deduplicates equivalent records, and returns the transformed records as JSON.

The current maximum query range is 4 hours.

## Prerequisites

- Go 1.25+
- Docker, if building an image locally
- Helm and `kubectl`, if deploying to Kubernetes
- An S3 bucket containing valid OpenTelemetry log export JSON
- AWS credentials with `s3:ListBucket` and `s3:GetObject` access for the configured bucket

## Configuration

The service reads configuration from environment variables:

| Variable | Required | Description |
| --- | --- | --- |
| `AWS_REGION` | Yes | AWS region for the S3 bucket. |
| `S3_BUCKET` | Yes | S3 bucket that stores the log files. |
| `AWS_ACCESS_KEY_ID` | Yes, unless using another AWS credential provider | AWS access key. |
| `AWS_SECRET_ACCESS_KEY` | Yes, unless using another AWS credential provider | AWS secret key. |
| `AWS_SESSION_TOKEN` | Optional | Session token for temporary credentials. |

## Local development

Run tests:

```bash
make test
```

Build the binary:

```bash
make build
```

Run the service locally with AWS credentials available in the environment:

```bash
AWS_REGION=us-east-1 \
S3_BUCKET=mdai-collector-logs \
go run ./cmd/mdai-s3-logs-reader
```

The service listens on port `4400`.

## Docker

Build a local image:

```bash
docker build -t mdai-s3-logs-reader:local .
```

Run the image:

```bash
docker run --rm -p 4400:4400 \
  -e AWS_REGION=us-east-1 \
  -e S3_BUCKET=mdai-collector-logs \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e AWS_SESSION_TOKEN \
  mdai-s3-logs-reader:local
```

## Kubernetes

Create a secret containing AWS credentials:

```bash
kubectl create secret generic aws-credentials \
  --namespace mdai \
  --from-literal=AWS_ACCESS_KEY_ID='<access-key>' \
  --from-literal=AWS_SECRET_ACCESS_KEY='<secret-key>'
```

Install the Helm chart:

```bash
helm upgrade --install mdai-s3-logs-reader ./deployment \
  --namespace mdai \
  --create-namespace \
  --set awsRegion=us-east-1 \
  --set s3Bucket=mdai-collector-logs \
  --set awsAccessKeySecret=aws-credentials
```

Check the deployment:

```bash
kubectl get pods -n mdai -l app=mdai-s3-logs-reader
```

Port-forward the service:

```bash
kubectl port-forward svc/mdai-s3-logs-reader-service 4400:4400 -n mdai
```

## Usage

Request logs by log path and Unix millisecond range:

```bash
curl 'http://localhost:4400/logs/mdaihub-sample-hub/files?start=1746735223658&end=1746746023659'
```

Example Grafana dashboard JSON is available at [sample-data/grafana/mdai-audit-streams-v2.json](sample-data/grafana/mdai-audit-streams-v2.json).
