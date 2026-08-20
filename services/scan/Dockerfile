# syntax=docker/dockerfile:1.7
FROM golang:1.26.4-alpine3.23 AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/scan ./cmd/server \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build --chown=65532:65532 /out/scan /scan
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
WORKDIR /data
ENV PORT=8080 \
    SCAN_DB_PATH=/data/scan.db
EXPOSE 8080
ENTRYPOINT ["/scan"]
