# mdai-s3-logs-reader

A lightweight Go API for retrieving and transforming OpenTelemetry-formatted log files from S3-compatible storage. Returns most recent JSON file for a given hourly timestamp.

---

# THIS README IS A WIP! Under construction and will be updated as the project progresses.

## Prerequisites

- Go 1.20+
- Docker
- [Kind](https://kind.sigs.k8s.io/)
- Existing Cluster (Recommended: [mdai](https://docs.mydecisive.ai/))
- mdai-collector - TBD adding explanation for this
- S3-compatible storage set up for mdai-collector **or** [MinIO](https://min.io/)
    - Deploy MinIO server into local cluster using the [Minio walkthrough](/simulation/README)
    - [S3 setup](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)
- Log files must be valid JSON

## Getting Started
- Clone the repository
  ```bash 
  go mod vendor
  ```

### Set up API for pulling logs from S3-compatible storage
- Create Docker image
  ```bash
  docker build -t mdai-s3-logs-reader:0.0.1 .
  ```
- Load Docker image into kind cluster
  ```bash
  kind load docker-image mdai-s3-logs-reader:0.0.1 --name mdai
  ```
- Create a secret.yaml using [template-secret.yaml](/template_secret.yaml)
- Apply the secret.yaml to the cluster
  ```bash
  kubectl apply -f secret.yaml
  ```
- Apply service account, service, and deployment
  ```bash
  kubectl apply -f deployment/serviceAccount.yaml -f deployment/service.yaml -f deployment/deployment.yaml
  ```
- Check deployment status
  ```bash
  kubectl get pods -n mdai
  ```
- Port forward the service
  ```bash
  kubectl port-forward svc/mdai-s3-logs-reader-service 4400:4400 -n mdai
    ```

### Test it!
- http://localhost:4400/logs/<bucket>/YYYY-MM-DDTHH or with range of up to 4 hours http://localhost:4400/logs/<bucket>/YYYY-MM-DDTHH?end=YYYY-MM-DDTHH&start=YYYY-MM-DDTHH (replace with date and time for your bucket) Note: UTC time is used
  - Example: http://localhost:4400/logs/mdaihub-sample-hub/2025-04-01T20
  - Example w/ Range: http://localhost:4400/logs/mdaihub-sample-hub/2025-05-06T17?end=2025-05-06T20&start=2025-05-06T17
- You can also port-forward Grafana and import the [example dashboard](/sample-data/grafana/mdai-audit-streams.json)
  ```bash
  kubectl port-forward svc/mdai-grafana 3000:80 -n mdai
  ```
