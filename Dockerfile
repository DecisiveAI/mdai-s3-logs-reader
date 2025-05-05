FROM --platform=$BUILDPLATFORM golang:1.23-bookworm AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /opt/mdai-s3-logs-reader

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /mdai-s3-logs-reader main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /mdai-s3-logs-reader /mdai-s3-logs-reader
CMD ["/mdai-s3-logs-reader"]
