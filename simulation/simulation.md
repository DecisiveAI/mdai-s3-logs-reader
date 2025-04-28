# mdai-s3-logs-reader Simulation

Test using a local MinIO server and MDAI cluster. This API retrieves and transforms OpenTelemetry-formatted log files from S3-compatible storage (e.g. MinIO). It returns the most recent JSON file for a given hourly timestamp.

---

## Prerequisites

- Go 1.20+
- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)
- [MDAI Cluster](https://docs.mydecisive.ai/))

### Set up AWS Simulation with local MinIO server:
- AWS CLI for MinIO
    - Run: 
    - ``` 
      aws configure --profile local-minio 
      ```
        ```
      // AWS CLI: //
          AWS Access Key ID [None]: admin_
          AWS Secret Access Key [None]: admin123
          Default region name [None]: us-east-1
          Default output format [None]:
        ```
    - Edit ~/.aws/config for endpoint:
      - ```
        echo -e "\n[profile local-minio]\ns3 =\n    endpoint_url = http://localhost:9000\n    addressing_style = path" >> ~/.aws/config
        ```
- Set up MinIO secrets
    -   ```
        kubectl create secret generic minio-creds \
        --from-literal=MINIO_ROOT_USER=admin \
        --from-literal=MINIO_ROOT_PASSWORD=admin123 \
        -n default
        ```
- `kubectl apply -f simulation/minio-env-deployment.yaml`

### Run it!
- `go run main.go`
- http://localhost:3000/logs/2025-04-22T22 (replace with date and time for your bucket)
