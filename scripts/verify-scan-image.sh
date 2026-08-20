#!/bin/sh
set -eu

image=${1:-ghcr.io/securecloud-project/securecloud-scan:v0.1.0}
case "$image" in
  ghcr.io/securecloud-project/securecloud-scan:v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "image must use the public GHCR repository and a vMAJOR.MINOR.PATCH tag" >&2; exit 1 ;;
esac

docker image inspect "$image" >/dev/null
user=$(docker image inspect --format '{{.Config.User}}' "$image")
case "$user" in
  ""|0|root|0:0|root:root) echo "image runs as root" >&2; exit 1 ;;
esac

size=$(docker image inspect --format '{{.Size}}' "$image")
max_size=$((30 * 1024 * 1024))
if [ "$size" -gt "$max_size" ]; then
  echo "image is larger than 30 MiB: $size bytes" >&2
  exit 1
fi

container="securecloud-scan-verify-$$"
cleanup() { docker rm -f "$container" >/dev/null 2>&1 || true; }
trap cleanup EXIT INT TERM
docker run -d --name "$container" -p 127.0.0.1::8080 --read-only --tmpfs /data:uid=65532,gid=65532,mode=0700 "$image" >/dev/null
port=$(docker port "$container" 8080/tcp | sed 's/.*://')

attempt=0
until curl --fail --silent --show-error "http://127.0.0.1:$port/healthz" >/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    docker logs "$container" >&2
    exit 1
  fi
  sleep 1
done

echo "verified $image: non-root user=$user size=$size healthz=ok read-only-rootfs=ok"

