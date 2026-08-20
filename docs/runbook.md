# SecureCloud operations runbook

## Scan Service configuration

All configuration is read from the environment and validated during startup.
Malformed values stop the process rather than silently weakening a check.

| Variable | Default | Purpose |
|---|---:|---|
| `PORT` | `8080` | HTTP listen port |
| `SCAN_DB_PATH` | `./data/scan.db` | SQLite database file |
| `NOTIFICATION_SERVICE_URL` | empty | Notification HTTP(S) origin; empty disables delivery |
| `SCAN_NETWORK_TIMEOUT` | `5s` | Per-connection/request timeout |
| `SCAN_JOB_TIMEOUT` | `20s` | Total budget for one queued scan |
| `NOTIFICATION_TIMEOUT` | `3s` | Notification delivery timeout |
| `CERT_EXPIRY_THRESHOLD` | `720h` | Certificate-expiry warning window (30 days) |
| `SCAN_WORKERS` | `2` | Fixed worker count, range 1–32 |
| `SCAN_QUEUE_DEPTH` | `64` | In-memory queue capacity |

There are no Scan Service secrets. `NOTIFICATION_SERVICE_URL` is supplied by
the OpenChoreo endpoint dependency in deployed environments. Do not put tokens,
credentials, or literal service DNS names in the workload. The database
directory is mode `0700` and the database file is forced to `0600`.

## Local verification

```bash
cd services/scan
go test ./...
go vet ./...
docker build -t ghcr.io/securecloud-project/securecloud-scan:v0.1.0 .
../../scripts/verify-scan-image.sh \
  ghcr.io/securecloud-project/securecloud-scan:v0.1.0
```

Before publishing, authenticate with GHCR and push the semver tag. Make the
package public in GitHub, then run `docker pull` and the verification script on
a clean machine. Production deployment should replace the tag in the workload
with the verified immutable `sha256` digest.

## Queue and restart behaviour

Requests are validated before persistence. A bounded queue returns HTTP 503
when full. Workers transition scans `queued -> running -> complete|failed`.
Notification failure is logged but never changes a completed scan. On startup,
interrupted `running` records are returned to `queued` and pending records are
loaded into the queue. Because SQLite and the queue are local, keep the deployed
replica count at one unless storage and job coordination are redesigned.

## OpenChoreo deployment and rollback

```bash
kubectl apply -f platform/notification/
kubectl apply -f platform/scan/
kubectl wait --for=condition=Ready component/scan-service -n default --timeout=5m
kubectl get componentrelease,releasebinding -n default
kubectl get deployment,service,httproute,networkpolicy -A \
  -l openchoreo.dev/component=scan-service
```

Test the Scan health endpoint through the external gateway, then run the
visibility test documented in `platform/notification/SECURITY-TEST.md`. Inspect
the Scan pod environment and confirm that OpenChoreo injected
`NOTIFICATION_SERVICE_URL`; never manually set it to a cluster DNS name.

To roll back, identify the last healthy `ComponentRelease` and update the
environment's `ReleaseBinding` through the OpenChoreo portal or CLI. Confirm the
binding, generated Deployment rollout, `/readyz`, and logs before closing the
incident. Do not delete the SQLite volume during rollback.

