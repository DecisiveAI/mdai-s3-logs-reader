# mdai-s3-logs-reader Simulation

Test using a local MinIO server and MDAI cluster. This API retrieves and transforms OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). It returns the most recent JSON file for a given hourly timestamp.

---

## Prerequisites
- [MDAI Cluster](https://docs.mydecisive.ai/))

### Set up Minio Helm Chart
- Install Minio Helm via the [Minio walkthrough](https://docs.min.io/docs/deploy-minio-on-kubernetes-using-helm-guide.html)
  ```bash
  helm repo add minio https://charts.min.io/
  helm repo update
  ```
- Deploy MinIO server into local cluster:
  ```bash  
  helm install minio minio/minio -n minio --create-namespace -f minio-values.yaml
  ```
- Confirm MinIO is running:
  ```bash
  kubectl get pods -n minio
  ```
- Port forward MinIO:
  ```bash
  kubectl port-forward svc/minio -n minio 9001:9001
  ```
### Run it!
- `go run main.go`
- http://localhost:3000/logs/2025-04-22T22 (replace with date and time for your bucket)

### Uninstall MinIO
- Uninstall & Delete PVC:
  ```bash
  helm uninstall minio -n minio
  kubectl delete pvc --all -n minio
  ```
