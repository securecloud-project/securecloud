#!/bin/sh
set -eu

manifest=platform/notification/workload.yaml
if grep -Eq '^[[:space:]]*-[[:space:]]*external[[:space:]]*$' "$manifest"; then
  echo "notification workload requests external visibility" >&2
  exit 1
fi

for command in kubectl curl jq; do
  command -v "$command" >/dev/null 2>&1 || { echo "$command is required" >&2; exit 1; }
done

if [ -z "${SECURECLOUD_TOKEN:-}" ]; then
  echo "SECURECLOUD_TOKEN must contain a valid Auth Service bearer token" >&2
  exit 1
fi

notification_selector='openchoreo.dev/component=notification-service'
scan_selector='openchoreo.dev/component=scan-service'

kubectl wait deployment -A -l "$notification_selector" --for=condition=Available --timeout=2m
kubectl wait deployment -A -l "$scan_selector" --for=condition=Available --timeout=2m

namespace=$(kubectl get service -A -l "$notification_selector" -o jsonpath='{.items[0].metadata.namespace}')
service=$(kubectl get service -A -l "$notification_selector" -o jsonpath='{.items[0].metadata.name}')
if [ -z "$namespace" ] || [ -z "$service" ]; then
  echo "generated notification Service was not found" >&2
  exit 1
fi

forward_log=$(mktemp)
kubectl port-forward -n "$namespace" "service/$service" 18081:8080 >"$forward_log" 2>&1 &
forward_pid=$!
cleanup() {
  kill "$forward_pid" >/dev/null 2>&1 || true
  rm -f "$forward_log"
}
trap cleanup EXIT INT TERM

attempt=0
until curl --fail --silent "http://127.0.0.1:18081/healthz" >/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 20 ]; then
    cat "$forward_log" >&2
    exit 1
  fi
  sleep 1
done

external_notification='http://development-default.openchoreoapis.localhost:19080/notification-http/healthz'
external_status=$(curl --silent --output /dev/null --write-out '%{http_code}' --max-time 5 "$external_notification" || true)
case "$external_status" in
  2??|3??) echo "notification is externally reachable (HTTP $external_status)" >&2; exit 1 ;;
esac

injected=$(kubectl get deployment -A -l "$scan_selector" -o jsonpath='{.items[0].spec.template.spec.containers[0].env[?(@.name=="NOTIFICATION_SERVICE_URL")].value}')
if [ -z "$injected" ]; then
  echo "NOTIFICATION_SERVICE_URL was not injected into Scan" >&2
  exit 1
fi

policy_count=$(kubectl get networkpolicy -A -l "$notification_selector" -o json | jq '.items | length')
if [ "$policy_count" -lt 1 ]; then
  echo "no generated Notification NetworkPolicy found" >&2
  exit 1
fi

scan_gateway='http://development-default.openchoreoapis.localhost:19080/scan-http'
scan_id=$(curl --fail --silent --show-error -H 'Content-Type: application/json' -H "Authorization: Bearer $SECURECLOUD_TOKEN" -d '{"target":"example.com"}' "$scan_gateway/scan" | jq -er '.id')

attempt=0
status=queued
while [ "$status" = queued ] || [ "$status" = running ]; do
  attempt=$((attempt + 1))
  if [ "$attempt" -ge 40 ]; then
    echo "scan $scan_id did not finish" >&2
    exit 1
  fi
  sleep 1
  status=$(curl --fail --silent -H "Authorization: Bearer $SECURECLOUD_TOKEN" "$scan_gateway/scan/$scan_id" | jq -er '.status')
done
if [ "$status" != complete ]; then
  echo "scan $scan_id ended with status $status" >&2
  exit 1
fi

curl --fail --silent "http://127.0.0.1:18081/notifications" | jq -e --arg id "$scan_id" 'any(.[]; .scan_id == $id)' >/dev/null
echo "verified: Notification is healthy internally, absent from external ingress, policy-protected, and reachable through Scan dependency"
